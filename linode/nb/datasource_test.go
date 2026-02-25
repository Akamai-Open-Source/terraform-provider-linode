//go:build integration || nb

package nb_test

import (
	"fmt"
	"log"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/nb/tmpl"
)

func TestAccDataSourceNodeBalancer_basic(t *testing.T) {
	t.Parallel()

	resName := "data.linode_nodebalancer.foobar"
	nodebalancerName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkNodeBalancerDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t, nodebalancerName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodeBalancerExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(nodebalancerName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("client_conn_throttle"), knownvalue.StringExact("20")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("client_udp_sess_throttle"), knownvalue.StringExact("10")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("hostname"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("ipv4"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("ipv6"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("created"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("updated"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("transfer"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("transfer").AtSliceIndex(0).AtMapKey("in"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("transfer").AtSliceIndex(0).AtMapKey("out"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("transfer").AtSliceIndex(0).AtMapKey("total"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("tags"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("tags").AtSliceIndex(0), knownvalue.StringExact("tf_test")),
				},
			},
		},
	})
}

func TestAccDataSourceNodeBalancer_firewalls(t *testing.T) {
	t.Parallel()

	resName := "data.linode_nodebalancer.foobar"
	nodebalancerName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkNodeBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataFirewalls(t, nodebalancerName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodeBalancerExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(nodebalancerName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("client_conn_throttle"), knownvalue.StringExact("20")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("hostname"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("ipv4"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("ipv6"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("created"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("updated"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("transfer"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("transfer").AtSliceIndex(0).AtMapKey("in"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("transfer").AtSliceIndex(0).AtMapKey("out"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("transfer").AtSliceIndex(0).AtMapKey("total"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("tags"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("tags").AtSliceIndex(0), knownvalue.StringExact("tf_test")),
					acceptance.StateCheckResourceAttrGreaterThan(resName, "firewalls.#", 0),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("firewalls").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact(fmt.Sprintf("%v-fw", nodebalancerName))),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("firewalls").AtSliceIndex(0).AtMapKey("inbound_policy"), knownvalue.StringExact("DROP")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("firewalls").AtSliceIndex(0).AtMapKey("inbound"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("firewalls").AtSliceIndex(0).AtMapKey("inbound").AtSliceIndex(0).AtMapKey("action"), knownvalue.StringExact("ACCEPT")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("firewalls").AtSliceIndex(0).AtMapKey("inbound").AtSliceIndex(0).AtMapKey("protocol"), knownvalue.StringExact("TCP")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("firewalls").AtSliceIndex(0).AtMapKey("inbound").AtSliceIndex(0).AtMapKey("ports"), knownvalue.StringExact("80")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("firewalls").AtSliceIndex(0).AtMapKey("inbound").AtSliceIndex(0).AtMapKey("ipv4"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("firewalls").AtSliceIndex(0).AtMapKey("inbound").AtSliceIndex(0).AtMapKey("ipv4").AtSliceIndex(0), knownvalue.StringExact("0.0.0.0/0")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("firewalls").AtSliceIndex(0).AtMapKey("inbound").AtSliceIndex(0).AtMapKey("ipv6"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("firewalls").AtSliceIndex(0).AtMapKey("inbound").AtSliceIndex(0).AtMapKey("ipv6").AtSliceIndex(0), knownvalue.StringExact("::/0")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("firewalls").AtSliceIndex(0).AtMapKey("outbound_policy"), knownvalue.StringExact("DROP")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("firewalls").AtSliceIndex(0).AtMapKey("outbound"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("firewalls").AtSliceIndex(0).AtMapKey("outbound").AtSliceIndex(0).AtMapKey("protocol"), knownvalue.StringExact("TCP")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("firewalls").AtSliceIndex(0).AtMapKey("outbound").AtSliceIndex(0).AtMapKey("ports"), knownvalue.StringExact("80")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("firewalls").AtSliceIndex(0).AtMapKey("outbound").AtSliceIndex(0).AtMapKey("ipv4"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("firewalls").AtSliceIndex(0).AtMapKey("outbound").AtSliceIndex(0).AtMapKey("ipv4").AtSliceIndex(0), knownvalue.StringExact("0.0.0.0/0")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("firewalls").AtSliceIndex(0).AtMapKey("outbound").AtSliceIndex(0).AtMapKey("ipv6"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("firewalls").AtSliceIndex(0).AtMapKey("outbound").AtSliceIndex(0).AtMapKey("ipv6").AtSliceIndex(0), knownvalue.StringExact("2001:db8::/32")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("firewalls").AtSliceIndex(0).AtMapKey("tags"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("firewalls").AtSliceIndex(0).AtMapKey("tags").AtSliceIndex(0), knownvalue.StringExact("test")),
				},
			},
		},
	})
}

func TestAccDataSourceNodeBalancer_vpc(t *testing.T) {
	t.Parallel()

	dsName := "data.linode_nodebalancer.test"
	nodebalancerName := acctest.RandomWithPrefix("tf-test")

	targetRegion, err := acceptance.GetRandomRegionWithCaps([]string{"NodeBalancers", "VPCs"}, "core")
	if err != nil {
		log.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkNodeBalancerDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.DataVPC(t, nodebalancerName, targetRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodeBalancerExists(),
					statecheck.ExpectKnownValue(
						dsName,
						tfjsonpath.New("vpcs").AtSliceIndex(0).AtMapKey("subnet_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						dsName,
						tfjsonpath.New("vpcs").AtSliceIndex(0).AtMapKey("ipv4_range"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}
