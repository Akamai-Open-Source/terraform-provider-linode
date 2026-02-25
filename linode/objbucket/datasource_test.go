//go:build integration || objbucket

package objbucket_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/objbucket/tmpl"
)

func TestAccDataSourceBucket_basic(t *testing.T) {
	t.Parallel()

	resourceName := "data.linode_object_storage_bucket.foobar"
	objectStorageBucketName := acctest.RandomWithPrefix("tf-test")

	acceptance.RunTestWithRetries(t, 5, func(t *acceptance.WrappedT) {
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             checkBucketDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.DataBasicWithCluster(t, objectStorageBucketName, testCluster),
					ConfigStateChecks: []statecheck.StateCheck{
						stateCheckBucketExists(),
						statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("cluster"), knownvalue.StringExact(testCluster)),
						statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
						statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("label"), knownvalue.StringExact(objectStorageBucketName)),
						statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("hostname"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("created"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("objects"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("size"), knownvalue.NotNull()),
					},
				},
			},
		})
	})
}

func TestAccDataSourceBucket_basic_cluster(t *testing.T) {
	t.Parallel()

	resourceName := "data.linode_object_storage_bucket.foobar"
	objectStorageBucketName := acctest.RandomWithPrefix("tf-test")

	acceptance.RunTestWithRetries(t, 5, func(t *acceptance.WrappedT) {
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             checkBucketDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.DataBasic(t, objectStorageBucketName, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						stateCheckBucketExists(),
						statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("cluster"), knownvalue.StringExact(testCluster)),
						statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
						statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("label"), knownvalue.StringExact(objectStorageBucketName)),
						statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("hostname"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("created"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("objects"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("size"), knownvalue.NotNull()),
					},
				},
			},
		})
	})
}
