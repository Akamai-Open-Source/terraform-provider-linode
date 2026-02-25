//go:build integration || objbucket

package objbucket_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/linodego"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/helper"
	"github.com/linode/terraform-provider-linode/v3/linode/objbucket"
	"github.com/linode/terraform-provider-linode/v3/linode/objbucket/tmpl"
)

const (
	objAccessKeyEnvVar = "LINODE_OBJ_ACCESS_KEY"
	objSecretKeyEnvVar = "LINODE_OBJ_SECRET_KEY"
)

var (
	testCluster      string
	testRegion       string
	testEndpointType string
	testEndpointURL  string
)

func init() {
	endpoint, err := acceptance.GetRandomObjectStorageEndpoint()
	if err != nil {
		log.Fatal(err)
	}

	testCluster, err = acceptance.GetEndpointCluster(*endpoint)
	if err != nil {
		log.Fatal(err)
	}

	testRegion = endpoint.Region
	testEndpointType = string(endpoint.EndpointType)
	testEndpointURL = *endpoint.S3Endpoint
}

func init() {
	resource.AddTestSweepers("linode_object_storage_bucket", &resource.Sweeper{
		Name: "linode_object_storage_bucket",
		F:    sweep,
	})
}

func generateTestCert(domain string) (certificate, privateKey string, err error) {
	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate key: %s", err)
	}
	keyUsage := x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment

	validFrom := time.Now()
	validUntil := validFrom.Add(time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate serial number: %s", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Linode"},
		},
		NotBefore:             validFrom,
		NotAfter:              validUntil,
		KeyUsage:              keyUsage,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{domain},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, priv.Public(), priv)
	if err != nil {
		return "", "", fmt.Errorf("failed to create certificate: %s", err)
	}
	certBuffer := new(bytes.Buffer)
	if err := pem.Encode(certBuffer, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return "", "", fmt.Errorf("failed to encode certificate to PEM: %s", err)
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal private key: %s", err)
	}
	keyBuffer := new(bytes.Buffer)
	if err := pem.Encode(keyBuffer, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		return "", "", fmt.Errorf("failed to encode private key to PEM: %s", err)
	}

	return string(certBuffer.Bytes()), string(keyBuffer.Bytes()), nil
}

func sweep(prefix string) error {
	client, err := acceptance.GetTestClient()
	if err != nil {
		return fmt.Errorf("Error getting client: %s", err)
	}

	listOpts := acceptance.SweeperListOptions(prefix, "label")
	objectStorageBuckets, err := client.ListObjectStorageBuckets(context.Background(), listOpts)
	if err != nil {
		return fmt.Errorf("Error getting object_storage_buckets: %s", err)
	}

	accessKey, accessKeyOk := os.LookupEnv(objAccessKeyEnvVar)
	secretKey, secretKeyOk := os.LookupEnv(objSecretKeyEnvVar)
	haveBucketAccess := accessKeyOk && secretKeyOk

	for _, objectStorageBucket := range objectStorageBuckets {
		if !acceptance.ShouldSweep(prefix, objectStorageBucket.Label) {
			continue
		}
		bucket := objectStorageBucket.Label
		s3client, err := helper.S3Connection(
			context.Background(),
			objectStorageBucket.S3Endpoint,
			accessKey,
			secretKey,
		)
		if err != nil {
			tflog.Error(context.Background(), fmt.Sprintf("failed to create s3 client: %v", err))
		}

		helper.PurgeAllObjects(context.Background(), bucket, s3client, true, true)

		_, err = s3client.DeleteBucket(
			context.Background(),
			&s3.DeleteBucketInput{
				Bucket: aws.String(bucket),
			},
		)
		if err != nil {
			if apiErr, ok := err.(*linodego.Error); ok && !haveBucketAccess && strings.HasPrefix(
				apiErr.Message, fmt.Sprintf("Bucket %s is not empty", bucket)) {
				tflog.Warn(
					context.Background(),
					fmt.Sprintf(
						"will not delete Object Storage Bucket (%s) as it needs to be emptied; "+
							"specify %q and %q env variables for bucket access",
						bucket, objAccessKeyEnvVar, objSecretKeyEnvVar,
					),
				)
				continue
			}
			return fmt.Errorf("Error destroying %s during sweep: %s", bucket, err)
		}
	}

	return nil
}

func TestSmokeTests_objbucket(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{"TestAccResourceBucket_basic_legacy_smoke", TestAccResourceBucket_basic_legacy_smoke},
		{"TestAccResourceBucket_basic_smoke", TestAccResourceBucket_basic_smoke},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestAccResourceBucket_basic_legacy_smoke(t *testing.T) {
	t.Parallel()

	acceptance.RunTestWithRetries(t, 5, func(t *acceptance.WrappedT) {
		resName := "linode_object_storage_bucket.foobar"
		objectStorageBucketName := acctest.RandomWithPrefix("tf-test")

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             checkBucketDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.BasicLegacy(t, objectStorageBucketName, testCluster),
					ConfigStateChecks: []statecheck.StateCheck{
						stateCheckBucketExists(),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(objectStorageBucketName)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("hostname"), knownvalue.NotNull()),
					},
				},
				{
					ResourceName:      resName,
					ImportState:       true,
					ImportStateVerify: true,
				},
			},
		})
	})
}

func TestAccResourceBucket_endpoint_type(t *testing.T) {
	t.Parallel()

	acceptance.RunTestWithRetries(t, 5, func(t *acceptance.WrappedT) {
		resName := "linode_object_storage_bucket.foobar"
		objectStorageBucketName := acctest.RandomWithPrefix("tf-test")

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             checkBucketDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.EndpointType(t, objectStorageBucketName, testRegion, testEndpointType),
					ConfigStateChecks: []statecheck.StateCheck{
						stateCheckBucketExists(),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(objectStorageBucketName)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("hostname"), knownvalue.NotNull()),
					},
				},
				{
					ResourceName:      resName,
					ImportState:       true,
					ImportStateVerify: true,
				},
			},
		})
	})
}

func TestAccResourceBucket_endpoint_url(t *testing.T) {
	t.Parallel()

	acceptance.RunTestWithRetries(t, 5, func(t *acceptance.WrappedT) {
		resName := "linode_object_storage_bucket.foobar"
		objectStorageBucketName := acctest.RandomWithPrefix("tf-test")

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             checkBucketDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.EndpointURL(t, objectStorageBucketName, testRegion, testEndpointURL),
					ConfigStateChecks: []statecheck.StateCheck{
						stateCheckBucketExists(),
						statecheck.ExpectKnownValue(
							resName,
							tfjsonpath.New("label"),
							knownvalue.StringExact(objectStorageBucketName),
						),
						statecheck.ExpectKnownValue(
							resName,
							tfjsonpath.New("hostname"),
							knownvalue.NotNull(),
						),
					},
				},
				{
					ResourceName:      resName,
					ImportState:       true,
					ImportStateVerify: true,
				},
			},
		})
	})
}

func TestAccResourceBucket_basic_smoke(t *testing.T) {
	t.Parallel()

	acceptance.RunTestWithRetries(t, 5, func(t *acceptance.WrappedT) {
		resName := "linode_object_storage_bucket.foobar"
		objectStorageBucketName := acctest.RandomWithPrefix("tf-test")

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             checkBucketDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.Basic(t, objectStorageBucketName, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						stateCheckBucketExists(),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(objectStorageBucketName)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("hostname"), knownvalue.NotNull()),
					},
				},
				{
					ResourceName:      resName,
					ImportState:       true,
					ImportStateVerify: true,
				},
			},
		})
	})
}

func TestAccResourceBucket_access(t *testing.T) {
	t.Parallel()

	acceptance.RunTestWithRetries(t, 5, func(t *acceptance.WrappedT) {
		resName := "linode_object_storage_bucket.foobar"
		objectStorageBucketName := acctest.RandomWithPrefix("tf-test")

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             checkBucketDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.Access(t, objectStorageBucketName, testRegion, "public-read", true),
					ConfigStateChecks: []statecheck.StateCheck{
						stateCheckBucketExists(),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(objectStorageBucketName)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("acl"), knownvalue.StringExact("public-read")),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("cors_enabled"), knownvalue.StringExact("true")),
					},
				},
				{
					Config: tmpl.Access(t, objectStorageBucketName, testRegion, "private", false),
					ConfigStateChecks: []statecheck.StateCheck{
						stateCheckBucketExists(),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(objectStorageBucketName)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("acl"), knownvalue.StringExact("private")),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("cors_enabled"), knownvalue.StringExact("false")),
					},
				},
			},
		})
	})
}

func TestAccResourceBucket_versioning(t *testing.T) {
	t.Parallel()

	acceptance.RunTestWithRetries(t, 5, func(t *acceptance.WrappedT) {
		resName := "linode_object_storage_bucket.foobar"
		objectStorageBucketName := acctest.RandomWithPrefix("tf-test")
		objectStorageKeyName := acctest.RandomWithPrefix("tf-test")

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             checkBucketDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.Versioning(t, objectStorageBucketName, testRegion, objectStorageKeyName, true),
					ConfigStateChecks: []statecheck.StateCheck{
						stateCheckBucketExists(),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(objectStorageBucketName)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("versioning"), knownvalue.StringExact("true")),
					},
				},
				{
					Config: tmpl.Versioning(t, objectStorageBucketName, testRegion, objectStorageKeyName, false),
					ConfigStateChecks: []statecheck.StateCheck{
						stateCheckBucketExists(),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(objectStorageBucketName)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("versioning"), knownvalue.StringExact("false")),
					},
				},
			},
		})
	})
}

func TestAccResourceBucket_lifecycle(t *testing.T) {
	t.Parallel()

	acceptance.RunTestWithRetries(t, 5, func(t *acceptance.WrappedT) {
		resName := "linode_object_storage_bucket.foobar"
		objectStorageBucketName := acctest.RandomWithPrefix("tf-test")
		objectStorageKeyName := acctest.RandomWithPrefix("tf-test")

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             checkBucketDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.LifeCycle(t, objectStorageBucketName, testRegion, objectStorageKeyName),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(objectStorageBucketName)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("cluster"), knownvalue.StringExact(testCluster)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("lifecycle_rule"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("lifecycle_rule").AtSliceIndex(0).AtMapKey("id"), knownvalue.StringExact("test-rule")),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("lifecycle_rule").AtSliceIndex(0).AtMapKey("prefix"), knownvalue.StringExact("tf")),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("lifecycle_rule").AtSliceIndex(0).AtMapKey("enabled"), knownvalue.StringExact("true")),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("lifecycle_rule").AtSliceIndex(0).AtMapKey("expiration"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("lifecycle_rule").AtSliceIndex(0).AtMapKey("abort_incomplete_multipart_upload_days"), knownvalue.StringExact("5")),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("lifecycle_rule").AtSliceIndex(0).AtMapKey("expiration").AtSliceIndex(0).AtMapKey("date"), knownvalue.NotNull()),
					},
				},
				{
					Config: tmpl.LifeCycleUpdates(t, objectStorageBucketName, testRegion, objectStorageKeyName),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(objectStorageBucketName)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("cluster"), knownvalue.StringExact(testCluster)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("lifecycle_rule"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("lifecycle_rule").AtSliceIndex(0).AtMapKey("id"), knownvalue.StringExact("test-rule-update")),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("lifecycle_rule").AtSliceIndex(0).AtMapKey("prefix"), knownvalue.StringExact("tf-update")),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("lifecycle_rule").AtSliceIndex(0).AtMapKey("enabled"), knownvalue.StringExact("false")),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("lifecycle_rule").AtSliceIndex(0).AtMapKey("abort_incomplete_multipart_upload_days"), knownvalue.StringExact("42")),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("lifecycle_rule").AtSliceIndex(0).AtMapKey("expiration"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("lifecycle_rule").AtSliceIndex(0).AtMapKey("expiration").AtSliceIndex(0).AtMapKey("days"), knownvalue.StringExact("37")),
					},
				},
				{
					Config: tmpl.LifeCycleRemoved(t, objectStorageBucketName, testRegion, objectStorageKeyName),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(objectStorageBucketName)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("cluster"), knownvalue.StringExact(testCluster)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("lifecycle_rule"), knownvalue.ListSizeExact(0)),
					},
				},
			},
		})
	})
}

func TestAccResourceBucket_lifecycleNoID(t *testing.T) {
	t.Parallel()

	resName := "linode_object_storage_bucket.foobar"
	objectStorageBucketName := acctest.RandomWithPrefix("tf-test")
	objectStorageKeyName := acctest.RandomWithPrefix("tf-test")

	acceptance.RunTestWithRetries(t, 5, func(t *acceptance.WrappedT) {
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             checkBucketDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.LifeCycleNoID(t, objectStorageBucketName, testRegion, objectStorageKeyName),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(objectStorageBucketName)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("cluster"), knownvalue.StringExact(testCluster)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("lifecycle_rule"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("lifecycle_rule").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("lifecycle_rule").AtSliceIndex(0).AtMapKey("prefix"), knownvalue.StringExact("tf")),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("lifecycle_rule").AtSliceIndex(0).AtMapKey("enabled"), knownvalue.StringExact("true")),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("lifecycle_rule").AtSliceIndex(0).AtMapKey("expiration"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("lifecycle_rule").AtSliceIndex(0).AtMapKey("abort_incomplete_multipart_upload_days"), knownvalue.StringExact("5")),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("lifecycle_rule").AtSliceIndex(0).AtMapKey("expiration").AtSliceIndex(0).AtMapKey("date"), knownvalue.NotNull()),
					},
				},
			},
		})
	})
}

func TestAccResourceBucket_cert(t *testing.T) {
	t.Parallel()

	resName := "linode_object_storage_bucket.foobar"
	objectStorageBucketName := acctest.RandomWithPrefix("tf-test") + ".io"
	cert, key, err := generateTestCert(objectStorageBucketName)
	if err != nil {
		t.Fatal(err)
	}

	invalidCert, invalidKey, err := generateTestCert("bogusdomain.com")
	if err != nil {
		t.Fatal(err)
	}

	otherCert, otherKey, err := generateTestCert(objectStorageBucketName)
	if err != nil {
		t.Fatal(err)
	}

	acceptance.RunTestWithRetries(t, 5, func(t *acceptance.WrappedT) {
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             checkBucketDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.Cert(t, objectStorageBucketName, testRegion, cert, key),
					ConfigStateChecks: []statecheck.StateCheck{
						stateCheckBucketExists(),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(objectStorageBucketName)),
					},
				},
				{
					Config:      tmpl.Cert(t, objectStorageBucketName, testRegion, invalidCert, invalidKey),
					ExpectError: regexp.MustCompile("failed to upload new bucket cert"),
				},
				{
					Config: tmpl.Cert(t, objectStorageBucketName, testRegion, otherCert, otherKey),
					ConfigStateChecks: []statecheck.StateCheck{
						stateCheckBucketExists(),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(objectStorageBucketName)),
					},
				},
				{
					Config: tmpl.Basic(t, objectStorageBucketName, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						stateCheckBucketExists(),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(objectStorageBucketName)),
					},
				},
			},
		})
	})
}

func TestAccResourceBucket_dataSource(t *testing.T) {
	t.Parallel()

	resName := "linode_object_storage_bucket.foobar"
	objectStorageBucketName := acctest.RandomWithPrefix("tf-test")

	acceptance.RunTestWithRetries(t, 5, func(t *acceptance.WrappedT) {
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             checkBucketDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.ClusterDataBasic(t, objectStorageBucketName, testCluster),
					ConfigStateChecks: []statecheck.StateCheck{
						stateCheckBucketExists(),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(objectStorageBucketName)),
					},
				},
				{
					ResourceName:      resName,
					ImportState:       true,
					ImportStateVerify: true,
				},
			},
		})
	})
}

func TestAccResourceBucket_update(t *testing.T) {
	t.Parallel()

	objectStorageBucketName := acctest.RandomWithPrefix("tf-test")
	resName := "linode_object_storage_bucket.foobar"

	acceptance.RunTestWithRetries(t, 5, func(t *acceptance.WrappedT) {
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             checkBucketDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.Basic(t, objectStorageBucketName, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						stateCheckBucketExists(),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(objectStorageBucketName)),
					},
				},
				{
					Config: tmpl.Updates(t, objectStorageBucketName, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						stateCheckBucketExists(),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(fmt.Sprintf("%s-renamed", objectStorageBucketName))),
					},
				},
			},
		})
	})
}

func TestAccResourceBucket_credsConfiged(t *testing.T) {
	t.Parallel()

	resName := "linode_object_storage_bucket.foobar"
	objectStorageBucketName := acctest.RandomWithPrefix("tf-test")
	objectStorageKeyName := acctest.RandomWithPrefix("tf-test")

	acceptance.RunTestWithRetries(t, 5, func(t *acceptance.WrappedT) {
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             checkBucketDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.CredsConfiged(t, objectStorageBucketName, testRegion, objectStorageKeyName),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(objectStorageBucketName)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("cluster"), knownvalue.StringExact(testCluster)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("lifecycle_rule"), knownvalue.ListSizeExact(1)),
					},
				},
			},
		})
	})
}

func TestAccResourceBucket_tempKeys(t *testing.T) {
	t.Parallel()

	resName := "linode_object_storage_bucket.foobar"
	objectStorageBucketName := acctest.RandomWithPrefix("tf-test")
	objectStorageKeyName := acctest.RandomWithPrefix("tf-test")

	acceptance.RunTestWithRetries(t, 5, func(t *acceptance.WrappedT) {
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             checkBucketDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.TempKeys(t, objectStorageBucketName, testRegion, objectStorageKeyName),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(objectStorageBucketName)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("cluster"), knownvalue.StringExact(testCluster)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("lifecycle_rule"), knownvalue.ListSizeExact(1)),
					},
				},
			},
		})
	})
}

func TestAccResourceBucket_forceDelete(t *testing.T) {
	t.Parallel()

	resName := "linode_object_storage_bucket.foobar"
	objectStorageBucketName := acctest.RandomWithPrefix("tf-test")
	objectStorageKeyName := acctest.RandomWithPrefix("tf-test")

	acceptance.RunTestWithRetries(t, 5, func(t *acceptance.WrappedT) {
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             checkBucketDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.ForceDelete(t, objectStorageBucketName, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(objectStorageBucketName)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("cluster"), knownvalue.StringExact(testCluster)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					},
				},
				{
					PreConfig: func() {
						client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client
						createOpts := linodego.ObjectStorageKeyCreateOptions{
							Label: fmt.Sprintf("temp_%s_%v", objectStorageBucketName, time.Now().Unix()),
							BucketAccess: &[]linodego.ObjectStorageKeyBucketAccess{{
								BucketName:  objectStorageBucketName,
								Region:      testRegion,
								Permissions: "read_write",
							}},
						}

						keys, err := client.CreateObjectStorageKey(context.Background(), createOpts)
						if err != nil {
							t.Errorf("error creating obj keys in PreConfig func: %v", err)
						}
						defer client.DeleteObjectStorageKey(context.Background(), keys.ID)

						bucket, err := client.GetObjectStorageBucket(context.Background(), testRegion, objectStorageBucketName)
						if err != nil {
							t.Errorf("error getting obj bucket in PreConfig func: %v", err)
						}

						s3client, err := helper.S3Connection(context.Background(), bucket.S3Endpoint, keys.AccessKey, keys.SecretKey)
						if err != nil {
							t.Errorf("error connecting s3 in PreConfig func: %v", err)
						}

						contentBytes := []byte("delete test")
						body := *s3manager.ReadSeekCloser(bytes.NewReader(contentBytes))
						putInput := &s3.PutObjectInput{
							Bucket: &objectStorageBucketName,
							Key:    &objectStorageKeyName,
							Body:   &body,
						}
						s3client.PutObject(context.Background(), putInput)
					},
					Config: tmpl.ForceDelete_Empty(t),
					ConfigStateChecks: []statecheck.StateCheck{
						stateCheckBucketDestroy(),
					},
				},
			},
		})
	})
}

func TestAccResourceBucket_invalid_region(t *testing.T) {
	t.Parallel()

	label := "tf-acc-bucket-invalid-region-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	invalidRegion := "us-mia-1"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkBucketDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, label, invalidRegion),
				ExpectError: regexp.MustCompile(
					regexp.QuoteMeta(fmt.Sprintf("Region '%s' is not valid for Object Storage", invalidRegion)),
				),
			},
		},
	})
}

func stateCheckBucketExists() statecheck.StateCheck {
	return acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Type != "linode_object_storage_objbucket" {
				continue
			}

			idVal, ok := rc.AttributeValues["id"]
			if !ok {
				resp.Error = fmt.Errorf("No ID is set")
				return
			}

			idStr, ok := idVal.(string)
			if !ok {
				resp.Error = fmt.Errorf("id is not a string")
				return
			}

			cluster, label, err := objbucket.DecodeBucketID(ctx, idStr, &schema.ResourceData{})
			if err != nil {
				resp.Error = fmt.Errorf("Error parsing %s, %s", idStr, err)
				return
			}

			_, err = client.GetObjectStorageBucket(ctx, cluster, label)
			if err != nil {
				labelVal, _ := rc.AttributeValues["label"]
				resp.Error = fmt.Errorf("Error retrieving state of ObjectStorageBucket %s: %s", labelVal, err)
				return
			}
		}
	})
}

func stateCheckBucketDestroy() statecheck.StateCheck {
	return acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Type != "linode_object_storage_bucket" {
				continue
			}

			idVal, ok := rc.AttributeValues["id"]
			if !ok {
				continue
			}

			idStr, ok := idVal.(string)
			if !ok {
				resp.Error = fmt.Errorf("id is not a string")
				return
			}
			cluster, label, err := objbucket.DecodeBucketID(ctx, idStr, &schema.ResourceData{})
			if err != nil {
				resp.Error = fmt.Errorf("Error parsing %s", idStr)
				return
			}
			if label == "" {
				resp.Error = fmt.Errorf("Would have considered %s as %s", idStr, label)
				return
			}

			_, err = client.GetObjectStorageBucket(ctx, cluster, label)
			if err == nil {
				resp.Error = fmt.Errorf("Linode ObjectStorageBucket with id %s still exists", idStr)
				return
			}

			if apiErr, ok := err.(*linodego.Error); ok && apiErr.Code != 404 && apiErr.Code != 500 {
				resp.Error = fmt.Errorf("Error requesting Linode ObjectStorageBucket with id %s: %s", idStr, err)
				return
			}
		}
	})
}

func checkBucketDestroy(s *terraform.State) error {
	client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_object_storage_bucket" {
			continue
		}

		id := rs.Primary.ID
		cluster, label, err := objbucket.DecodeBucketID(context.Background(), id, &schema.ResourceData{})
		if err != nil {
			return fmt.Errorf("Error parsing %s", id)
		}
		if label == "" {
			return fmt.Errorf("Would have considered %s as %s", id, label)
		}

		_, err = client.GetObjectStorageBucket(context.Background(), cluster, label)

		if err == nil {
			return fmt.Errorf("Linode ObjectStorageBucket with id %s still exists", id)
		}

		if apiErr, ok := err.(*linodego.Error); ok && apiErr.Code != 404 && apiErr.Code != 500 {
			return fmt.Errorf("Error requesting Linode ObjectStorageBucket with id %s: %s", id, err)
		}
	}

	return nil
}
