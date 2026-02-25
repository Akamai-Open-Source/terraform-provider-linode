//go:build integration || rdns

package rdns_test

import (
	"context"
	"fmt"
	"log"
	"regexp"
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
	"github.com/linode/terraform-provider-linode/v3/linode/rdns/tmpl"
)

var testRegion string

func init() {
	resource.AddTestSweepers("linode_rdns", &resource.Sweeper{
		Name: "linode_rdns",
		F:    sweep,
	})

	region, err := acceptance.GetRandomRegionWithCaps([]string{"linodes"}, "core")
	if err != nil {
		log.Fatal(err)
	}

	testRegion = region
}

func sweep(prefix string) error {
	client, err := acceptance.GetTestClient()
	if err != nil {
		return fmt.Errorf("Error getting client: %s", err)
	}

	ips, err := client.ListIPAddresses(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("Error getting IPAddresses: %s", err)
	}
	updateOpts := linodego.IPAddressUpdateOptions{RDNS: nil}
	for _, ip := range ips {
		if !acceptance.ShouldSweep(prefix, ip.RDNS) {
			continue
		}
		_, err := client.UpdateIPAddress(context.Background(), ip.Address, updateOpts)
		if err != nil {
			return fmt.Errorf("Error clearing RDNS %s during sweep: %s", ip.RDNS, err)
		}
	}

	return nil
}

func TestAccResourceRDNS_basic(t *testing.T) {
	t.Parallel()

	resName := "linode_rdns.foobar"
	linodeLabel := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkRDNSDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, linodeLabel, testRegion, false),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckRDNSExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("rdns"), knownvalue.StringRegexp(regexp.MustCompile(`.nip.io$`))),
				},
			},
			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"wait_for_available", "firewall_id"},
			},
		},
	})
}

func TestAccResourceRDNS_update(t *testing.T) {
	t.Parallel()

	label := acctest.RandomWithPrefix("tf_test")
	resName := "linode_rdns.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkRDNSDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, label, testRegion, false),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckRDNSExists(),
					statecheck.CompareValuePairs(resName, tfjsonpath.New("address"), "linode_instance.foobar", tfjsonpath.New("ip_address"), compare.ValuesSame()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("rdns"), knownvalue.StringRegexp(regexp.MustCompile(`([0-9]{1,3}\.){4}nip.io$`))),
				},
			},
			{
				Config: tmpl.Changed(t, label, testRegion, false),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckRDNSExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("rdns"), knownvalue.StringRegexp(regexp.MustCompile(`([0-9]{1,3}\-){3}[0-9]{1,3}.nip.io$`))),
				},
			},
			{
				Config: tmpl.Deleted(t, label, testRegion),
			},
			{
				Config: tmpl.Deleted(t, label, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.linode_networking_ip.foobar", tfjsonpath.New("rdns"), knownvalue.StringRegexp(regexp.MustCompile(`.ip.linodeusercontent.com$`))),
				},
			},
		},
	})
}

// This test case simply ensures a
func TestAccResourceRDNS_waitForAvailable(t *testing.T) {
	t.Parallel()

	label := acctest.RandomWithPrefix("tf_test")
	resName := "linode_rdns.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkRDNSDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, label, testRegion, true),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckRDNSExists(),
					statecheck.CompareValuePairs(resName, tfjsonpath.New("address"), "linode_instance.foobar", tfjsonpath.New("ip_address"), compare.ValuesSame()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("rdns"), knownvalue.StringRegexp(regexp.MustCompile(`([0-9]{1,3}\.){4}nip.io$`))),
				},
			},
			{
				Config: tmpl.Changed(t, label, testRegion, true),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckRDNSExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("rdns"), knownvalue.StringRegexp(regexp.MustCompile(`([0-9]{1,3}\-){3}[0-9]{1,3}.nip.io$`))),
				},
			},
			{
				Config: tmpl.Deleted(t, label, testRegion),
			},
			{
				Config: tmpl.Deleted(t, label, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.linode_networking_ip.foobar", tfjsonpath.New("rdns"), knownvalue.StringRegexp(regexp.MustCompile(`.ip.linodeusercontent.com$`))),
				},
			},
		},
	})
}

func TestAccResourceRDNS_waitForAvailableWithTimeout(t *testing.T) {
	t.Parallel()

	resName := "linode_rdns.foobar"
	linodeLabel := acctest.RandomWithPrefix("tf_test")

	createTimeout := "15m"
	updateTimeout := "15m"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkRDNSDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.WithTimeout(t, linodeLabel, testRegion, createTimeout, updateTimeout),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckRDNSExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("rdns"), knownvalue.StringRegexp(regexp.MustCompile(`.nip.io$`))),
				},
			},
			{
				Config: tmpl.WithTimeoutUpdated(t, linodeLabel, testRegion, createTimeout, updateTimeout),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckRDNSExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("rdns"), knownvalue.StringRegexp(regexp.MustCompile(`([0-9]{1,3}\-){3}[0-9]{1,3}.nip.io$`))),
				},
			},
			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"wait_for_available", "firewall_id"},
			},
		},
	})
}

func stateCheckRDNSExists() statecheck.StateCheck {
	return acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		client := acceptance.TestAccFrameworkProvider.Meta.Client

		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Type != "linode_rdns" {
				continue
			}

			address, ok := rc.AttributeValues["address"]
			if !ok {
				resp.Error = fmt.Errorf("No address attribute found for RDNS resource")
				return
			}

			addrStr, ok := address.(string)
			if !ok {
				resp.Error = fmt.Errorf("address is not a string")
				return
			}

			_, err := client.GetIPAddress(context.Background(), addrStr)
			if err != nil {
				rdns := rc.AttributeValues["rdns"]
				resp.Error = fmt.Errorf("Error retrieving state of RDNS %s: %s", rdns, err)
				return
			}
		}
	})
}

func checkRDNSDestroy(s *terraform.State) error {
	client := acceptance.TestAccFrameworkProvider.Meta.Client
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_rdns" {
			continue
		}

		id := rs.Primary.ID
		ip, err := client.GetIPAddress(context.Background(), id)
		if err != nil {
			if apiErr, ok := err.(*linodego.Error); ok && apiErr.Code == 404 {
				return nil
			}

			if ip.RDNS[len(ip.RDNS)-len("ip.linodeusercontent.com"):] == "ip.linodeusercontent.com" {
				return nil
			}

			return fmt.Errorf("Linode RDNS with IP %s still exists: %s", id, err)
		}
	}

	return nil
}
