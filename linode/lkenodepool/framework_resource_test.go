//go:build integration || lkenodepool

package lkenodepool_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
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
	acceptanceTmpl "github.com/linode/terraform-provider-linode/v3/linode/acceptance/tmpl"
	"github.com/linode/terraform-provider-linode/v3/linode/helper"
	"github.com/linode/terraform-provider-linode/v3/linode/lkenodepool/tmpl"
)

var (
	clusterID  string
	k8sVersion string
	testRegion string
)

func init() {
	resource.AddTestSweepers("linode_lke_node_pool", &resource.Sweeper{
		Name: "linode_lke_node_pool",
		F:    sweep,
	})

	clusterID = os.Getenv("LINODE_TEST_CLUSTER_ID")

	if clusterID == "" {
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

		k8sVersion = k8sVersions[len(k8sVersions)-1]

		region, err := acceptance.GetRandomRegionWithCaps([]string{"kubernetes"}, "core")
		if err != nil {
			log.Fatal(err)
		}

		testRegion = region
	}
}

func sweep(prefix string) error {
	client, err := acceptance.GetTestClient()
	if err != nil {
		return fmt.Errorf("Error getting client: %s", err)
	}
	clusterID, err := strconv.Atoi(os.Getenv("LINODE_TEST_CLUSTER_ID"))

	clusters, err := client.ListLKEClusters(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("Error getting clusters: %s", err)
	}
	var manyErrors []error
	for _, cluster := range clusters {
		if acceptance.ShouldSweep(prefix, cluster.Label) {
			if err := client.DeleteLKECluster(context.Background(), cluster.ID); err != nil {
				manyErrors = append(manyErrors, fmt.Errorf("Error destroying LKE cluster %d during sweep: %s", cluster.ID, err))
			}
		} else {
			pools, err := client.ListLKENodePools(context.Background(), clusterID, nil)
			if err != nil {
				return fmt.Errorf("Error getting node pools: %s", err)
			}
			for _, pool := range pools {
				if containsTagWithPrefix(pool, prefix) {
					log.Printf("[DEBUG] Found a leaked node pool, clusterID: %d, poolID: %d. Deleting", clusterID, pool.ID)
					err := client.DeleteLKENodePool(context.Background(), clusterID, pool.ID)
					if err != nil {
						manyErrors = append(manyErrors, fmt.Errorf("Error destroying nodepool %v during sweep: %s", pool.ID, err))
					}
				}
			}
		}
	}

	return errors.Join(manyErrors...)
}

func TestAccResourceNodePool_basic(t *testing.T) {
	t.Parallel()

	resName := "linode_lke_node_pool.foobar"
	clusterLabel := acctest.RandomWithPrefix("tf_test_")
	poolTag := acctest.RandomWithPrefix("tf_test_")

	templateData := createTemplateData()
	templateData.ClusterLabel = clusterLabel
	templateData.PoolTag = poolTag
	templateData.AutoscalerEnabled = true
	templateData.AutoscalerMin = 1
	templateData.AutoscalerMax = 2
	createConfig := createResourceConfig(t, &templateData)
	templateData.AutoscalerMin = 2
	templateData.AutoscalerMax = 3
	updateConfig := createResourceConfig(t, &templateData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkNodePoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: createConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodePoolExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-standard-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk_encryption"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("tags"), knownvalue.SetExact([]knownvalue.Check{
						knownvalue.StringExact("external"),
						knownvalue.StringExact(poolTag),
					})),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("autoscaler"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("autoscaler").AtSliceIndex(0).AtMapKey("min"), knownvalue.Int64Exact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("autoscaler").AtSliceIndex(0).AtMapKey("max"), knownvalue.Int64Exact(2)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("node_count"), knownvalue.Int64Exact(1)),
				},
			},
			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: resourceImportStateID,
			},
			{
				Config: updateConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodePoolExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("autoscaler").AtSliceIndex(0).AtMapKey("min"), knownvalue.Int64Exact(2)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("autoscaler").AtSliceIndex(0).AtMapKey("max"), knownvalue.Int64Exact(3)),
				},
			},
		},
	})
}

func TestAccResourceNodePool_disableAutoscaling(t *testing.T) {
	t.Parallel()

	resName := "linode_lke_node_pool.foobar"
	clusterLabel := acctest.RandomWithPrefix("tf_test_")
	poolTag := acctest.RandomWithPrefix("tf_test_")

	templateData := createTemplateData()
	templateData.ClusterLabel = clusterLabel
	templateData.PoolTag = poolTag
	templateData.AutoscalerEnabled = true
	templateData.AutoscalerMin = 1
	templateData.AutoscalerMax = 2
	createConfig := createResourceConfig(t, &templateData)
	templateData.AutoscalerEnabled = false
	templateData.NodeCount = 2
	updateConfig := createResourceConfig(t, &templateData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkNodePoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: createConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodePoolExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-standard-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("tags"), knownvalue.SetExact([]knownvalue.Check{
						knownvalue.StringExact("external"),
						knownvalue.StringExact(poolTag),
					})),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("autoscaler").AtSliceIndex(0).AtMapKey("min"), knownvalue.Int64Exact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("autoscaler").AtSliceIndex(0).AtMapKey("max"), knownvalue.Int64Exact(2)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("node_count"), knownvalue.Int64Exact(1)),
				},
			},
			{
				Config: updateConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodePoolExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("autoscaler"), knownvalue.ListSizeExact(0)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("node_count"), knownvalue.Int64Exact(2)),
				},
			},
		},
	})
}

func TestAccResourceNodePool_enableAutoscaling(t *testing.T) {
	t.Parallel()

	resName := "linode_lke_node_pool.foobar"
	clusterLabel := acctest.RandomWithPrefix("tf_test_")
	poolTag := acctest.RandomWithPrefix("tf_test_")

	templateData := createTemplateData()
	templateData.ClusterLabel = clusterLabel
	templateData.PoolTag = poolTag
	templateData.AutoscalerEnabled = false
	templateData.NodeCount = 2
	createConfig := createResourceConfig(t, &templateData)
	templateData.AutoscalerEnabled = true
	templateData.AutoscalerMin = 1
	templateData.AutoscalerMax = 2
	updateConfig := createResourceConfig(t, &templateData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkNodePoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: createConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodePoolExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-standard-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("tags"), knownvalue.SetExact([]knownvalue.Check{
						knownvalue.StringExact("external"),
						knownvalue.StringExact(poolTag),
					})),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("autoscaler"), knownvalue.ListSizeExact(0)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("node_count"), knownvalue.Int64Exact(2)),
				},
			},
			{
				Config: updateConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodePoolExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("autoscaler"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("autoscaler").AtSliceIndex(0).AtMapKey("min"), knownvalue.Int64Exact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("autoscaler").AtSliceIndex(0).AtMapKey("max"), knownvalue.Int64Exact(2)),
				},
			},
		},
	})
}

func TestAccResourceNodePool_update_type(t *testing.T) {
	t.Parallel()

	resName := "linode_lke_node_pool.foobar"
	clusterLabel := acctest.RandomWithPrefix("tf_test_")
	poolTag := acctest.RandomWithPrefix("tf_test_")

	templateData := createTemplateData()
	templateData.ClusterLabel = clusterLabel
	templateData.PoolTag = poolTag
	templateData.AutoscalerEnabled = false
	templateData.NodeCount = 1
	createConfig := createResourceConfig(t, &templateData)

	templateData.PoolNodeType = "g6-standard-2"
	updateConfig := createResourceConfig(t, &templateData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkNodePoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: createConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodePoolExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-standard-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("tags"), knownvalue.SetExact([]knownvalue.Check{
						knownvalue.StringExact("external"),
						knownvalue.StringExact(poolTag),
					})),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("node_count"), knownvalue.Int64Exact(1)),
				},
			},
			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: resourceImportStateID,
			},
			{
				Config: updateConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodePoolExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-standard-2")),
				},
			},
		},
	})
}

func TestAccResourceNodePool_taints_labels(t *testing.T) {
	t.Parallel()

	resName := "linode_lke_node_pool.foobar"
	clusterLabel := acctest.RandomWithPrefix("tf_test_")
	poolTag := acctest.RandomWithPrefix("tf_test_")

	templateData := createTemplateData()
	templateData.ClusterLabel = clusterLabel
	templateData.PoolTag = poolTag
	templateData.AutoscalerEnabled = false
	templateData.NodeCount = 1

	configWithoutTaintsLabels := createResourceConfig(t, &templateData)

	templateData.Labels = map[string]string{"foo": "bar"}
	templateData.Taints = []tmpl.TaintData{
		{
			Effect: "PreferNoSchedule",
			Key:    "foo",
			Value:  "bar",
		},
	}
	configWithTaintsLabels := createResourceConfig(t, &templateData)

	templateData.Labels = map[string]string{"bar": "baz"}
	templateData.Taints = []tmpl.TaintData{
		{
			Effect: "NoExecute",
			Key:    "bar",
			Value:  "baz",
		},
	}
	configWithUpdatedTaintsLabels := createResourceConfig(t, &templateData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkNodePoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: configWithTaintsLabels,
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodePoolExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("taint"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("taint").AtSliceIndex(0).AtMapKey("effect"), knownvalue.StringExact("PreferNoSchedule")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("taint").AtSliceIndex(0).AtMapKey("key"), knownvalue.StringExact("foo")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("taint").AtSliceIndex(0).AtMapKey("value"), knownvalue.StringExact("bar")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("labels"), knownvalue.MapSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("labels").AtMapKey("foo"), knownvalue.StringExact("bar")),
				},
			},
			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: resourceImportStateID,
			},
			{
				Config: configWithoutTaintsLabels,
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodePoolExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("taint"), knownvalue.SetSizeExact(0)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("labels"), knownvalue.MapSizeExact(0)),
				},
			},
			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: resourceImportStateID,
			},
			{
				Config: configWithTaintsLabels,
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodePoolExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("taint"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("taint").AtSliceIndex(0).AtMapKey("effect"), knownvalue.StringExact("PreferNoSchedule")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("taint").AtSliceIndex(0).AtMapKey("key"), knownvalue.StringExact("foo")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("taint").AtSliceIndex(0).AtMapKey("value"), knownvalue.StringExact("bar")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("labels"), knownvalue.MapSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("labels").AtMapKey("foo"), knownvalue.StringExact("bar")),
				},
			},
			{
				Config: configWithUpdatedTaintsLabels,
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodePoolExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("taint"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("taint").AtSliceIndex(0).AtMapKey("effect"), knownvalue.StringExact("NoExecute")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("taint").AtSliceIndex(0).AtMapKey("key"), knownvalue.StringExact("bar")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("taint").AtSliceIndex(0).AtMapKey("value"), knownvalue.StringExact("baz")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("labels"), knownvalue.MapSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("labels").AtMapKey("bar"), knownvalue.StringExact("baz")),
				},
			},
		},
	})
}

func TestAccResourceNodePoolEnterprise_basic(t *testing.T) {
	t.Parallel()
	resName := "linode_lke_node_pool.foobar"
	clusterLabel := acctest.RandomWithPrefix("tf_test_")
	poolTag := acctest.RandomWithPrefix("tf_test_")

	client, err := acceptance.GetTestClient()
	if err != nil {
		log.Fatalf("failed to get client: %s", err)
	}

	versions, err := client.ListLKETierVersions(context.Background(), "enterprise", nil)
	if err != nil {
		log.Fatal(err)
	}

	if len(versions) < 1 {
		t.Skip("No enterprise k8s versions found for test. Skipping now...")
	}

	enterpriseK8sVersion := versions[0].ID

	region, err := acceptance.GetRandomRegionWithCaps([]string{"Kubernetes Enterprise"}, "core")
	if err != nil {
		log.Fatal(err)
	}

	templateData := createTemplateData()
	templateData.ClusterLabel = clusterLabel
	templateData.PoolTag = poolTag
	templateData.K8sVersion = enterpriseK8sVersion
	templateData.Region = region
	templateData.Label = "foobar-pool"
	templateData.UpdateStrategy = "on_recycle"
	createConfig := createEnterpriseResourceConfig(t, &templateData)
	templateData.UpdateStrategy = "rolling_update"
	templateData.Label = ""
	updateConfig := createEnterpriseResourceConfig(t, &templateData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkNodePoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: createConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodePoolExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("k8s_version"), knownvalue.StringExact(enterpriseK8sVersion)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("update_strategy"), knownvalue.StringExact("on_recycle")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact("foobar-pool")),
				},
			},
			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: resourceImportStateID,
			},
			{
				Config: updateConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodePoolExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("k8s_version"), knownvalue.StringExact(enterpriseK8sVersion)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("update_strategy"), knownvalue.StringExact("rolling_update")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact("")),
				},
			},
		},
	})
}

func TestAccResourceNodePoolEnterprise_withFirewall(t *testing.T) {
	t.Parallel()
	resName := "linode_lke_node_pool.foobar"
	clusterLabel := acctest.RandomWithPrefix("tf_test_")
	poolTag := acctest.RandomWithPrefix("tf_test_")

	client, err := acceptance.GetTestClient()
	if err != nil {
		log.Fatalf("failed to get client: %s", err)
	}

	versions, err := client.ListLKETierVersions(context.Background(), "enterprise", nil)
	if err != nil {
		log.Fatal(err)
	}

	if len(versions) < 1 {
		t.Skip("No enterprise k8s versions found for test. Skipping now...")
	}

	enterpriseK8sVersion := versions[0].ID

	region, err := acceptance.GetRandomRegionWithCaps([]string{"Kubernetes Enterprise"}, "core")
	if err != nil {
		log.Fatal(err)
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

	templateData := createTemplateData()
	templateData.ClusterLabel = clusterLabel
	templateData.PoolTag = poolTag
	templateData.K8sVersion = enterpriseK8sVersion
	templateData.Region = region
	templateData.FirewallID = linodego.Pointer(firewall.ID)
	templateData.UpdateStrategy = "on_recycle"
	createConfig := createEnterpriseResourceConfig(t, &templateData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkNodePoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: createConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodePoolExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("firewall_id"), knownvalue.Int64Exact(int64(firewall.ID))),
				},
			},
			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: resourceImportStateID,
			},
		},
	})

	client.DeleteFirewall(context.Background(), firewall.ID)
}

func TestAccResourceNodePool_disableAutoscalingExplicitNodeCount(t *testing.T) {
	t.Parallel()

	resName := "linode_lke_node_pool.foobar"
	clusterLabel := acctest.RandomWithPrefix("tf_test_")
	poolTag := acctest.RandomWithPrefix("tf_test_")

	templateData := createTemplateData()
	templateData.ClusterLabel = clusterLabel
	templateData.PoolTag = poolTag
	templateData.NodeCount = 2
	templateData.AutoscalerEnabled = true
	templateData.AutoscalerMin = 1
	templateData.AutoscalerMax = 4

	createConfig := createResourceConfig(t, &templateData)

	templateData.AutoscalerEnabled = false

	dropAutoscalerConfig := createResourceConfig(t, &templateData)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkNodePoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: createConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodePoolExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("autoscaler"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("autoscaler").AtSliceIndex(0).AtMapKey("min"), knownvalue.Int64Exact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("autoscaler").AtSliceIndex(0).AtMapKey("max"), knownvalue.Int64Exact(4)),
				},
			},
			{
				Config: dropAutoscalerConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodePoolExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("autoscaler"), knownvalue.ListSizeExact(0)),
				},
			},
			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: resourceImportStateID,
			},
		},
	})
}

func stateCheckNodePoolExists() statecheck.StateCheck {
	return acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Type != "linode_lke_node_pool" {
				continue
			}

			idVal, ok := rc.AttributeValues["id"]
			if !ok {
				resp.Error = fmt.Errorf("No ID is set")
				return
			}

			id, err := strconv.Atoi(idVal.(string))
			if err != nil {
				resp.Error = fmt.Errorf("Error parsing ID %v to int", idVal)
				return
			}

			clusterIDVal, ok := rc.AttributeValues["cluster_id"]
			if !ok {
				resp.Error = fmt.Errorf("No cluster_id is set")
				return
			}

			// cluster_id is Int64Attribute in the schema, so tfjson stores it as float64
			clusterIDFloat, ok := clusterIDVal.(float64)
			if !ok {
				resp.Error = fmt.Errorf("expected cluster_id to be float64, got %T", clusterIDVal)
				return
			}
			clusterIDInt := int(clusterIDFloat)

			_, err = client.GetLKENodePool(context.Background(), clusterIDInt, id)
			if err != nil {
				resp.Error = fmt.Errorf("Error retrieving state of node pool %d: %v", id, err)
				return
			}
			return
		}

		resp.Error = fmt.Errorf("Error finding lke_node_pool")
	})
}

func checkNodePoolDestroy(s *terraform.State) error {
	client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client
	clusterID, poolID, err := extractIDs(s)
	if err != nil {
		return err
	}
	_, err = client.GetLKENodePool(context.Background(), clusterID, poolID)

	if err == nil {
		return fmt.Errorf("Node Pool with id %d still exists in cluster %d", poolID, clusterID)
	}

	if apiErr, ok := err.(*linodego.Error); ok && apiErr.Code != 404 {
		return fmt.Errorf("Error requesting Node Pool with id %d in cluster %d", poolID, clusterID)
	}
	return nil
}

func containsTagWithPrefix(pool linodego.LKENodePool, prefix string) bool {
	for _, tag := range pool.Tags {
		if strings.HasPrefix(tag, prefix) {
			return true
		}
	}
	return false
}

func createResourceConfig(t testing.TB, data *tmpl.TemplateData) string {
	return acceptanceTmpl.ProviderNoPoll(t) + tmpl.Generate(t, data)
}

func createEnterpriseResourceConfig(t testing.TB, data *tmpl.TemplateData) string {
	return acceptanceTmpl.ProviderNoPoll(t) + tmpl.EnterpriseBasic(t, data)
}

func createTemplateData() tmpl.TemplateData {
	var data tmpl.TemplateData
	data.ClusterID = clusterID
	data.K8sVersion = k8sVersion
	data.Region = testRegion
	data.PoolNodeType = "g6-standard-1"
	return data
}

func extractIDs(s *terraform.State) (int, int, error) {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_lke_node_pool" {
			continue
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return 0, 0, fmt.Errorf("Error parsing ID %v to int", rs.Primary.ID)
		}

		clusterID, err := strconv.Atoi(rs.Primary.Attributes["cluster_id"])
		if err != nil {
			return 0, 0, fmt.Errorf("Error parsing cluster_id %v to int", rs.Primary.Attributes["cluster_id"])
		}
		return clusterID, id, nil
	}

	return 0, 0, fmt.Errorf("Error finding lke_node_pool")
}

func resourceImportStateID(s *terraform.State) (string, error) {
	clusterID, id, err := extractIDs(s)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d,%d", clusterID, id), nil
}
