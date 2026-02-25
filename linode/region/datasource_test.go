//go:build integration || region

package region_test

import (
	"context"
	"log"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/region/tmpl"
)

var (
	testRegion string
	testLabel  string
)

func init() {
	region, err := acceptance.GetRandomRegionWithCaps([]string{"linodes"}, "core")
	if err != nil {
		log.Fatal(err)
	}

	testRegion = region

	client, err := acceptance.GetTestClient()
	if err != nil {
		log.Fatal(err)
	}

	r, err := client.GetRegion(context.Background(), testRegion)
	if err != nil {
		log.Fatal(err)
	}

	testLabel = r.Label
}

func TestAccDataSourceRegion_basic(t *testing.T) {
	t.Parallel()

	regionID := testRegion
	label := testLabel
	resourceName := "data.linode_region.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t, regionID, label),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("country"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("id"), knownvalue.StringExact(regionID)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("label"), knownvalue.StringExact(label)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("status"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("site_type"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("resolvers").AtSliceIndex(0).AtMapKey("ipv4"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("resolvers").AtSliceIndex(0).AtMapKey("ipv6"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("placement_group_limits").AtSliceIndex(0).AtMapKey("maximum_pgs_per_customer"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("placement_group_limits").AtSliceIndex(0).AtMapKey("maximum_linodes_per_pg"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("monitors").AtMapKey("alerts").AtSliceIndex(0), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("monitors").AtMapKey("metrics").AtSliceIndex(0), knownvalue.NotNull()),
					acceptance.StateCheckResourceAttrGreaterThan(resourceName, "capabilities.#", 0),
				},
			},
		},
	})
}
