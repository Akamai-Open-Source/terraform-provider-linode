//go:build integration || ipv6ranges

package ipv6ranges_test

import (
	"log"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/ipv6ranges/tmpl"
)

var testRegion string

func init() {
	region, err := acceptance.GetRandomRegionWithCaps([]string{"Linodes"}, "core")
	if err != nil {
		log.Fatal(err)
	}

	testRegion = region
}

func TestAccDataSourceIPv6Ranges_basic(t *testing.T) {
	t.Parallel()

	instanceLabel := acctest.RandomWithPrefix("tf_test")
	dataSourceName := "data.linode_ipv6_ranges.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t, instanceLabel, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("ranges"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("ranges").AtSliceIndex(0).AtMapKey("range"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("ranges").AtSliceIndex(0).AtMapKey("route_target"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("ranges").AtSliceIndex(0).AtMapKey("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("ranges").AtSliceIndex(0).AtMapKey("prefix"), knownvalue.Int64Exact(64)),
				},
			},
		},
	})
}
