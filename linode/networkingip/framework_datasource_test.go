//go:build integration || networkingip

package networkingip_test

import (
	"log"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/networkingip/tmpl"
)

var testRegion string

func init() {
	region, err := acceptance.GetRandomRegionWithCaps([]string{"linodes"}, "core")
	if err != nil {
		log.Fatal(err)
	}

	testRegion = region
}

func TestAccDataSourceNetworkingIP_basic(t *testing.T) {
	t.Parallel()

	resourceName := "linode_instance.foobar"
	dataResourceName := "data.linode_networking_ip.foobar"

	label := acctest.RandomWithPrefix("tf-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t, label, testRegion),
			},
			{
				Config: tmpl.DataBasic(t, label, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					// NOTE: linode_id is an Int64 attribute and id is a String attribute.
					// In the JSON state representation, both values serialize to comparable types,
					// allowing ValuesSame() to function correctly for this comparison.
					statecheck.CompareValuePairs(dataResourceName, tfjsonpath.New("linode_id"), resourceName, tfjsonpath.New("id"), compare.ValuesSame()),
					statecheck.CompareValuePairs(
						dataResourceName,
						tfjsonpath.New("address"),
						resourceName,
						tfjsonpath.New("ipv4").AtSliceIndex(0),
						compare.ValuesSame(),
					),
					statecheck.CompareValuePairs(dataResourceName, tfjsonpath.New("region"), resourceName, tfjsonpath.New("region"), compare.ValuesSame()),
					statecheck.ExpectKnownValue(dataResourceName, tfjsonpath.New("gateway"), knownvalue.StringRegexp(regexp.MustCompile(`\.1$`))),
					statecheck.ExpectKnownValue(dataResourceName, tfjsonpath.New("type"), knownvalue.StringExact("ipv4")),
					statecheck.ExpectKnownValue(dataResourceName, tfjsonpath.New("public"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(dataResourceName, tfjsonpath.New("reserved"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dataResourceName, tfjsonpath.New("prefix"), knownvalue.Int64Exact(24)),
					statecheck.ExpectKnownValue(
						dataResourceName,
						tfjsonpath.New("rdns"),
						knownvalue.StringRegexp(regexp.MustCompile(`.ip.linodeusercontent.com$`)),
					),
					statecheck.ExpectKnownValue(dataResourceName, tfjsonpath.New("vpc_nat_1_1"), knownvalue.Null()),
					statecheck.ExpectKnownValue(dataResourceName, tfjsonpath.New("interface_id"), knownvalue.Null()),
				},
			},
		},
	})
}
