//go:build integration || lke

package lke_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/linodego"
	"github.com/linode/terraform-provider-linode/v3/linode"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/helper"
	"github.com/linode/terraform-provider-linode/v3/linode/lke/tmpl"
)

var (
	k8sVersions          []string
	k8sVersionLatest     string
	k8sVersionPrevious   string
	k8sVersionEnterprise string
	testRegion           string
)

const resourceClusterName = "linode_lke_cluster.test"

func init() {
	resource.AddTestSweepers("linode_lke_cluster", &resource.Sweeper{
		Name: "linode_lke_cluster",
		F:    sweep,
	})

	// Get valid K8s versions for testing
	client, err := acceptance.GetTestClient()
	if err != nil {
		log.Fatalf("failed to get client: %s", err)
	}

	versions, err := client.ListLKEVersions(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}

	k8sVersions = make([]string, len(versions))
	for i, v := range versions {
		k8sVersions[i] = v.ID
	}

	sort.Strings(k8sVersions)

	if len(k8sVersions) < 1 {
		log.Fatal("no k8s versions found")
	}

	k8sVersionLatest = k8sVersions[len(k8sVersions)-1]

	k8sVersionPrevious = k8sVersionLatest

	// If there are multiple images, use the second to last image
	if len(k8sVersions) > 1 {
		k8sVersionPrevious = k8sVersions[len(k8sVersions)-2]
	}

	region, err := acceptance.GetRandomRegionWithCaps([]string{"kubernetes"}, "core")
	if err != nil {
		log.Fatal(err)
	}

	testRegion = region

	enterpriseVersions, err := client.ListLKETierVersions(context.Background(), "enterprise", nil)
	if err != nil {
		log.Fatal(err)
	}

	if len(enterpriseVersions) < 1 {
		log.Print("no enterprise k8s versions found")
	} else {
		k8sVersionEnterprise = enterpriseVersions[0].ID
	}
}

func sweep(prefix string) error {
	client, err := acceptance.GetTestClient()
	if err != nil {
		return fmt.Errorf("Error getting client: %s", err)
	}

	clusters, err := client.ListLKEClusters(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("Error getting clusters: %s", err)
	}
	for _, cluster := range clusters {
		if !acceptance.ShouldSweep(prefix, cluster.Label) {
			continue
		}
		if err := client.DeleteLKECluster(context.Background(), cluster.ID); err != nil {
			return fmt.Errorf("Error destroying LKE cluster %d during sweep: %s", cluster.ID, err)
		}
	}

	return nil
}

func checkLKEExistsStateCheck(cluster *linodego.LKECluster) statecheck.StateCheck {
	return acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		if req.State == nil || req.State.Values == nil {
			resp.Error = fmt.Errorf("invalid state")
			return
		}

		var resourceID string
		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Address == resourceClusterName {
				idVal, ok := rc.AttributeValues["id"]
				if !ok {
					resp.Error = fmt.Errorf("No ID is set")
					return
				}
				resourceID = fmt.Sprintf("%v", idVal)
				break
			}
		}

		if resourceID == "" {
			resp.Error = fmt.Errorf("could not find resource %s", resourceClusterName)
			return
		}

		id, err := strconv.Atoi(resourceID)
		if err != nil {
			resp.Error = fmt.Errorf("Error parsing %v to int", resourceID)
			return
		}

		found, err := client.GetLKECluster(context.Background(), id)
		if err != nil {
			resp.Error = fmt.Errorf("Error retrieving state of LKE Cluster %s: %s", resourceID, err)
			return
		}

		*cluster = *found
	})
}

// waitForAllNodesReady waits for every Node in every NodePool of the LKE Cluster to be in
// a ready state.
func waitForAllNodesReady(t testing.TB, cluster *linodego.LKECluster, pollInterval, timeout time.Duration) {
	t.Helper()

	ctx := context.Background()
	client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(timeout))
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for LKE Cluster (%d) Nodes to be ready", cluster.ID)

		case <-time.NewTicker(pollInterval).C:
			nodePools, err := client.ListLKENodePools(ctx, cluster.ID, &linodego.ListOptions{})
			if err != nil {
				t.Fatalf("failed to get NodePools for LKE Cluster (%d): %s", cluster.ID, err)
			}

			// Check that all NodePools are ready.
			for _, nodePool := range nodePools {
				for _, linode := range nodePool.Linodes {
					if linode.Status != linodego.LKELinodeReady {
						// This NodePool is not finished initializing; check again later.
						continue
					}
				}
			}

			// If we get to this point, all NodePools must be ready.
			return
		}
	}
}

func TestSmokeTests_lke(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{"TestAccResourceLKECluster_basic_smoke", TestAccResourceLKECluster_basic_smoke},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestAccResourceLKECluster_basic_smoke(t *testing.T) {
	t.Parallel()

	acceptance.RunTestWithRetries(t, 2, func(t *acceptance.WrappedT) {
		clusterName := acctest.RandomWithPrefix("tf_test")
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             acceptance.CheckLKEClusterDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.Basic(t, clusterName, k8sVersionLatest, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(clusterName)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("k8s_version"), knownvalue.StringExact(k8sVersionLatest)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("status"), knownvalue.StringExact("ready")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("tier"), knownvalue.StringExact("standard")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("tags"), knownvalue.SetSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("type"), knownvalue.StringExact("g6-standard-1")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("count"), knownvalue.StringExact("3")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("disk_encryption"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("nodes"), knownvalue.ListSizeExact(3)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("tags"), knownvalue.SetSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("tags").AtSliceIndex(0), knownvalue.StringExact("test")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("control_plane"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("control_plane").AtSliceIndex(0).AtMapKey("high_availability"), knownvalue.Bool(false)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("id"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("kubeconfig"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("dashboard_url"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("test")),

						// Ensure the lke_cluster_id field is populated on a sample
						// node from the new cluster.
						statecheck.CompareValuePairs(
							resourceClusterName,
							tfjsonpath.New("id"),
							"data.linode_instances.test",
							tfjsonpath.New("instances").AtSliceIndex(0).AtMapKey("lke_cluster_id"),
							compare.ValuesSame(),
						),
					},
				},
			},
		})
	})
}

func TestAccResourceLKECluster_k8sUpgrade(t *testing.T) {
	t.Parallel()

	var cluster linodego.LKECluster

	acceptance.RunTestWithRetries(t, 2, func(t *acceptance.WrappedT) {
		clusterName := acctest.RandomWithPrefix("tf_test")
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             acceptance.CheckLKEClusterDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.ManyPools(t, clusterName, k8sVersionPrevious, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						checkLKEExistsStateCheck(&cluster),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(clusterName)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("k8s_version"), knownvalue.StringExact(k8sVersionPrevious)),
					},
				},
				{
					PreConfig: func() {
						// Before we upgrade the Cluster to a newer version of Kubernetes, we need to first
						// ensure that every Node in each of this cluster's NodePool is ready. Otherwise, the
						// recycle will not actually occur.
						waitForAllNodesReady(t, &cluster, time.Second*5, time.Minute*5)
					},
					Config: tmpl.ManyPools(t, clusterName, k8sVersionLatest, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(clusterName)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("k8s_version"), knownvalue.StringExact(k8sVersionLatest)),
					},
				},
			},
		})
	})
}

func TestAccResourceLKECluster_basicUpdates(t *testing.T) {
	t.Parallel()

	provider, providerMap := acceptance.CreateTestProvider()

	// We want to ensure that non-updated values are excluded from update requests
	acceptance.ModifyProviderMeta(provider,
		func(ctx context.Context, _ *schema.ResourceData, config *helper.ProviderMeta) error {
			config.Client.OnBeforeRequest(func(request *linodego.Request) error {
				if request.Method != "PUT" {
					return nil
				}

				var opts linodego.LKEClusterUpdateOptions

				if err := json.Unmarshal([]byte(request.Body.(string)), &opts); err != nil {
					t.Fatal(err)
				}

				if opts.K8sVersion != "" {
					t.Fatalf(
						"expected k8s version to be excluded from update request, got %s",
						opts.K8sVersion)
				}

				return nil
			})

			return nil
		})

	acceptance.RunTestWithRetries(t, 2, func(t *acceptance.WrappedT) {
		clusterName := acctest.RandomWithPrefix("tf_test")
		newClusterName := acctest.RandomWithPrefix("tf_test")
		resource.Test(t, resource.TestCase{
			PreCheck:  func() { acceptance.PreCheck(t) },
			Providers: providerMap,
			Steps: []resource.TestStep{
				{
					Config: tmpl.Basic(t, clusterName, k8sVersionLatest, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(clusterName)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("tags"), knownvalue.SetSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("tags"), knownvalue.SetSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("tags").AtSliceIndex(0), knownvalue.StringExact("test")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("count"), knownvalue.StringExact("3")),
					},
				},
				{
					Config: tmpl.Updates(t, newClusterName, k8sVersionLatest, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(newClusterName)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("tags"), knownvalue.SetSizeExact(2)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("tags"), knownvalue.SetSizeExact(2)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("tags").AtSliceIndex(0), knownvalue.StringExact("test")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("tags").AtSliceIndex(1), knownvalue.StringExact("test-2")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("count"), knownvalue.StringExact("4")),
					},
				},
			},
		})
	})
}

func TestAccResourceLKECluster_poolUpdates(t *testing.T) {
	t.Parallel()

	acceptance.RunTestWithRetries(t, 2, func(t *acceptance.WrappedT) {
		clusterName := acctest.RandomWithPrefix("tf_test")
		newClusterName := acctest.RandomWithPrefix("tf_test")
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             acceptance.CheckLKEClusterDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.Basic(t, clusterName, k8sVersionLatest, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(clusterName)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("count"), knownvalue.StringExact("3")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("status"), knownvalue.StringExact("ready")),
					},
				},
				{
					Config: tmpl.ComplexPools(t, newClusterName, k8sVersionLatest, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(newClusterName)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("count"), knownvalue.StringExact("2")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(1).AtMapKey("count"), knownvalue.StringExact("1")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(2).AtMapKey("count"), knownvalue.StringExact("2")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(3).AtMapKey("count"), knownvalue.StringExact("2")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("status"), knownvalue.StringExact("ready")),
					},
				},
				{
					Config: tmpl.Basic(t, clusterName, k8sVersionLatest, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(clusterName)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("count"), knownvalue.StringExact("3")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("status"), knownvalue.StringExact("ready")),
					},
				},
				{
					Config: tmpl.LabelledPools(t, clusterName, k8sVersionLatest, testRegion, "test"),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("test")),
					},
				},
			},
		})
	})
}

func TestAccResourceLKECluster_removeUnmanagedPool(t *testing.T) {
	t.Parallel()

	var cluster linodego.LKECluster

	acceptance.RunTestWithRetries(t, 2, func(t *acceptance.WrappedT) {
		clusterName := acctest.RandomWithPrefix("tf_test")
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             acceptance.CheckLKEClusterDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.Basic(t, clusterName, k8sVersionLatest, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						checkLKEExistsStateCheck(&cluster),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(clusterName)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("status"), knownvalue.StringExact("ready")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool"), knownvalue.ListSizeExact(1)),
					},
				},
				{
					PreConfig: func() {
						client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client
						if _, err := client.CreateLKENodePool(context.Background(), cluster.ID, linodego.LKENodePoolCreateOptions{
							Count: 1,
							Type:  "g6-standard-1",
						}); err != nil {
							t.Errorf("failed to create unmanaged pool for cluster %d: %s", cluster.ID, err)
						}

						pools, err := client.ListLKENodePools(context.Background(), cluster.ID, nil)
						if err != nil {
							t.Errorf("failed to get pools for cluster %d: %s", cluster.ID, err)
						}

						if len(pools) != 2 {
							t.Errorf("expected cluster to have 2 pools but got %d", len(pools))
						}
					},
					Config: tmpl.Basic(t, clusterName, k8sVersionLatest, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool"), knownvalue.ListSizeExact(1)),
					},
				},
			},
		})
	})
}

func TestAccResourceLKECluster_autoScaler(t *testing.T) {
	t.Parallel()

	acceptance.RunTestWithRetries(t, 2, func(t *acceptance.WrappedT) {
		clusterName := acctest.RandomWithPrefix("tf_test")
		// newClusterName := acctest.RandomWithPrefix("tf_test")
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             acceptance.CheckLKEClusterDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.Basic(t, clusterName, k8sVersionLatest, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(clusterName)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("count"), knownvalue.StringExact("3")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("autoscaler"), knownvalue.ListSizeExact(0)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("status"), knownvalue.StringExact("ready")),
					},
				},
				{
					Config: tmpl.Autoscaler(t, clusterName, k8sVersionLatest, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(clusterName)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("count"), knownvalue.StringExact("3")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("autoscaler"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("autoscaler").AtSliceIndex(0).AtMapKey("min"), knownvalue.StringExact("1")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("autoscaler").AtSliceIndex(0).AtMapKey("max"), knownvalue.StringExact("5")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("status"), knownvalue.StringExact("ready")),
					},
				},
				{
					Config: tmpl.AutoscalerUpdates(t, clusterName, k8sVersionLatest, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(clusterName)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("count"), knownvalue.StringExact("3")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("autoscaler"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("autoscaler").AtSliceIndex(0).AtMapKey("min"), knownvalue.StringExact("1")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("autoscaler").AtSliceIndex(0).AtMapKey("max"), knownvalue.StringExact("8")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("status"), knownvalue.StringExact("ready")),
					},
				},
				{
					Config: tmpl.AutoscalerManyPools(t, clusterName, k8sVersionLatest, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(clusterName)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool"), knownvalue.ListSizeExact(2)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("count"), knownvalue.StringExact("5")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("autoscaler"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("autoscaler").AtSliceIndex(0).AtMapKey("min"), knownvalue.StringExact("3")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("autoscaler").AtSliceIndex(0).AtMapKey("max"), knownvalue.StringExact("8")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(1).AtMapKey("count"), knownvalue.StringExact("3")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(1).AtMapKey("autoscaler"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(1).AtMapKey("autoscaler").AtSliceIndex(0).AtMapKey("min"), knownvalue.StringExact("1")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(1).AtMapKey("autoscaler").AtSliceIndex(0).AtMapKey("max"), knownvalue.StringExact("8")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("status"), knownvalue.StringExact("ready")),
					},
				},
				{
					Config: tmpl.Basic(t, clusterName, k8sVersionLatest, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(clusterName)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("count"), knownvalue.StringExact("3")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("autoscaler"), knownvalue.ListSizeExact(0)),
					},
				},
			},
		})
	})
}

func TestAccResourceLKECluster_controlPlane(t *testing.T) {
	t.Parallel()

	acceptance.RunTestWithRetries(t, 2, func(t *acceptance.WrappedT) {
		clusterName := acctest.RandomWithPrefix("tf_test")
		testIPv4 := "0.0.0.0/0"
		testIPv6 := "2001:db8::/32"
		testIPv4Updated := "203.0.113.1"
		testIPv6Updated := "2001:db8:1234:abcd::/64"

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             acceptance.CheckLKEClusterDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.ControlPlane(t, clusterName, k8sVersionLatest, testRegion, testIPv4, testIPv6, false, true),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(clusterName)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("count"), knownvalue.StringExact("1")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("autoscaler"), knownvalue.ListSizeExact(0)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("control_plane").AtSliceIndex(0).AtMapKey("high_availability"), knownvalue.Bool(false)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("control_plane").AtSliceIndex(0).AtMapKey("acl").AtSliceIndex(0).AtMapKey("enabled"), knownvalue.Bool(true)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("control_plane").AtSliceIndex(0).AtMapKey("acl").AtSliceIndex(0).AtMapKey("addresses").AtSliceIndex(0).AtMapKey("ipv4").AtSliceIndex(0), knownvalue.StringExact(testIPv4)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("control_plane").AtSliceIndex(0).AtMapKey("acl").AtSliceIndex(0).AtMapKey("addresses").AtSliceIndex(0).AtMapKey("ipv6").AtSliceIndex(0), knownvalue.StringExact(testIPv6)),
					},
				},
				{
					Config: tmpl.ControlPlane(t, clusterName, k8sVersionLatest, testRegion, testIPv4Updated, testIPv6Updated, true, true),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(clusterName)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("count"), knownvalue.StringExact("1")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("autoscaler"), knownvalue.ListSizeExact(0)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("control_plane").AtSliceIndex(0).AtMapKey("high_availability"), knownvalue.Bool(true)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("control_plane").AtSliceIndex(0).AtMapKey("acl").AtSliceIndex(0).AtMapKey("enabled"), knownvalue.Bool(true)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("control_plane").AtSliceIndex(0).AtMapKey("acl").AtSliceIndex(0).AtMapKey("addresses").AtSliceIndex(0).AtMapKey("ipv4").AtSliceIndex(0), knownvalue.StringExact(testIPv4Updated)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("control_plane").AtSliceIndex(0).AtMapKey("acl").AtSliceIndex(0).AtMapKey("addresses").AtSliceIndex(0).AtMapKey("ipv6").AtSliceIndex(0), knownvalue.StringExact(testIPv6Updated)),
					},
				},
				{
					Config: tmpl.ControlPlane(t, clusterName, k8sVersionLatest, testRegion, testIPv4Updated, testIPv6Updated, false, true),

					// Expect a 400 response when attempting to disable HA
					ExpectError: regexp.MustCompile("\\[400]"),
				},
			},
		})
	})
}

func TestAccResourceLKECluster_noCount(t *testing.T) {
	t.Parallel()

	clusterName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckLKEClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config:      tmpl.NoCount(t, clusterName, k8sVersionLatest, testRegion),
				ExpectError: regexp.MustCompile("pool.*: `count` must be defined when no autoscaler is defined"),
			},
		},
	})
}

func TestAccResourceLKECluster_implicitCount(t *testing.T) {
	t.Parallel()

	clusterName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckLKEClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.AutoscalerNoCount(t, clusterName, k8sVersionLatest, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(clusterName)),
					statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("count"), knownvalue.StringExact("2")),
					statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("autoscaler"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("autoscaler").AtSliceIndex(0).AtMapKey("min"), knownvalue.StringExact("2")),
					statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("autoscaler").AtSliceIndex(0).AtMapKey("max"), knownvalue.StringExact("4")),
				},
			},
			{
				Config: tmpl.AutoscalerNoCount(t, clusterName, k8sVersionLatest, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(clusterName)),
					statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("count"), knownvalue.StringExact("2")),
					statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("autoscaler"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("autoscaler").AtSliceIndex(0).AtMapKey("min"), knownvalue.StringExact("2")),
					statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("autoscaler").AtSliceIndex(0).AtMapKey("max"), knownvalue.StringExact("4")),
				},
			},
		},
	})
}

func TestAccResourceLKEClusterNodePoolTaintsLabels(t *testing.T) {
	t.Parallel()

	acceptance.RunTestWithRetries(t, 2, func(t *acceptance.WrappedT) {
		clusterName := acctest.RandomWithPrefix("tf_test")
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             acceptance.CheckLKEClusterDestroy,
			Steps: []resource.TestStep{
				{
					// create a cluster with taints and labels
					Config: tmpl.DataTaintsLabels(
						t, clusterName, k8sVersionLatest, testRegion, []tmpl.TaintData{
							{
								Effect: "PreferNoSchedule",
								Key:    "foo",
								Value:  "bar",
							},
						},
						map[string]string{"foo": "bar"},
					),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("nodes"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("taint"), knownvalue.SetSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("taint").AtSliceIndex(0).AtMapKey("effect"), knownvalue.StringExact("PreferNoSchedule")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("taint").AtSliceIndex(0).AtMapKey("key"), knownvalue.StringExact("foo")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("taint").AtSliceIndex(0).AtMapKey("value"), knownvalue.StringExact("bar")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("labels").AtMapKey("foo"), knownvalue.StringExact("bar")),
					},
				},
				{
					// update taints and labels values
					Config: tmpl.DataTaintsLabels(
						t, clusterName, k8sVersionLatest, testRegion, []tmpl.TaintData{
							{
								Effect: "PreferNoSchedule",
								Key:    "baz",
								Value:  "qux",
							},
						},
						map[string]string{"baz": "qux"},
					),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("nodes"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("taint"), knownvalue.SetSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("taint").AtSliceIndex(0).AtMapKey("effect"), knownvalue.StringExact("PreferNoSchedule")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("taint").AtSliceIndex(0).AtMapKey("key"), knownvalue.StringExact("baz")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("taint").AtSliceIndex(0).AtMapKey("value"), knownvalue.StringExact("qux")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("labels").AtMapKey("baz"), knownvalue.StringExact("qux")),
					},
				},
				{
					// remove taints and labels
					Config: tmpl.DataTaintsLabels(t, clusterName, k8sVersionLatest, testRegion, nil, nil),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("nodes"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("taint"), knownvalue.SetSizeExact(0)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("labels"), knownvalue.MapSizeExact(0)),
					},
				},
				{
					// add taints and labels back
					Config: tmpl.DataTaintsLabels(
						t, clusterName, k8sVersionLatest, testRegion, []tmpl.TaintData{
							{
								Effect: "PreferNoSchedule",
								Key:    "foo",
								Value:  "bar",
							},
						},
						map[string]string{"foo": "bar"},
					),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("nodes"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("taint"), knownvalue.SetSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("taint").AtSliceIndex(0).AtMapKey("effect"), knownvalue.StringExact("PreferNoSchedule")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("taint").AtSliceIndex(0).AtMapKey("key"), knownvalue.StringExact("foo")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("taint").AtSliceIndex(0).AtMapKey("value"), knownvalue.StringExact("bar")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("labels").AtMapKey("foo"), knownvalue.StringExact("bar")),
					},
				},
				{
					ResourceName:            resourceClusterName,
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"pool.0.nodes.0.status"},
				},
			},
		})
	})
}

func TestAccResourceLKECluster_enterprise(t *testing.T) {
	t.Parallel()

	k8sVersionEnterprise = "v1.31.9+lke5" // currently only this version works with BYO VPC

	if k8sVersionEnterprise == "" {
		t.Skip("No available k8s version for LKE Enterprise test. Skipping now...")
	}

	enterpriseRegion := "no-osl-1" // currently only oslo region works with BYO VPC

	// TODO: revert to dynamic selection once more regions available
	//enterpriseRegion, err := acceptance.GetRandomRegionWithCaps([]string{"Kubernetes Enterprise", "VPCs"}, "core")
	//if err != nil {
	//	log.Fatal(err)
	//}

	client, err := acceptance.GetTestClient()
	if err != nil {
		log.Fatalf("failed to get client: %s", err)
	}

	firewall, err := client.CreateFirewall(context.Background(), linodego.FirewallCreateOptions{
		Label: "tftest-enterprise-upgrade-" + acctest.RandString(5),
		Rules: linodego.FirewallRuleSet{
			InboundPolicy:  "ACCEPT",
			OutboundPolicy: "ACCEPT",
		},
	})
	if err != nil {
		t.Errorf("failed creating firewall: %v", err)
	}

	acceptance.RunTestWithRetries(t, 2, func(t *acceptance.WrappedT) {
		clusterName := acctest.RandomWithPrefix("tf-test")
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             acceptance.CheckLKEClusterDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.Enterprise(t, clusterName, k8sVersionEnterprise, enterpriseRegion, "on_recycle"),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(clusterName)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("region"), knownvalue.StringExact(enterpriseRegion)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("k8s_version"), knownvalue.StringExact(k8sVersionEnterprise)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("status"), knownvalue.StringExact("ready")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("tier"), knownvalue.StringExact("enterprise")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("tags"), knownvalue.SetSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("type"), knownvalue.StringExact("g6-standard-1")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("count"), knownvalue.StringExact("3")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("k8s_version"), knownvalue.StringExact(k8sVersionEnterprise)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("update_strategy"), knownvalue.StringExact("on_recycle")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("kubeconfig"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("vpc_id"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("subnet_id"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("stack_type"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("control_plane").AtSliceIndex(0).AtMapKey("audit_logs_enabled"), knownvalue.NotNull()),
					},
				},
				{
					Config: tmpl.Enterprise(t, clusterName, k8sVersionEnterprise, enterpriseRegion, "rolling_update"),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(clusterName)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("region"), knownvalue.StringExact(enterpriseRegion)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("k8s_version"), knownvalue.StringExact(k8sVersionEnterprise)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("status"), knownvalue.StringExact("ready")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("tier"), knownvalue.StringExact("enterprise")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("tags"), knownvalue.SetSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("type"), knownvalue.StringExact("g6-standard-1")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("count"), knownvalue.StringExact("3")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("k8s_version"), knownvalue.StringExact(k8sVersionEnterprise)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("update_strategy"), knownvalue.StringExact("rolling_update")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("kubeconfig"), knownvalue.NotNull()),
					},
				},
				{
					Config: tmpl.EnterpriseFirewall(t, clusterName, k8sVersionEnterprise, enterpriseRegion, firewall.ID),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(clusterName)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("region"), knownvalue.StringExact(enterpriseRegion)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("k8s_version"), knownvalue.StringExact(k8sVersionEnterprise)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("status"), knownvalue.StringExact("ready")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("tier"), knownvalue.StringExact("enterprise")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("tags"), knownvalue.SetSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("firewall_id"), knownvalue.StringExact(fmt.Sprintf("%d", firewall.ID))),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("count"), knownvalue.StringExact("3")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("k8s_version"), knownvalue.StringExact(k8sVersionEnterprise)),
					},
				},
			},
		})
	})
}

func TestAccResourceLKECluster_enterpriseNoPools(t *testing.T) {
	t.Parallel()

	k8sVersionEnterprise = "v1.31.9+lke7"

	enterpriseRegion := "us-ord"

	acceptance.RunTestWithRetries(t, 2, func(t *acceptance.WrappedT) {
		clusterName := acctest.RandomWithPrefix("tf-test")
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             acceptance.CheckLKEClusterDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.EnterpriseNoPools(t, clusterName, k8sVersionEnterprise, enterpriseRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(clusterName)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("region"), knownvalue.StringExact(enterpriseRegion)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("k8s_version"), knownvalue.StringExact(k8sVersionEnterprise)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("status"), knownvalue.StringExact("ready")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("tier"), knownvalue.StringExact("enterprise")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool"), knownvalue.ListSizeExact(0)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("kubeconfig"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("vpc_id"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("subnet_id"), knownvalue.NotNull()),
					},
				},
			},
		})
	})
}

func TestAccResourceLKECluster_standardNoPools(t *testing.T) {
	t.Parallel()

	clusterName := acctest.RandomWithPrefix("tf-test")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckLKEClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config:      tmpl.StandardNoPools(t, clusterName, k8sVersionLatest, testRegion),
				ExpectError: regexp.MustCompile("at least one pool is required for standard tier clusters"),
			},
		},
	})
}

func TestAccResourceLKECluster_apl(t *testing.T) {
	t.Parallel()
	acceptance.RunTestWithRetries(t, 2, func(t *acceptance.WrappedT) {
		clusterName := acctest.RandomWithPrefix("tf_test")
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             acceptance.CheckLKEClusterDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.APLEnabled(t, clusterName, k8sVersionLatest, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(clusterName)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("apl_enabled"), knownvalue.Bool(true)),
					},
				},
			},
		})
	})
}

func TestAccResourceLKECluster_acl_disabled_addresses(t *testing.T) {
	t.Parallel()

	clusterName := acctest.RandomWithPrefix("tf_test")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckLKEClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config:      tmpl.ACLDisabledAddressesDisallowed(t, clusterName, k8sVersionLatest, testRegion),
				ExpectError: regexp.MustCompile("addresses are not acceptable when ACL is disabled"),
			},
		},
	})
}

// TestAccResourceLKECluster_tierNoAccess ensures that a tier value of
// `standard` will not trigger diffs in a cluster that does not expose
// the `tier` field due to various circumstances.
func TestAccResourceLKECluster_tierNoAccess(t *testing.T) {
	t.Parallel()

	clusterName := acctest.RandomWithPrefix("tf_test")

	apiVersion := "v4"

	apiVersionOverrideProvider := acceptance.ModifyProviderMeta(
		linode.Provider(),
		func(ctx context.Context, data *schema.ResourceData, config *helper.ProviderMeta) error {
			config.Config.APIVersion = apiVersion
			config.Client.SetAPIVersion(apiVersion)

			return nil
		},
	)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"linode": func() (tfprotov6.ProviderServer, error) {
				return acceptance.ProtoV6CustomProviderFactories["linode"](nil, apiVersionOverrideProvider)
			},
		},
		CheckDestroy: acceptance.CheckLKEClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.TierConditional(
					t,
					clusterName,
					k8sVersionLatest,
					testRegion,
					string(linodego.LKEVersionStandard),
				),
				ExpectError: regexp.MustCompile(
					"tier: The api_version provider argument must be set to 'v4beta' to use this field\\.",
				),
			},
			{
				Config: tmpl.TierConditional(
					t,
					clusterName,
					k8sVersionLatest,
					testRegion,
					"",
				),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.CustomStateCheck(
						func(
							ctx context.Context,
							req statecheck.CheckStateRequest,
							resp *statecheck.CheckStateResponse,
						) {
							if req.State == nil || req.State.Values == nil {
								resp.Error = fmt.Errorf("invalid state")
								return
							}

							for _, resourceEntry := range req.State.Values.RootModule.Resources {
								if resourceEntry.Type != "linode_lke_cluster" || resourceEntry.Name != "test" {
									continue
								}

								tier := resourceEntry.AttributeValues["tier"].(string)
								if tier != "" {
									log.Printf(
										"MAINTAINER ALERT: The tier field is now populated under "+
											"the v4 API version (value %s). Validation should be removed.\n",
										tier,
									)
								}
							}
						},
					),
				},
			},
			{
				Config: tmpl.TierConditional(
					t,
					clusterName,
					k8sVersionLatest,
					testRegion,
					string(linodego.LKEVersionEnterprise),
				),
				ExpectError: regexp.MustCompile(
					"tier: The api_version provider argument must be set to 'v4beta' to use this field\\.",
				),
			},
			{
				PreConfig: func() {
					apiVersion = "v4beta"
				},
				Config: tmpl.TierConditional(
					t,
					clusterName,
					k8sVersionLatest,
					testRegion,
					string(linodego.LKEVersionStandard),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"linode_lke_cluster.test",
						tfjsonpath.New("tier"),
						knownvalue.StringExact(string(linodego.LKEVersionStandard)),
					),
				},
			},
		},
	})
}
