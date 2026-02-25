//go:build integration || obj

package obj_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/linodego"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/helper"
	"github.com/linode/terraform-provider-linode/v3/linode/obj/tmpl"
)

var (
	testCluster string
	testRegion  string
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

	testRegion = acceptance.GetEndpointRegion(*endpoint)
}

func TestAccResourceObject_basic_cluster(t *testing.T) {
	t.Parallel()

	validateObjectUpdates := func(resourceName, key, content string) []statecheck.StateCheck {
		return append(stateCheckValidateObject(resourceName, key, content),
			statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("acl"), knownvalue.StringExact("public-read")),
			statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("content_type"), knownvalue.StringExact("text/plain")),
			statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("content_encoding"), knownvalue.StringExact("utf8")),
			statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("content_language"), knownvalue.StringExact("en")),
			statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("website_redirect"), knownvalue.StringExact("test.com")),
			statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("force_destroy"), knownvalue.StringExact("true")),
			statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("content_disposition"), knownvalue.StringExact("attachment")),
			statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("cache_control"), knownvalue.StringExact("max-age=2592000")),
			statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("metadata").AtMapKey("foo"), knownvalue.StringExact("bar")),
			statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("metadata").AtMapKey("bar"), knownvalue.StringExact("foo")),
		)
	}

	content := "testing123"
	contentUpdated := "testing456"

	contentSource := acceptance.CreateTempFile(t, "tf-test-obj-source", content)
	contentSourceUpdated := acceptance.CreateTempFile(t, "tf-test-obj-source-updated", contentUpdated)

	acceptance.RunTestWithRetries(t, 6, func(t *acceptance.WrappedT) {
		bucketName := acctest.RandomWithPrefix("tf-test")
		keyName := acctest.RandomWithPrefix("tf_test")

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             checkObjectDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.BasicWithCluster(t, bucketName, testCluster, keyName, content, contentSource.Name()),
					ConfigStateChecks: append(append(
						stateCheckValidateObject(getObjectResourceName("basic"), "test_basic", content),
						stateCheckValidateObject(getObjectResourceName("base64"), "test_base64", content)...),
						stateCheckValidateObject(getObjectResourceName("source"), "test_source", content)...,
					),
				},
				{
					Config: tmpl.Updates(t, bucketName, testRegion, keyName, contentUpdated, contentSourceUpdated.Name()),
					ConfigStateChecks: append(append(
						validateObjectUpdates(getObjectResourceName("basic"), "test_basic", contentUpdated),
						validateObjectUpdates(getObjectResourceName("base64"), "test_base64", contentUpdated)...),
						validateObjectUpdates(getObjectResourceName("source"), "test_source", contentUpdated)...,
					),
				},
			},
		})
	})
}

func TestAccResourceObject_basic(t *testing.T) {
	t.Parallel()

	validateObjectUpdates := func(resourceName, key, content string) []statecheck.StateCheck {
		return append(stateCheckValidateObject(resourceName, key, content),
			statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("acl"), knownvalue.StringExact("public-read")),
			statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("content_type"), knownvalue.StringExact("text/plain")),
			statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("content_encoding"), knownvalue.StringExact("utf8")),
			statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("content_language"), knownvalue.StringExact("en")),
			statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("website_redirect"), knownvalue.StringExact("test.com")),
			statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("force_destroy"), knownvalue.StringExact("true")),
			statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("content_disposition"), knownvalue.StringExact("attachment")),
			statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("cache_control"), knownvalue.StringExact("max-age=2592000")),
			statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("metadata").AtMapKey("foo"), knownvalue.StringExact("bar")),
			statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("metadata").AtMapKey("bar"), knownvalue.StringExact("foo")),
		)
	}

	content := "testing123"
	contentUpdated := "testing456"

	contentSource := acceptance.CreateTempFile(t, "tf-test-obj-source", content)
	contentSourceUpdated := acceptance.CreateTempFile(t, "tf-test-obj-source-updated", contentUpdated)

	acceptance.RunTestWithRetries(t, 6, func(t *acceptance.WrappedT) {
		bucketName := acctest.RandomWithPrefix("tf-test")
		keyName := acctest.RandomWithPrefix("tf_test")

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             checkObjectDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.Basic(t, bucketName, testRegion, keyName, content, contentSource.Name()),
					ConfigStateChecks: append(append(
						stateCheckValidateObject(getObjectResourceName("basic"), "test_basic", content),
						stateCheckValidateObject(getObjectResourceName("base64"), "test_base64", content)...),
						stateCheckValidateObject(getObjectResourceName("source"), "test_source", content)...,
					),
				},
				{
					Config: tmpl.Updates(t, bucketName, testRegion, keyName, contentUpdated, contentSourceUpdated.Name()),
					ConfigStateChecks: append(append(
						validateObjectUpdates(getObjectResourceName("basic"), "test_basic", contentUpdated),
						validateObjectUpdates(getObjectResourceName("base64"), "test_base64", contentUpdated)...),
						validateObjectUpdates(getObjectResourceName("source"), "test_source", contentUpdated)...,
					),
				},
			},
		})
	})
}

func TestAccResourceObject_credsConfiged(t *testing.T) {
	t.Parallel()

	content := "test_creds_configed"

	acceptance.RunTestWithRetries(t, 6, func(t *acceptance.WrappedT) {
		bucketName := acctest.RandomWithPrefix("tf-test")
		keyName := acctest.RandomWithPrefix("tf_test")

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             checkObjectDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.CredsConfiged(t, bucketName, testRegion, keyName, content),
					ConfigStateChecks: stateCheckValidateObject(getObjectResourceName("creds_configed"), "test_creds_configed", content),
				},
			},
		})
	})
}

func TestAccResourceObject_tempKeys(t *testing.T) {
	t.Parallel()

	content := "test_temp_keys"

	acceptance.RunTestWithRetries(t, 6, func(t *acceptance.WrappedT) {
		bucketName := acctest.RandomWithPrefix("tf-test")
		keyName := acctest.RandomWithPrefix("tf_test")

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             checkObjectDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.TempKeys(t, bucketName, testRegion, keyName, content),
					ConfigStateChecks: stateCheckValidateObject(getObjectResourceName("temp_keys"), "test_temp_keys", content),
				},
			},
		})
	})
}

func getObject(ctx context.Context, rs *terraform.ResourceState) (*s3.GetObjectOutput, error) {
	bucket := rs.Primary.Attributes["bucket"]
	key := rs.Primary.Attributes["key"]
	etag := rs.Primary.Attributes["etag"]
	accessKey := rs.Primary.Attributes["access_key"]
	secretKey := rs.Primary.Attributes["secret_key"]
	endpoint := rs.Primary.Attributes["endpoint"]

	if accessKey == "" || secretKey == "" {
		client, err := acceptance.GetTestClient()
		if err != nil {
			return nil, fmt.Errorf("Error getting client: %s", err)
		}

		createOpts := linodego.ObjectStorageKeyCreateOptions{
			Label: fmt.Sprintf("temp_%s_%v", bucket, time.Now().Unix()),
			BucketAccess: &[]linodego.ObjectStorageKeyBucketAccess{{
				BucketName:  bucket,
				Region:      rs.Primary.Attributes["region"],
				Permissions: "read_write",
			}},
		}

		key, err := client.CreateObjectStorageKey(ctx, createOpts)
		if err != nil {
			return nil, err
		}

		accessKey = key.AccessKey
		secretKey = key.SecretKey

		defer func() {
			if err := client.DeleteObjectStorageKey(ctx, key.ID); err != nil {
				log.Printf("[WARN] Failed to clean up temporary object storage keys: %s\n", err)
			}
		}()
	}

	s3client, err := helper.S3Connection(ctx, endpoint, accessKey, secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get create s3 client: %w", err)
	}

	return s3client.GetObject(
		ctx,
		&s3.GetObjectInput{
			Bucket:  &bucket,
			Key:     &key,
			IfMatch: &etag,
		},
	)
}

func checkObjectDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_object_storage_object" {
			continue
		}

		key := rs.Primary.Attributes["key"]

		if _, err := getObject(context.Background(), rs); err == nil {
			return fmt.Errorf("object with %s Key still exists", key)
		}
	}

	return nil
}

func getObjectResourceName(name string) string {
	return fmt.Sprintf("linode_object_storage_object.%s", name)
}

// stateCheckObjectExists is the statecheck.StateCheck equivalent of checkObjectExists.
// It finds the resource in the tfjson state, extracts attributes, creates temporary
// S3 credentials if needed, and stores the GetObjectOutput in the shared pointer.
func stateCheckObjectExists(resourceName string, obj *s3.GetObjectOutput) statecheck.StateCheck {
	return acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		var bucket, key, etag, accessKey, secretKey, endpoint, region string
		found := false
		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Address != resourceName {
				continue
			}
			found = true
			if v, ok := rc.AttributeValues["bucket"]; ok && v != nil {
				if s, ok := v.(string); ok {
					bucket = s
				} else {
					resp.Error = fmt.Errorf("bucket attribute is not a string")
					return
				}
			}
			if v, ok := rc.AttributeValues["key"]; ok && v != nil {
				if s, ok := v.(string); ok {
					key = s
				} else {
					resp.Error = fmt.Errorf("key attribute is not a string")
					return
				}
			}
			if v, ok := rc.AttributeValues["etag"]; ok && v != nil {
				if s, ok := v.(string); ok {
					etag = s
				} else {
					resp.Error = fmt.Errorf("etag attribute is not a string")
					return
				}
			}
			if v, ok := rc.AttributeValues["access_key"]; ok && v != nil {
				if s, ok := v.(string); ok {
					accessKey = s
				} else {
					resp.Error = fmt.Errorf("access_key attribute is not a string")
					return
				}
			}
			if v, ok := rc.AttributeValues["secret_key"]; ok && v != nil {
				if s, ok := v.(string); ok {
					secretKey = s
				} else {
					resp.Error = fmt.Errorf("secret_key attribute is not a string")
					return
				}
			}
			if v, ok := rc.AttributeValues["endpoint"]; ok && v != nil {
				if s, ok := v.(string); ok {
					endpoint = s
				} else {
					resp.Error = fmt.Errorf("endpoint attribute is not a string")
					return
				}
			}
			if v, ok := rc.AttributeValues["region"]; ok && v != nil {
				if s, ok := v.(string); ok {
					region = s
				} else {
					resp.Error = fmt.Errorf("region attribute is not a string")
					return
				}
			}
			break
		}
		if !found {
			resp.Error = fmt.Errorf("could not find resource %s in root module", resourceName)
			return
		}

		if accessKey == "" || secretKey == "" {
			client, err := acceptance.GetTestClient()
			if err != nil {
				resp.Error = fmt.Errorf("Error getting client: %s", err)
				return
			}

			createOpts := linodego.ObjectStorageKeyCreateOptions{
				Label: fmt.Sprintf("temp_%s_%v", bucket, time.Now().Unix()),
				BucketAccess: &[]linodego.ObjectStorageKeyBucketAccess{{
					BucketName:  bucket,
					Region:      region,
					Permissions: "read_write",
				}},
			}

			tempKey, err := client.CreateObjectStorageKey(ctx, createOpts)
			if err != nil {
				resp.Error = err
				return
			}

			accessKey = tempKey.AccessKey
			secretKey = tempKey.SecretKey

			defer func() {
				if err := client.DeleteObjectStorageKey(ctx, tempKey.ID); err != nil {
					log.Printf("[WARN] Failed to clean up temporary object storage keys: %s\n", err)
				}
			}()
		}

		s3client, err := helper.S3Connection(ctx, endpoint, accessKey, secretKey)
		if err != nil {
			resp.Error = fmt.Errorf("failed to create s3 client: %w", err)
			return
		}

		out, err := s3client.GetObject(ctx, &s3.GetObjectInput{
			Bucket:  &bucket,
			Key:     &key,
			IfMatch: &etag,
		})
		if err != nil {
			resp.Error = fmt.Errorf("failed to get Bucket (%s) Object (%s): %s", bucket, key, err)
			return
		}

		*obj = *out
	})
}

// stateCheckObjectBodyContains is the statecheck.StateCheck equivalent of checkObjectBodyContains.
// It reads the object body from the shared *s3.GetObjectOutput pointer (populated by
// stateCheckObjectExists which runs first in the ConfigStateChecks slice) and validates its content.
func stateCheckObjectBodyContains(obj *s3.GetObjectOutput, expected string) statecheck.StateCheck {
	return acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		body, err := io.ReadAll(obj.Body)
		if err != nil {
			resp.Error = fmt.Errorf("failed to read body: %s", err)
			return
		}
		obj.Body.Close()

		if got := string(body); got != expected {
			resp.Error = fmt.Errorf("expected body to be %q; got %q", expected, got)
		}
	})
}

// stateCheckValidateObject is the statecheck equivalent of validateObject.
// It returns a []statecheck.StateCheck slice combining the existence check, body
// content check, and key attribute check. Order matters: stateCheckObjectExists
// must come first because it populates the object pointer that
// stateCheckObjectBodyContains reads.
func stateCheckValidateObject(resourceName, key, content string) []statecheck.StateCheck {
	var object s3.GetObjectOutput

	return []statecheck.StateCheck{
		stateCheckObjectExists(resourceName, &object),
		stateCheckObjectBodyContains(&object, content),
		statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("key"), knownvalue.StringExact(key)),
	}
}
