//go:build integration || networkingip

package networkingips_test

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/linodego"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/networkingips/tmpl"
)

var testRegion string

func init() {
	region, err := acceptance.GetRandomRegionWithCaps([]string{linodego.CapabilityLinodes}, "core")
	if err != nil {
		log.Fatal(err)
	}

	testRegion = region
}

func TestAccDataSourceNetworkingIP_list(t *testing.T) {
	t.Parallel()

	dataResourceName := "data.linode_networking_ips.list"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataList(t),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataResourceName, tfjsonpath.New("ip_addresses"), knownvalue.NotNull()),
					acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
						for _, r := range req.State.Values.RootModule.Resources {
							if r.Address == dataResourceName {
								ipAddressesRaw, ok := r.AttributeValues["ip_addresses"]
								if !ok || ipAddressesRaw == nil {
									resp.Error = fmt.Errorf("ip_addresses attribute not found for %s", dataResourceName)
									return
								}

								ipAddresses, ok := ipAddressesRaw.([]interface{})
								if !ok {
									resp.Error = fmt.Errorf("ip_addresses is not a list")
									return
								}

								for _, ipRaw := range ipAddresses {
									ip, ok := ipRaw.(map[string]interface{})
									if !ok {
										continue
									}

									// Check if all required fields are set (non-nil and non-empty for strings)
									gateway, _ := ip["gateway"].(string)
									rdns, _ := ip["rdns"].(string)
									address, _ := ip["address"].(string)
									region, _ := ip["region"].(string)
									ipType, _ := ip["type"].(string)
									subnetMask, _ := ip["subnet_mask"].(string)

									linodeID := ip["linode_id"]
									public := ip["public"]
									prefix := ip["prefix"]
									reserved := ip["reserved"]

									if gateway != "" &&
										rdns != "" &&
										address != "" &&
										linodeID != nil &&
										region != "" &&
										ipType != "" &&
										public != nil &&
										prefix != nil &&
										subnetMask != "" &&
										reserved != nil {

										// Perform assertions for the selected IP address
										if !regexp.MustCompile(`\.1$`).MatchString(gateway) {
											resp.Error = fmt.Errorf("attribute gateway has invalid value: %s", gateway)
											return
										}

										if !regexp.MustCompile(`.ip.linodeusercontent.com$`).MatchString(rdns) {
											resp.Error = fmt.Errorf("attribute rdns has invalid value: %s", rdns)
											return
										}

										return // Success - found an IP with all fields set and valid
									}
								}

								resp.Error = fmt.Errorf("no IP address found with all attributes set")
								return
							}
						}

						resp.Error = fmt.Errorf("resource not found: %s", dataResourceName)
					}),
				},
			},
		},
	})
}

func TestAccDataSourceNetworkingIP_filterReserved(t *testing.T) {
	t.Parallel()

	dataResourceName := "data.linode_networking_ips.filtered"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataFilterReserved(t),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataResourceName, tfjsonpath.New("ip_addresses"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dataResourceName, tfjsonpath.New("ip_addresses").AtSliceIndex(0).AtMapKey("reserved"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(dataResourceName, tfjsonpath.New("ip_addresses").AtSliceIndex(0).AtMapKey("address"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dataResourceName, tfjsonpath.New("ip_addresses").AtSliceIndex(0).AtMapKey("linode_id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dataResourceName, tfjsonpath.New("ip_addresses").AtSliceIndex(0).AtMapKey("interface_id"), knownvalue.Null()),
					statecheck.ExpectKnownValue(dataResourceName, tfjsonpath.New("ip_addresses").AtSliceIndex(0).AtMapKey("region"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(
						dataResourceName,
						tfjsonpath.New("ip_addresses").AtSliceIndex(0).AtMapKey("gateway"),
						knownvalue.StringRegexp(regexp.MustCompile(`\.1$`)),
					),
					statecheck.ExpectKnownValue(
						dataResourceName,
						tfjsonpath.New("ip_addresses").AtSliceIndex(0).AtMapKey("type"),
						knownvalue.StringExact("ipv4"),
					),
					statecheck.ExpectKnownValue(dataResourceName, tfjsonpath.New("ip_addresses").AtSliceIndex(0).AtMapKey("public"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(dataResourceName, tfjsonpath.New("ip_addresses").AtSliceIndex(0).AtMapKey("prefix"), knownvalue.Int64Exact(24)),
					statecheck.ExpectKnownValue(dataResourceName, tfjsonpath.New("ip_addresses").AtSliceIndex(0).AtMapKey("subnet_mask"), knownvalue.NotNull()),
				},
			},
		},
	})
}
