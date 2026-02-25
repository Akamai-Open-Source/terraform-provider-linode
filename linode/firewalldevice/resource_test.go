//go:build integration || firewalldevice

package firewalldevice_test

import (
	"fmt"
	"log"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/linodego"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	acceptanceTmpl "github.com/linode/terraform-provider-linode/v3/linode/acceptance/tmpl"
	"github.com/linode/terraform-provider-linode/v3/linode/firewalldevice/tmpl"
)

var testRegion string

func init() {
	region, err := acceptance.GetRandomRegionWithCaps([]string{"Cloud Firewall", "NodeBalancers"}, "core")
	if err != nil {
		log.Fatal(err)
	}

	testRegion = region
}

func TestSmokeTests_firewalldevice(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{"TestAccResourceFirewallDevice_basic_smoke", TestAccResourceFirewallDevice_basic_smoke},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestAccResourceFirewallDevice_basic_smoke(t *testing.T) {
	t.Parallel()

	var firewall linodego.Firewall

	firewallName := "linode_firewall.foobar"
	instanceName := "linode_instance.foobar"
	deviceName := "linode_firewall_device.foobar"

	label := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreventPostDestroyRefresh: true,
		PreCheck:                  func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories:  acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acceptanceTmpl.ProviderNoPoll(t) + tmpl.Basic(t, label, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckFirewallExists(firewallName, &firewall),
					statecheck.ExpectKnownValue(deviceName, tfjsonpath.New("created"), knownvalue.NotNull()),
				},
			},
			// Refresh the state and verify the attachment
			{
				Config: acceptanceTmpl.ProviderNoPoll(t) + tmpl.Basic(t, label, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckFirewallExists(firewallName, &firewall),
					statecheck.ExpectKnownValue(firewallName, tfjsonpath.New("devices"), knownvalue.ListSizeExact(1)),
					statecheck.CompareValuePairs(firewallName, tfjsonpath.New("linodes").AtSliceIndex(0), instanceName, tfjsonpath.New("id"), compare.ValuesSame()),
				},
			},
			{
				ResourceName:      deviceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: resourceImportStateID,
			},
			{
				Config: acceptanceTmpl.ProviderNoPoll(t) + tmpl.Detached(t, label, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckFirewallExists(firewallName, &firewall),
				},
			},
			// Refresh the state and verify the detachment
			{
				Config: acceptanceTmpl.ProviderNoPoll(t) + tmpl.Detached(t, label, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckFirewallExists(firewallName, &firewall),
					statecheck.ExpectKnownValue(firewallName, tfjsonpath.New("devices"), knownvalue.ListSizeExact(0)),
					statecheck.ExpectKnownValue(firewallName, tfjsonpath.New("linodes"), knownvalue.SetSizeExact(0)),
				},
			},
		},
	})
}

func TestAccResourceFirewallDevice_withNodeBalancer(t *testing.T) {
	t.Parallel()

	var firewall linodego.Firewall

	firewallName := "linode_firewall.foobar"
	nodebalancerName := "linode_nodebalancer.foobar"
	deviceName := "linode_firewall_device.foobar"

	label := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreventPostDestroyRefresh: true,
		PreCheck:                  func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories:  acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acceptanceTmpl.ProviderNoPoll(t) + tmpl.WithNodeBalancer(t, label, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckFirewallExists(firewallName, &firewall),
					statecheck.ExpectKnownValue(deviceName, tfjsonpath.New("created"), knownvalue.NotNull()),
				},
			},
			// Refresh the state and verify the attachment
			{
				Config: acceptanceTmpl.ProviderNoPoll(t) + tmpl.WithNodeBalancer(t, label, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckFirewallExists(firewallName, &firewall),
					statecheck.ExpectKnownValue(firewallName, tfjsonpath.New("devices"), knownvalue.ListSizeExact(1)),
					statecheck.CompareValuePairs(firewallName, tfjsonpath.New("nodebalancers").AtSliceIndex(0), nodebalancerName, tfjsonpath.New("id"), compare.ValuesSame()),
				},
			},
			{
				ResourceName:      deviceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: resourceImportStateID,
			},
			{
				Config: acceptanceTmpl.ProviderNoPoll(t) + tmpl.Detached(t, label, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckFirewallExists(firewallName, &firewall),
				},
			},
			// Refresh the state and verify the detachment
			{
				Config: acceptanceTmpl.ProviderNoPoll(t) + tmpl.Detached(t, label, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckFirewallExists(firewallName, &firewall),
					statecheck.ExpectKnownValue(firewallName, tfjsonpath.New("devices"), knownvalue.ListSizeExact(0)),
					statecheck.ExpectKnownValue(firewallName, tfjsonpath.New("linodes"), knownvalue.SetSizeExact(0)),
				},
			},
		},
	})
}

func resourceImportStateID(s *terraform.State) (string, error) {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_firewall_device" {
			continue
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return "", fmt.Errorf("Error parsing ID %v to int", rs.Primary.ID)
		}

		firewallID, err := strconv.Atoi(rs.Primary.Attributes["firewall_id"])
		if err != nil {
			return "", fmt.Errorf("Error parsing firewall_id %v to int", rs.Primary.Attributes["firewall_id"])
		}
		return fmt.Sprintf("%d,%d", firewallID, id), nil
	}

	return "", fmt.Errorf("Error finding firewall_device")
}
