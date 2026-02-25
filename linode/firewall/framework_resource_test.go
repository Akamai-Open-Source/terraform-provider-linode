//go:build integration || firewall

package firewall_test

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
	acceptanceTmpl "github.com/linode/terraform-provider-linode/v3/linode/acceptance/tmpl"
	"github.com/linode/terraform-provider-linode/v3/linode/firewall/tmpl"
	"github.com/linode/terraform-provider-linode/v3/linode/helper"
)

const testFirewallResName = "linode_firewall.test"

var testRegion string

func init() {
	resource.AddTestSweepers("linode_firewall", &resource.Sweeper{
		Name: "linode_firewall",
		F:    sweep,
	})

	region, err := acceptance.GetRandomRegionWithCaps([]string{"Cloud Firewall", "NodeBalancers"}, "core")
	if err != nil {
		log.Fatal(err)
	}

	testRegion = region
}

func sweep(prefix string) error {
	client, err := acceptance.GetTestClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %s", err)
	}

	firewalls, err := client.ListFirewalls(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("failed to get firewalls: %s", err)
	}
	for _, firewall := range firewalls {
		if !acceptance.ShouldSweep(prefix, firewall.Label) {
			continue
		}
		if err := client.DeleteFirewall(context.Background(), firewall.ID); err != nil {
			return fmt.Errorf("failed to destroy firewall %d during sweep: %s", firewall.ID, err)
		}
	}

	return nil
}

// TODO: Add a test case for interfaces when interfaces resource is implemented.

func TestSmokeTests_firewall(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{"TestAccLinodeFirewall_basic", TestAccLinodeFirewall_basic},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestAccLinodeFirewall_basic(t *testing.T) {
	t.Parallel()

	name := acctest.RandomWithPrefix("tf_test")
	devicePrefix := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acceptanceTmpl.ProviderNoPoll(t) + tmpl.Basic(t, name, devicePrefix, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("label"), knownvalue.StringExact(name)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("disabled"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound_policy"), knownvalue.StringExact("DROP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("action"), knownvalue.StringExact("ACCEPT")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("protocol"), knownvalue.StringExact("TCP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ports"), knownvalue.StringExact("80")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv4"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv4").AtSliceIndex(0), knownvalue.StringExact("0.0.0.0/0")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv6"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv6").AtSliceIndex(0), knownvalue.StringExact("::/0")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound_policy"), knownvalue.StringExact("DROP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("protocol"), knownvalue.StringExact("TCP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ports"), knownvalue.StringExact("80")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ipv4"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ipv4").AtSliceIndex(0), knownvalue.StringExact("0.0.0.0/0")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ipv6"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ipv6").AtSliceIndex(0), knownvalue.StringExact("2001:db8::/32")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("devices"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("devices").AtSliceIndex(0).AtMapKey("type"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("linodes"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("nodebalancers"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("tags"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("tags").AtSliceIndex(0), knownvalue.StringExact("test")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("devices").AtSliceIndex(0).AtMapKey("url"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("devices").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("devices").AtSliceIndex(0).AtMapKey("entity_id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("devices").AtSliceIndex(0).AtMapKey("label"), knownvalue.NotNull()),
				},
			},
			{
				ResourceName:            testFirewallResName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created", "updated"},
			},
		},
	})
}

func TestAccLinodeFirewall_minimum(t *testing.T) {
	t.Parallel()

	name := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acceptanceTmpl.ProviderNoPoll(t) + tmpl.Minimum(t, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("label"), knownvalue.StringExact(name)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("disabled"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("protocol"), knownvalue.StringExact("TCP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv4"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv4").AtSliceIndex(0), knownvalue.StringExact("0.0.0.0/0")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv6"), knownvalue.ListSizeExact(0)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound"), knownvalue.ListSizeExact(0)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("devices"), knownvalue.ListSizeExact(0)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("linodes"), knownvalue.SetSizeExact(0)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("tags"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("tags").AtSliceIndex(0), knownvalue.StringExact("test")),
				},
			},
			{
				ResourceName:            testFirewallResName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created", "updated"},
			},
		},
	})
}

func TestAccLinodeFirewall_multipleRules(t *testing.T) {
	t.Parallel()

	name := acctest.RandomWithPrefix("tf_test")
	devicePrefix := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.MultipleRules(t, name, devicePrefix, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("label"), knownvalue.StringExact(name)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("disabled"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound_policy"), knownvalue.StringExact("DROP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound"), knownvalue.ListSizeExact(2)),

					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("action"), knownvalue.StringExact("ACCEPT")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("protocol"), knownvalue.StringExact("TCP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ports"), knownvalue.StringExact("80")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv4"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv4").AtSliceIndex(0), knownvalue.StringExact("0.0.0.0/0")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv6"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv6").AtSliceIndex(0), knownvalue.StringExact("::/0")),

					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(1).AtMapKey("action"), knownvalue.StringExact("ACCEPT")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(1).AtMapKey("protocol"), knownvalue.StringExact("TCP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(1).AtMapKey("ports"), knownvalue.StringExact("443")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(1).AtMapKey("ipv4"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(1).AtMapKey("ipv4").AtSliceIndex(0), knownvalue.StringExact("0.0.0.0/0")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(1).AtMapKey("ipv6"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(1).AtMapKey("ipv6").AtSliceIndex(0), knownvalue.StringExact("::/0")),

					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound_policy"), knownvalue.StringExact("DROP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound"), knownvalue.ListSizeExact(2)),

					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("protocol"), knownvalue.StringExact("TCP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ports"), knownvalue.StringExact("80")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ipv4"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ipv4").AtSliceIndex(0), knownvalue.StringExact("0.0.0.0/0")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ipv6"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ipv6").AtSliceIndex(0), knownvalue.StringExact("2001:db8::/32")),

					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(1).AtMapKey("protocol"), knownvalue.StringExact("TCP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(1).AtMapKey("ports"), knownvalue.StringExact("443")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(1).AtMapKey("ipv4"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(1).AtMapKey("ipv4").AtSliceIndex(0), knownvalue.StringExact("0.0.0.0/0")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(1).AtMapKey("ipv6"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(1).AtMapKey("ipv6").AtSliceIndex(0), knownvalue.StringExact("2001:db8::/32")),

					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("devices"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("linodes"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("tags"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("tags").AtSliceIndex(0), knownvalue.StringExact("test")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("devices").AtSliceIndex(0).AtMapKey("url"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("devices").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("devices").AtSliceIndex(0).AtMapKey("entity_id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("devices").AtSliceIndex(0).AtMapKey("label"), knownvalue.NotNull()),
				},
			},
			{
				ResourceName:            testFirewallResName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created", "updated"},
			},
		},
	})
}

func TestAccLinodeFirewall_no_device(t *testing.T) {
	t.Parallel()

	name := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acceptanceTmpl.ProviderNoPoll(t) + tmpl.NoDevice(t, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("label"), knownvalue.StringExact(name)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("disabled"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("protocol"), knownvalue.StringExact("TCP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ports"), knownvalue.StringExact("80")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv6").AtSliceIndex(0), knownvalue.StringExact("::/0")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("protocol"), knownvalue.StringExact("TCP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ports"), knownvalue.StringExact("80")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ipv6").AtSliceIndex(0), knownvalue.StringExact("::/0")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("devices"), knownvalue.ListSizeExact(0)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("linodes"), knownvalue.SetSizeExact(0)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("tags"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("tags").AtSliceIndex(0), knownvalue.StringExact("test")),
				},
			},
			{
				ResourceName:            testFirewallResName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created", "updated"},
			},
		},
	})
}

func TestAccLinodeFirewall_updates(t *testing.T) {
	t.Parallel()

	name := acctest.RandomWithPrefix("tf_test")
	newName := acctest.RandomWithPrefix("tf_test")
	devicePrefix := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acceptanceTmpl.ProviderNoPoll(t) + tmpl.Basic(t, name, devicePrefix, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("label"), knownvalue.StringExact(name)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("disabled"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound_policy"), knownvalue.StringExact("DROP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("action"), knownvalue.StringExact("ACCEPT")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("protocol"), knownvalue.StringExact("TCP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ports"), knownvalue.StringExact("80")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv4"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv4").AtSliceIndex(0), knownvalue.StringExact("0.0.0.0/0")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv6"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv6").AtSliceIndex(0), knownvalue.StringExact("::/0")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound_policy"), knownvalue.StringExact("DROP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("action"), knownvalue.StringExact("ACCEPT")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("protocol"), knownvalue.StringExact("TCP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ports"), knownvalue.StringExact("80")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ipv4"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ipv4").AtSliceIndex(0), knownvalue.StringExact("0.0.0.0/0")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ipv6"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ipv6").AtSliceIndex(0), knownvalue.StringExact("2001:db8::/32")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("devices"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("devices").AtSliceIndex(0).AtMapKey("type"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("linodes"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("nodebalancers"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("tags"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("tags").AtSliceIndex(0), knownvalue.StringExact("test")),
				},
			},
			{
				Config: acceptanceTmpl.ProviderNoPoll(t) + tmpl.Updates(t, newName, devicePrefix, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("label"), knownvalue.StringExact(newName)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("disabled"), knownvalue.StringExact("true")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound_policy"), knownvalue.StringExact("ACCEPT")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound"), knownvalue.ListSizeExact(3)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("action"), knownvalue.StringExact("DROP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("protocol"), knownvalue.StringExact("TCP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ports"), knownvalue.StringExact("80")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv4"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv4").AtSliceIndex(0), knownvalue.StringExact("0.0.0.0/0")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv6"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv6").AtSliceIndex(0), knownvalue.StringExact("::/0")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv6").AtSliceIndex(1), knownvalue.StringExact("ff00::/8")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(1).AtMapKey("action"), knownvalue.StringExact("DROP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(1).AtMapKey("protocol"), knownvalue.StringExact("TCP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(1).AtMapKey("ports"), knownvalue.StringExact("443")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(1).AtMapKey("ipv4"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(1).AtMapKey("ipv4").AtSliceIndex(0), knownvalue.StringExact("0.0.0.0/0")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(1).AtMapKey("ipv4").AtSliceIndex(1), knownvalue.StringExact("127.0.0.1/32")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(1).AtMapKey("ipv6"), knownvalue.ListSizeExact(0)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(2).AtMapKey("action"), knownvalue.StringExact("DROP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(2).AtMapKey("protocol"), knownvalue.StringExact("TCP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(2).AtMapKey("ports"), knownvalue.StringExact("22")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(2).AtMapKey("ipv4"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(2).AtMapKey("ipv4").AtSliceIndex(0), knownvalue.StringExact("0.0.0.0/0")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(2).AtMapKey("ipv6"), knownvalue.ListSizeExact(0)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound_policy"), knownvalue.StringExact("ACCEPT")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound"), knownvalue.ListSizeExact(0)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("linodes"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("nodebalancers"), knownvalue.SetSizeExact(2)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("tags"), knownvalue.SetSizeExact(2)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("tags").AtSliceIndex(0), knownvalue.StringExact("test")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("tags").AtSliceIndex(1), knownvalue.StringExact("test2")),
				},
			},
		},
	})
}

func TestAccLinodeFirewall_externalDelete(t *testing.T) {
	t.Parallel()

	var firewall linodego.Firewall
	name := acctest.RandomWithPrefix("tf_test")
	devicePrefix := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acceptanceTmpl.ProviderNoPoll(t) + tmpl.Basic(t, name, devicePrefix, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckFirewallExists(testFirewallResName, &firewall),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("label"), knownvalue.StringExact(name)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("disabled"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound_policy"), knownvalue.StringExact("DROP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("action"), knownvalue.StringExact("ACCEPT")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("protocol"), knownvalue.StringExact("TCP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ports"), knownvalue.StringExact("80")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv4"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv4").AtSliceIndex(0), knownvalue.StringExact("0.0.0.0/0")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv6"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv6").AtSliceIndex(0), knownvalue.StringExact("::/0")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound_policy"), knownvalue.StringExact("DROP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("action"), knownvalue.StringExact("ACCEPT")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("protocol"), knownvalue.StringExact("TCP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ports"), knownvalue.StringExact("80")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ipv4"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ipv4").AtSliceIndex(0), knownvalue.StringExact("0.0.0.0/0")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ipv6"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ipv6").AtSliceIndex(0), knownvalue.StringExact("2001:db8::/32")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("devices"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("linodes"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("tags"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("tags").AtSliceIndex(0), knownvalue.StringExact("test")),
				},
			},
			{
				PreConfig: func() {
					// Delete the Firewall external from Terraform
					client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

					if err := client.DeleteFirewall(context.Background(), firewall.ID); err != nil {
						t.Fatalf("failed to delete firewall: %s", err)
					}
				},
				Config: acceptanceTmpl.ProviderNoPoll(t) + tmpl.Basic(t, name, devicePrefix, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckFirewallExists(testFirewallResName, &firewall),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("label"), knownvalue.StringExact(name)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("disabled"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound_policy"), knownvalue.StringExact("DROP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("action"), knownvalue.StringExact("ACCEPT")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("protocol"), knownvalue.StringExact("TCP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ports"), knownvalue.StringExact("80")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv4"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv4").AtSliceIndex(0), knownvalue.StringExact("0.0.0.0/0")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv6"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv6").AtSliceIndex(0), knownvalue.StringExact("::/0")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound_policy"), knownvalue.StringExact("DROP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("action"), knownvalue.StringExact("ACCEPT")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("protocol"), knownvalue.StringExact("TCP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ports"), knownvalue.StringExact("80")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ipv4"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ipv4").AtSliceIndex(0), knownvalue.StringExact("0.0.0.0/0")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ipv6"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound").AtSliceIndex(0).AtMapKey("ipv6").AtSliceIndex(0), knownvalue.StringExact("2001:db8::/32")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("devices"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("linodes"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("nodebalancers"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("tags"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("tags").AtSliceIndex(0), knownvalue.StringExact("test")),
				},
			},
		},
	})
}

func TestAccLinodeFirewall_noIPv6(t *testing.T) {
	t.Parallel()

	name := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acceptanceTmpl.ProviderNoPoll(t) + tmpl.NoIPv6(t, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("label"), knownvalue.StringExact(name)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("disabled"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("protocol"), knownvalue.StringExact("TCP")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ports"), knownvalue.StringExact("80")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound").AtSliceIndex(0).AtMapKey("ipv6"), knownvalue.ListSizeExact(0)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound"), knownvalue.ListSizeExact(0)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("devices"), knownvalue.ListSizeExact(0)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("linodes"), knownvalue.SetSizeExact(0)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("tags"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("tags").AtSliceIndex(0), knownvalue.StringExact("test")),
				},
			},
			{
				ResourceName:            testFirewallResName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created", "updated"},
			},
		},
	})
}

func TestAccLinodeFirewall_noRules(t *testing.T) {
	t.Parallel()

	name := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acceptanceTmpl.ProviderNoPoll(t) + tmpl.NoRules(t, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("label"), knownvalue.StringExact(name)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("disabled"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("inbound"), knownvalue.ListSizeExact(0)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("outbound"), knownvalue.ListSizeExact(0)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("devices"), knownvalue.ListSizeExact(0)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("linodes"), knownvalue.SetSizeExact(0)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("tags"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(testFirewallResName, tfjsonpath.New("tags").AtSliceIndex(0), knownvalue.StringExact("test")),
				},
			},
			{
				ResourceName:            testFirewallResName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created", "updated"},
			},
		},
	})
}
