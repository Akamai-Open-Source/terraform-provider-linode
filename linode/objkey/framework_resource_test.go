//go:build integration || objkey

package objkey_test

import (
	"context"
	"fmt"
	"log"
	"maps"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/linodego"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/helper"
	"github.com/linode/terraform-provider-linode/v3/linode/objkey/tmpl"
)

var (
	testCluster string
	testRegion  string
)

func init() {
	resource.AddTestSweepers("linode_object_storage_key", &resource.Sweeper{
		Name: "linode_object_storage_key",
		F:    sweep,
	})

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

func sweep(prefix string) error {
	client, err := acceptance.GetTestClient()
	if err != nil {
		return fmt.Errorf("Error getting client: %s", err)
	}

	objectStorageKeys, err := client.ListObjectStorageKeys(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("Error getting object storage keys: %s", err)
	}
	for _, objectStorageKey := range objectStorageKeys {
		if !acceptance.ShouldSweep(prefix, objectStorageKey.Label) || !strings.HasPrefix(objectStorageKey.Label, prefix) {
			continue
		}
		err := client.DeleteObjectStorageKey(context.Background(), objectStorageKey.ID)
		if err != nil {
			return fmt.Errorf("Error destroying %s during sweep: %s", objectStorageKey.Label, err)
		}
	}

	return nil
}

func TestAccResourceObjectKey_basic(t *testing.T) {
	t.Parallel()

	resName := "linode_object_storage_key.foobar"
	objectStorageKeyLabel := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkObjectKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, objectStorageKeyLabel),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckObjectKeyExists(),
					stateCheckObjectKeySecretAccessible(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(objectStorageKeyLabel)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("access_key"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("secret_key"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("limited"), knownvalue.Bool(false)),
				},
			},
		},
	})
}

func TestAccResourceObjectKey_all_regions(t *testing.T) {
	t.Parallel()

	resName := "linode_object_storage_key.foobar"
	objectStorageKeyLabel := acctest.RandomWithPrefix("tf_test")
	client := acceptance.GetFrameworkTestClient(t, []helper.HTTPClientModifier{})

	endpoints, err := client.ListObjectStorageEndpoints(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// Extract unique regions from endpoints
	regionSet := make(helper.StringSet)
	for _, endpoint := range endpoints {
		regionSet[endpoint.Region] = helper.ExistsInSet
	}

	regions := slices.Collect(maps.Keys(regionSet))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkObjectKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.AllRegions(t, objectStorageKeyLabel, regions),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("regions"),
						knownvalue.SetSizeExact(len(regions)),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("access_key"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("secret_key"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccResourceObjectKey_limited_cluster(t *testing.T) {
	t.Parallel()

	resName := "linode_object_storage_key.foobar"
	objectStorageKeyLabel := acctest.RandomWithPrefix("tf-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkObjectKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.ClusterLimited(t, objectStorageKeyLabel, testCluster),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckObjectKeyExists(),
					stateCheckObjectKeySecretAccessible(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(fmt.Sprintf("%s_key", objectStorageKeyLabel))),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("access_key"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("secret_key"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("limited"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("bucket_access"), knownvalue.SetSizeExact(2)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("bucket_access").AtSliceIndex(0).AtMapKey("bucket_name"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("bucket_access").AtSliceIndex(1).AtMapKey("bucket_name"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("bucket_access").AtSliceIndex(0).AtMapKey("cluster"), knownvalue.StringExact(testCluster)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("bucket_access").AtSliceIndex(1).AtMapKey("cluster"), knownvalue.StringExact(testCluster)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("bucket_access").AtSliceIndex(0).AtMapKey("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("bucket_access").AtSliceIndex(1).AtMapKey("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("bucket_access").AtSliceIndex(0).AtMapKey("permissions"), knownvalue.StringExact("read_only")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("bucket_access").AtSliceIndex(1).AtMapKey("permissions"), knownvalue.StringExact("read_write")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("regions"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("regions").AtSliceIndex(0), knownvalue.StringExact(testRegion)),
				},
			},
		},
	})
}

func TestAccResourceObjectKey_limited(t *testing.T) {
	t.Parallel()

	resName := "linode_object_storage_key.foobar"
	objectStorageKeyLabel := acctest.RandomWithPrefix("tf-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkObjectKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Limited(t, objectStorageKeyLabel, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckObjectKeyExists(),
					stateCheckObjectKeySecretAccessible(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(fmt.Sprintf("%s_key", objectStorageKeyLabel))),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("access_key"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("secret_key"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("limited"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("bucket_access"), knownvalue.SetSizeExact(2)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("bucket_access").AtSliceIndex(0).AtMapKey("bucket_name"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("bucket_access").AtSliceIndex(1).AtMapKey("bucket_name"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("bucket_access").AtSliceIndex(0).AtMapKey("cluster"), knownvalue.StringExact(testCluster)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("bucket_access").AtSliceIndex(1).AtMapKey("cluster"), knownvalue.StringExact(testCluster)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("bucket_access").AtSliceIndex(0).AtMapKey("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("bucket_access").AtSliceIndex(1).AtMapKey("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("bucket_access").AtSliceIndex(0).AtMapKey("permissions"), knownvalue.StringExact("read_only")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("bucket_access").AtSliceIndex(1).AtMapKey("permissions"), knownvalue.StringExact("read_write")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("regions"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("regions").AtSliceIndex(0), knownvalue.StringExact(testRegion)),
				},
			},
		},
	})
}

func TestAccResourceObjectKey_update(t *testing.T) {
	t.Parallel()
	resName := "linode_object_storage_key.foobar"
	objectStorageKeyLabel := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkObjectKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, objectStorageKeyLabel),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckObjectKeyExists(),
					stateCheckObjectKeySecretAccessible(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(objectStorageKeyLabel)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("access_key"), knownvalue.NotNull()),
				},
			},
			{
				Config: tmpl.Updates(t, objectStorageKeyLabel),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckObjectKeyExists(),
					stateCheckObjectKeySecretAccessible(), // should be preserved in state
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(fmt.Sprintf("%s_renamed", objectStorageKeyLabel))),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("access_key"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func findObjectKeyResource(s *terraform.State) []*terraform.ResourceState {
	keys := []*terraform.ResourceState{}
	for _, res := range s.RootModule().Resources {
		if res.Type != "linode_object_storage_key" {
			continue
		}
		keys = append(keys, res)
	}
	return keys
}

func checkObjectKeySecretAccessible(s *terraform.State) error {
	keys := findObjectKeyResource(s)
	secret := keys[0].Primary.Attributes["secret_key"]

	if secret == "[REDACTED]" {
		return fmt.Errorf("Expected secret_key to be accessible but got '%s'", secret)
	}
	return nil
}

func checkObjectKeyExists(s *terraform.State) error {
	client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_object_storage_key" {
			continue
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error parsing %v to int", rs.Primary.ID)
		}

		_, err = client.GetObjectStorageKey(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Error retrieving state of Object Storage Key %s: %s", rs.Primary.Attributes["label"], err)
		}
	}

	return nil
}

func checkObjectKeyDestroy(s *terraform.State) error {
	client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_object_storage_key" {
			continue
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error parsing %v to int", rs.Primary.ID)
		}
		if id == 0 {
			return fmt.Errorf("Would have considered %v as %d", rs.Primary.ID, id)
		}

		_, err = client.GetObjectStorageKey(context.Background(), id)

		if err == nil {
			return fmt.Errorf("Linode Object Storage Key with id %d still exists", id)
		}

		if apiErr, ok := err.(*linodego.Error); ok && apiErr.Code != 404 {
			return fmt.Errorf("Error requesting Linode Object Storage Key with id %d", id)
		}
	}

	return nil
}

func stateCheckObjectKeyExists() statecheck.StateCheck {
	return acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Type != "linode_object_storage_key" {
				continue
			}

			idVal, ok := rc.AttributeValues["id"]
			if !ok {
				resp.Error = fmt.Errorf("No ID is set")
				return
			}

			idStr, ok := idVal.(string)
			if !ok {
				resp.Error = fmt.Errorf("Error: id is not a string")
				return
			}

			id, err := strconv.Atoi(idStr)
			if err != nil {
				resp.Error = fmt.Errorf("Error parsing %v to int", idStr)
				return
			}

			_, err = client.GetObjectStorageKey(ctx, id)
			if err != nil {
				resp.Error = fmt.Errorf("Error retrieving state of Object Storage Key %s: %s", idStr, err)
				return
			}
		}
	})
}

func stateCheckObjectKeySecretAccessible() statecheck.StateCheck {
	return acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Type != "linode_object_storage_key" {
				continue
			}

			secret, ok := rc.AttributeValues["secret_key"]
			if !ok {
				resp.Error = fmt.Errorf("secret_key attribute not found")
				return
			}

			secretStr, ok := secret.(string)
			if !ok || secretStr == "" {
				resp.Error = fmt.Errorf("Expected secret_key to be accessible but got '%v'", secret)
				return
			}

			if secretStr == "[REDACTED]" {
				resp.Error = fmt.Errorf("Expected secret_key to be accessible but got '%s'", secretStr)
				return
			}
			return
		}
	})
}
