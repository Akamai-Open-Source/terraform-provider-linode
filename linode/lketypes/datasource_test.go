//go:build integration || lketypes

package lketypes_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/lketypes/tmpl"
)

func TestAccDataSourceLKETypes_basic(t *testing.T) {
	t.Parallel()

	dataSourceName := "data.linode_lke_types.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						dataSourceName,
						tfjsonpath.New("types"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						dataSourceName,
						tfjsonpath.New("types").AtSliceIndex(0).AtMapKey("id"),
						knownvalue.StringExact("lke-sa"),
					),
					statecheck.ExpectKnownValue(
						dataSourceName,
						tfjsonpath.New("types").AtSliceIndex(0).AtMapKey("label"),
						knownvalue.StringExact("LKE Standard Availability"),
					),
					statecheck.ExpectKnownValue(
						dataSourceName,
						tfjsonpath.New("types").AtSliceIndex(0).AtMapKey("transfer"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						dataSourceName,
						tfjsonpath.New("types").AtSliceIndex(0).AtMapKey("price").AtSliceIndex(0).AtMapKey("hourly"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						dataSourceName,
						tfjsonpath.New("types").AtSliceIndex(0).AtMapKey("price").AtSliceIndex(0).AtMapKey("monthly"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}
