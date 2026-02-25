//go:build integration || producerimagesharegroups

package producerimagesharegroups_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/producerimagesharegroups/tmpl"
)

func TestAccDataSourceImageShareGroups_basic(t *testing.T) {
	t.Parallel()

	const dsAll = "data.linode_producer_image_share_groups.all"
	const dsByLabel = "data.linode_producer_image_share_groups.by_label"
	const dsByID = "data.linode_producer_image_share_groups.by_id"
	const dsByIsSuspended = "data.linode_producer_image_share_groups.by_is_suspended"

	label1 := acctest.RandomWithPrefix("tf-test")
	label2 := acctest.RandomWithPrefix("tf-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t, label1, label2),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckResourceAttrGreaterThan(dsAll, "image_share_groups.#", 1),
					statecheck.ExpectKnownValue(dsAll, tfjsonpath.New("image_share_groups").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsAll, tfjsonpath.New("image_share_groups").AtSliceIndex(0).AtMapKey("uuid"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsAll, tfjsonpath.New("image_share_groups").AtSliceIndex(0).AtMapKey("label"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsAll, tfjsonpath.New("image_share_groups").AtSliceIndex(0).AtMapKey("is_suspended"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsAll, tfjsonpath.New("image_share_groups").AtSliceIndex(0).AtMapKey("images_count"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsAll, tfjsonpath.New("image_share_groups").AtSliceIndex(0).AtMapKey("members_count"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsAll, tfjsonpath.New("image_share_groups").AtSliceIndex(0).AtMapKey("created"), knownvalue.NotNull()),

					statecheck.ExpectKnownValue(dsByLabel, tfjsonpath.New("image_share_groups"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(dsByLabel, tfjsonpath.New("image_share_groups").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsByLabel, tfjsonpath.New("image_share_groups").AtSliceIndex(0).AtMapKey("uuid"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsByLabel, tfjsonpath.New("image_share_groups").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact(label1)),
					statecheck.ExpectKnownValue(dsByLabel, tfjsonpath.New("image_share_groups").AtSliceIndex(0).AtMapKey("is_suspended"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsByLabel, tfjsonpath.New("image_share_groups").AtSliceIndex(0).AtMapKey("images_count"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsByLabel, tfjsonpath.New("image_share_groups").AtSliceIndex(0).AtMapKey("members_count"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsByLabel, tfjsonpath.New("image_share_groups").AtSliceIndex(0).AtMapKey("created"), knownvalue.NotNull()),

					statecheck.ExpectKnownValue(dsByID, tfjsonpath.New("image_share_groups"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(dsByID, tfjsonpath.New("image_share_groups").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsByID, tfjsonpath.New("image_share_groups").AtSliceIndex(0).AtMapKey("uuid"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsByID, tfjsonpath.New("image_share_groups").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact(label2)),
					statecheck.ExpectKnownValue(dsByID, tfjsonpath.New("image_share_groups").AtSliceIndex(0).AtMapKey("is_suspended"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsByID, tfjsonpath.New("image_share_groups").AtSliceIndex(0).AtMapKey("images_count"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsByID, tfjsonpath.New("image_share_groups").AtSliceIndex(0).AtMapKey("members_count"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dsByID, tfjsonpath.New("image_share_groups").AtSliceIndex(0).AtMapKey("created"), knownvalue.NotNull()),

					acceptance.StateCheckResourceAttrGreaterThan(dsByIsSuspended, "image_share_groups.#", 1),
				},
			},
		},
	})
}
