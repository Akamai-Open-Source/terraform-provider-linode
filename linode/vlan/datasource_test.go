//go:build integration || vlan

package vlan_test

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/linodego"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/helper"
	"github.com/linode/terraform-provider-linode/v3/linode/vlan/tmpl"
)

var testRegion string

func init() {
	region, err := acceptance.GetRandomRegionWithCaps([]string{"vlans"}, "core")
	if err != nil {
		log.Fatal(err)
	}

	testRegion = region
}

func preConfigVLANPoll(t testing.TB, vlanName string) func() {
	return func() {
		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client
		if _, err := waitForVLANWithLabel(client, vlanName, 30); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAccDataSourceVLANs_basic(t *testing.T) {
	t.Parallel()

	instanceName := acctest.RandomWithPrefix("tf_test")
	vlanName := "tf-test"
	resourceName := "data.linode_vlans.foolan"
	label := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t, instanceName, testRegion, vlanName, label),
			},
			{
				PreConfig: preConfigVLANPoll(t, vlanName),
				Config:    tmpl.DataBasic(t, instanceName, testRegion, vlanName, label),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("vlans").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact(vlanName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("vlans").AtSliceIndex(0).AtMapKey("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("vlans").AtSliceIndex(0).AtMapKey("created"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("vlans").AtSliceIndex(0).AtMapKey("linodes"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func TestAccDataSourceVLANs_regex(t *testing.T) {
	t.Parallel()

	instanceName := acctest.RandomWithPrefix("tf_test")
	vlanName := "tf-test"
	resourceName := "data.linode_vlans.foolan"
	label := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: tmpl.DataRegex(t, instanceName, testRegion, vlanName, label),
			},
			{
				PreConfig: preConfigVLANPoll(t, vlanName),
				Config:    tmpl.DataRegex(t, instanceName, testRegion, vlanName, label),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckResourceAttrGreaterThan(resourceName, "vlans.#", 0),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("vlans").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact(vlanName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("vlans").AtSliceIndex(0).AtMapKey("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("vlans").AtSliceIndex(0).AtMapKey("created"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("vlans").AtSliceIndex(0).AtMapKey("linodes"), knownvalue.NotNull()),
				},
			},
		},
	})
}

// waitForVLANWithLabel polls for a VLAN with the given label to exist
// This is necessary in this context because it is not guaranteed that a VLAN will
// be created immediately after the instance is created.
func waitForVLANWithLabel(client linodego.Client, label string, timeoutSeconds int) (*linodego.VLAN, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(timeoutSeconds))
	defer cancel()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			vlans, err := client.ListVLANs(ctx,
				&linodego.ListOptions{Filter: fmt.Sprintf("{\"label\": \"%s\"}", label)})
			if err != nil {
				return nil, err
			}

			if len(vlans) > 0 {
				return &vlans[0], err
			}

		case <-ctx.Done():
			return nil, fmt.Errorf("Error waiting for VLAN %s: %s", label, ctx.Err())
		}
	}
}
