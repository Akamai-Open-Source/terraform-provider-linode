//go:build integration || lkeclusters

package lkeclusters_test

import (
	"context"
	"log"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/lkeclusters/tmpl"
)

var (
	k8sVersionLatest string
	testRegion       string
)

func init() {
	// Get valid K8s versions for testing
	client, err := acceptance.GetTestClient()
	if err != nil {
		log.Fatalf("failed to get client: %s", err)
	}

	versions, err := client.ListLKEVersions(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}

	k8sVersions := make([]string, len(versions))
	for i, v := range versions {
		k8sVersions[i] = v.ID
	}

	sort.Strings(k8sVersions)

	if len(k8sVersions) < 1 {
		log.Fatal("no k8s versions found")
	}

	k8sVersionLatest = k8sVersions[len(k8sVersions)-1]

	region, err := acceptance.GetRandomRegionWithCaps([]string{"kubernetes"}, "core")
	if err != nil {
		log.Fatal(err)
	}

	testRegion = region
}

func TestAccDataSourceLKEClusters_basic(t *testing.T) {
	t.Parallel()

	dataSourceName := "data.linode_lke_clusters.test"

	acceptance.RunTestWithRetries(t, 2, func(t *acceptance.WrappedT) {
		clusterName := acctest.RandomWithPrefix("tf_test")
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             acceptance.CheckLKEClusterDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.DataBasic(t, clusterName, k8sVersionLatest, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						acceptance.StateCheckResourceAttrGreaterThan(dataSourceName, "lke_clusters.#", 1),
					},
				},
				{
					Config: tmpl.DataFilter(t, clusterName, k8sVersionLatest, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("lke_clusters"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("lke_clusters").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact(clusterName)),
						statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("lke_clusters").AtSliceIndex(0).AtMapKey("region"), knownvalue.StringExact(testRegion)),
						statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("lke_clusters").AtSliceIndex(0).AtMapKey("k8s_version"), knownvalue.StringExact(k8sVersionLatest)),
						statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("lke_clusters").AtSliceIndex(0).AtMapKey("status"), knownvalue.StringExact("ready")),
						statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("lke_clusters").AtSliceIndex(0).AtMapKey("tags"), knownvalue.SetSizeExact(1)),
						statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("lke_clusters").AtSliceIndex(0).AtMapKey("tier"), knownvalue.StringExact("standard")),
						statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("lke_clusters").AtSliceIndex(0).AtMapKey("control_plane").AtMapKey("high_availability"), knownvalue.Bool(false)),
					},
				},
			},
		})
	})
}
