//go:build integration || firewall

package firewall_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/firewall/tmpl"
)

const testFirewallDataName = "data.linode_firewall.test"

// TODO: Add a test case for interfaces when interfaces resource is implemented.
func TestAccDataSourceFirewall_basic(t *testing.T) {
	t.Parallel()

	firewallName := acctest.RandomWithPrefix("tf_test")
	devicePrefix := acctest.RandomWithPrefix("tf_test")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t, firewallName, devicePrefix, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("label"), knownvalue.StringExact(firewallName)),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("disabled"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("inbound_policy"), knownvalue.StringExact("DROP")),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("inbound"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("action"), knownvalue.StringExact("ACCEPT")),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("protocol"), knownvalue.StringExact("TCP")),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ports"), knownvalue.StringExact("80")),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv4"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv4").AtSliceIndex(0), knownvalue.StringExact("0.0.0.0/0")),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv6"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv6").AtSliceIndex(0), knownvalue.StringExact("::/0")),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("outbound_policy"), knownvalue.StringExact("DROP")),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("outbound"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("protocol"), knownvalue.StringExact("TCP")),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ports"), knownvalue.StringExact("80")),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ipv4"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ipv4").AtSliceIndex(0), knownvalue.StringExact("0.0.0.0/0")),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ipv6"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ipv6").AtSliceIndex(0), knownvalue.StringExact("2001:db8::/32")),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("devices"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("devices").AtSliceIndex(0).AtMapKey("type"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("nodebalancers"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("linodes"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("tags"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("tags").AtSliceIndex(0), knownvalue.StringExact("test")),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("devices").AtSliceIndex(0).AtMapKey("url"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("devices").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("devices").AtSliceIndex(0).AtMapKey("entity_id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testFirewallDataName, tfjsonpath.New("devices").AtSliceIndex(0).AtMapKey("label"), knownvalue.NotNull()),
				},
			},
		},
	})
}
