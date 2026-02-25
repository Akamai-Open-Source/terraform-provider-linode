//go:build integration || nbs

package nbs_test

import (
	"log"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/nbs/tmpl"
)

func TestAccDataSourceNodeBalancers_basic(t *testing.T) {
	t.Parallel()

	resourceName := "data.linode_nodebalancers.nbs"

	nbLabel := acctest.RandomWithPrefix("tf_test")
	nbRegion, err := acceptance.GetRandomRegionWithCaps([]string{"NodeBalancers"}, "core")
	if err != nil {
		log.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t, nbLabel, nbRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancers").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancers").AtSliceIndex(0).AtMapKey("client_conn_throttle"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancers").AtSliceIndex(0).AtMapKey("client_udp_sess_throttle"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancers").AtSliceIndex(0).AtMapKey("created"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancers").AtSliceIndex(0).AtMapKey("hostname"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancers").AtSliceIndex(0).AtMapKey("ipv4"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancers").AtSliceIndex(0).AtMapKey("ipv6"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancers").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact(nbLabel+"-0")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancers").AtSliceIndex(0).AtMapKey("region"), knownvalue.StringExact(nbRegion)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancers").AtSliceIndex(0).AtMapKey("transfer"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancers").AtSliceIndex(0).AtMapKey("transfer").AtSliceIndex(0).AtMapKey("in"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancers").AtSliceIndex(0).AtMapKey("transfer").AtSliceIndex(0).AtMapKey("out"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancers").AtSliceIndex(0).AtMapKey("transfer").AtSliceIndex(0).AtMapKey("total"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancers").AtSliceIndex(0).AtMapKey("tags"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancers").AtSliceIndex(0).AtMapKey("tags").AtSliceIndex(0), knownvalue.StringExact("tf_test_1")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancers").AtSliceIndex(0).AtMapKey("updated"), knownvalue.NotNull()),
				},
			},
			{
				Config: tmpl.DataFilter(t, nbLabel, nbRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancers"), knownvalue.ListSizeExact(1)),
				},
			},
			{
				Config: tmpl.DataFilterEmpty(t, nbLabel, nbRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancers"), knownvalue.ListSizeExact(0)),
				},
			},
			{
				Config: tmpl.DataFilterTags(t, nbLabel, nbRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancers"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancers").AtSliceIndex(0).AtMapKey("region"), knownvalue.StringExact(nbRegion)),
					acceptance.StateCheckListContains(resourceName, "nodebalancers.0.tags", "tf_test_2"),
				},
			},
			{
				Config: tmpl.DataOrder(t, nbLabel, nbRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancers").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact(nbLabel+"-0")),
				},
			},
		},
	})
}
