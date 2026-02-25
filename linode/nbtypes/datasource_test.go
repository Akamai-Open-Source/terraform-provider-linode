//go:build integration || nbtypes

package nbtypes_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/nbtypes/tmpl"
)

func TestAccDataSourceNodeBalancerTypes_basic(t *testing.T) {
	t.Parallel()

	dataSourceName := "data.linode_nb_types.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("types"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("types").AtSliceIndex(0).AtMapKey("id"), knownvalue.StringExact("nodebalancer")),
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("types").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("NodeBalancer")),
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("types").AtSliceIndex(0).AtMapKey("transfer"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("types").AtSliceIndex(0).AtMapKey("price").AtSliceIndex(0).AtMapKey("hourly"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("types").AtSliceIndex(0).AtMapKey("price").AtSliceIndex(0).AtMapKey("monthly"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("types").AtSliceIndex(0).AtMapKey("region_prices").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("types").AtSliceIndex(0).AtMapKey("region_prices").AtSliceIndex(0).AtMapKey("hourly"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("types").AtSliceIndex(0).AtMapKey("region_prices").AtSliceIndex(0).AtMapKey("monthly"), knownvalue.NotNull()),
				},
			},
		},
	})
}
