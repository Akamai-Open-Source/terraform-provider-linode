//go:build integration || ipv6range

package ipv6range_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/ipv6range/tmpl"
)

func TestAccDataSourceIPv6Range_basic(t *testing.T) {
	t.Parallel()

	instanceLabel := acctest.RandomWithPrefix("tf_test")
	resourceName := "data.linode_ipv6_range.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t, instanceLabel, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("range"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("linodes"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("is_bgp"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("prefix"), knownvalue.Int64Exact(64)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
				},
			},
		},
	})
}
