//go:build integration || placementgroups

package placementgroups_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/placementgroups/tmpl"
)

func TestAccDataSourcePlacementGroups_basic(t *testing.T) {
	t.Parallel()

	const dsAllName = "data.linode_placement_groups.all"
	const dsByLabelName = "data.linode_placement_groups.by-label"
	const dsByATName = "data.linode_placement_groups.by-placement-group-type"

	baseLabel := acctest.RandomWithPrefix("tf-test")

	testRegion, err := acceptance.GetRandomRegionWithCaps([]string{"Placement Group"}, "core")
	if err != nil {
		t.Error(fmt.Errorf("failed to get region with PG capability: %w", err))
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t, baseLabel, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					// dsAllName ("data.linode_placement_groups.all") checks
					acceptance.StateCheckResourceAttrGreaterThan(dsAllName, "placement_groups.#", 2),
					statecheck.ExpectKnownValue(dsAllName, tfjsonpath.New("placement_groups").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsAllName, tfjsonpath.New("placement_groups").AtSliceIndex(0).AtMapKey("label"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsAllName, tfjsonpath.New("placement_groups").AtSliceIndex(0).AtMapKey("placement_group_type"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsAllName, tfjsonpath.New("placement_groups").AtSliceIndex(0).AtMapKey("region"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsAllName, tfjsonpath.New("placement_groups").AtSliceIndex(0).AtMapKey("is_compliant"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsAllName, tfjsonpath.New("placement_groups").AtSliceIndex(0).AtMapKey("placement_group_policy"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsAllName, tfjsonpath.New("placement_groups").AtSliceIndex(0).AtMapKey("members"), knownvalue.NotNull()),

					// dsByLabelName ("data.linode_placement_groups.by-label") checks
					statecheck.ExpectKnownValue(dsByLabelName, tfjsonpath.New("placement_groups"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(dsByLabelName, tfjsonpath.New("placement_groups").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsByLabelName, tfjsonpath.New("placement_groups").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact(baseLabel+"-1")),
					statecheck.ExpectKnownValue(dsByLabelName, tfjsonpath.New("placement_groups").AtSliceIndex(0).AtMapKey("placement_group_type"), knownvalue.StringExact("anti_affinity:local")),
					statecheck.ExpectKnownValue(dsByLabelName, tfjsonpath.New("placement_groups").AtSliceIndex(0).AtMapKey("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(dsByLabelName, tfjsonpath.New("placement_groups").AtSliceIndex(0).AtMapKey("is_compliant"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsByLabelName, tfjsonpath.New("placement_groups").AtSliceIndex(0).AtMapKey("placement_group_policy"), knownvalue.StringExact("strict")),
					statecheck.ExpectKnownValue(dsByLabelName, tfjsonpath.New("placement_groups").AtSliceIndex(0).AtMapKey("members"), knownvalue.ListSizeExact(0)),

					// dsByATName ("data.linode_placement_groups.by-placement-group-type") checks
					acceptance.StateCheckResourceAttrGreaterThan(dsByATName, "placement_groups.#", 2),
					statecheck.ExpectKnownValue(dsByATName, tfjsonpath.New("placement_groups").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsByATName, tfjsonpath.New("placement_groups").AtSliceIndex(0).AtMapKey("label"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsByATName, tfjsonpath.New("placement_groups").AtSliceIndex(0).AtMapKey("placement_group_type"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsByATName, tfjsonpath.New("placement_groups").AtSliceIndex(0).AtMapKey("region"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsByATName, tfjsonpath.New("placement_groups").AtSliceIndex(0).AtMapKey("is_compliant"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsByATName, tfjsonpath.New("placement_groups").AtSliceIndex(0).AtMapKey("placement_group_policy"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsByATName, tfjsonpath.New("placement_groups").AtSliceIndex(0).AtMapKey("members"), knownvalue.NotNull()),
				},
			},
		},
	})
}
