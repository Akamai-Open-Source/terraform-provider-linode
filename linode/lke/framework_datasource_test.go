//go:build integration || lke

package lke_test

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/linodego"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/lke/tmpl"
)

const dataSourceClusterName = "data.linode_lke_cluster.test"

func TestAccDataSourceLKECluster_taints_labels(t *testing.T) {
	t.Parallel()

	acceptance.RunTestWithRetries(t, 2, func(t *acceptance.WrappedT) {
		clusterName := acctest.RandomWithPrefix("tf_test")
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             acceptance.CheckLKEClusterDestroy,
			Steps: []resource.TestStep{
				{
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
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools").AtSliceIndex(0).AtMapKey("nodes"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools").AtSliceIndex(0).AtMapKey("taints"), knownvalue.SetSizeExact(1)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools").AtSliceIndex(0).AtMapKey("labels").AtMapKey("foo"), knownvalue.StringExact("bar")),
					},
				},
			},
		})
	})
}

func TestAccDataSourceLKECluster_basic(t *testing.T) {
	t.Parallel()

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
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(clusterName)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("k8s_version"), knownvalue.StringExact(k8sVersionLatest)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("status"), knownvalue.StringExact("ready")),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("tier"), knownvalue.StringExact("standard")),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("tags"), knownvalue.SetSizeExact(1)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools").AtSliceIndex(0).AtMapKey("type"), knownvalue.StringExact("g6-standard-1")),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("test")),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools").AtSliceIndex(0).AtMapKey("count"), knownvalue.StringExact("3")),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools").AtSliceIndex(0).AtMapKey("disk_encryption"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools").AtSliceIndex(0).AtMapKey("nodes"), knownvalue.ListSizeExact(3)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools").AtSliceIndex(0).AtMapKey("autoscaler"), knownvalue.ListSizeExact(0)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("control_plane").AtSliceIndex(0).AtMapKey("high_availability"), knownvalue.Bool(false)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("kubeconfig"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("dashboard_url"), knownvalue.NotNull()),
					},
				},
			},
		})
	})
}

func TestAccDataSourceLKECluster_autoscaler(t *testing.T) {
	t.Parallel()

	acceptance.RunTestWithRetries(t, 2, func(t *acceptance.WrappedT) {
		clusterName := acctest.RandomWithPrefix("tf_test")
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             acceptance.CheckLKEClusterDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.DataAutoscaler(t, clusterName, k8sVersionLatest, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(clusterName)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("k8s_version"), knownvalue.StringExact(k8sVersionLatest)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("status"), knownvalue.StringExact("ready")),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("tags"), knownvalue.SetSizeExact(1)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools").AtSliceIndex(0).AtMapKey("type"), knownvalue.StringExact("g6-standard-1")),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools").AtSliceIndex(0).AtMapKey("count"), knownvalue.StringExact("3")),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools").AtSliceIndex(0).AtMapKey("nodes"), knownvalue.ListSizeExact(3)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("kubeconfig"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("autoscaler"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("autoscaler").AtSliceIndex(0).AtMapKey("min"), knownvalue.StringExact("1")),
						statecheck.ExpectKnownValue(resourceClusterName, tfjsonpath.New("pool").AtSliceIndex(0).AtMapKey("autoscaler").AtSliceIndex(0).AtMapKey("max"), knownvalue.StringExact("5")),
					},
				},
			},
		})
	})
}

func TestAccDataSourceLKECluster_controlPlane(t *testing.T) {
	t.Parallel()

	acceptance.RunTestWithRetries(t, 2, func(t *acceptance.WrappedT) {
		clusterName := acctest.RandomWithPrefix("tf_test")
		testIPv4 := "0.0.0.0/0"
		testIPv6 := "2001:db8::/32"

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             acceptance.CheckLKEClusterDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.DataControlPlane(t, clusterName, k8sVersionLatest, testRegion, testIPv4, testIPv6, false, true),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(clusterName)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("k8s_version"), knownvalue.StringExact(k8sVersionLatest)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("status"), knownvalue.StringExact("ready")),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("tags"), knownvalue.SetSizeExact(1)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools").AtSliceIndex(0).AtMapKey("type"), knownvalue.StringExact("g6-standard-2")),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools").AtSliceIndex(0).AtMapKey("count"), knownvalue.StringExact("1")),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools").AtSliceIndex(0).AtMapKey("nodes"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools").AtSliceIndex(0).AtMapKey("autoscaler"), knownvalue.ListSizeExact(0)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("control_plane").AtSliceIndex(0).AtMapKey("high_availability"), knownvalue.Bool(false)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("control_plane").AtSliceIndex(0).AtMapKey("acl").AtSliceIndex(0).AtMapKey("enabled"), knownvalue.Bool(true)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("control_plane").AtSliceIndex(0).AtMapKey("acl").AtSliceIndex(0).AtMapKey("addresses").AtSliceIndex(0).AtMapKey("ipv4").AtSliceIndex(0), knownvalue.StringExact(testIPv4)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("control_plane").AtSliceIndex(0).AtMapKey("acl").AtSliceIndex(0).AtMapKey("addresses").AtSliceIndex(0).AtMapKey("ipv6").AtSliceIndex(0), knownvalue.StringExact(testIPv6)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("kubeconfig"), knownvalue.NotNull()),
					},
				},
			},
		})
	})
}

func TestAccDataSourceLKECluster_enterprise(t *testing.T) {
	t.Parallel()

	enterpriseRegion := "no-osl-1" // currently only oslo region works with BYO VPC

	// TODO: revert to dynamic selection once more regions available
	//enterpriseRegion, err := acceptance.GetRandomRegionWithCaps([]string{"Kubernetes Enterprise", "VPCs"}, "core")
	//if err != nil {
	//	log.Fatal(err)
	//}

	var k8sVersionEnterprise string

	k8sVersionEnterprise = "v1.31.9+lke5" // currently only this version works with BYO VPC

	// TODO: revert to select versions from the k8s versions list once more versions available
	//client, err := acceptance.GetTestClient()
	//
	//enterpriseVersions, err := client.ListLKETierVersions(context.Background(), "enterprise", nil)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//if len(enterpriseVersions) < 1 {
	//	t.Skip("No available k8s version for LKE Enterprise test. Skipping now...")
	//} else {
	//	k8sVersionEnterprise = enterpriseVersions[0].ID
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
					Config: tmpl.DataEnterprise(t, clusterName, k8sVersionEnterprise, enterpriseRegion, "on_recycle", firewall.ID),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("label"), knownvalue.StringExact(clusterName)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("region"), knownvalue.StringExact(enterpriseRegion)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("k8s_version"), knownvalue.StringExact(k8sVersionEnterprise)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("status"), knownvalue.StringExact("ready")),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("tier"), knownvalue.StringExact("enterprise")),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("tags"), knownvalue.SetSizeExact(1)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools").AtSliceIndex(0).AtMapKey("type"), knownvalue.StringExact("g6-standard-1")),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools").AtSliceIndex(0).AtMapKey("firewall_id"), knownvalue.StringExact(fmt.Sprintf("%d", firewall.ID))),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools").AtSliceIndex(0).AtMapKey("count"), knownvalue.StringExact("3")),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools").AtSliceIndex(0).AtMapKey("k8s_version"), knownvalue.StringExact(k8sVersionEnterprise)),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("pools").AtSliceIndex(0).AtMapKey("update_strategy"), knownvalue.StringExact("on_recycle")),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("kubeconfig"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("vpc_id"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("subnet_id"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("stack_type"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(dataSourceClusterName, tfjsonpath.New("control_plane").AtSliceIndex(0).AtMapKey("audit_logs_enabled"), knownvalue.NotNull()),
					},
				},
			},
		})
	})

	client.DeleteFirewall(context.Background(), firewall.ID)
}
