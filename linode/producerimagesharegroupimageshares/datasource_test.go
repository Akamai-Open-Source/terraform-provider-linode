//go:build integration || producerimagesharegroupimageshares

package producerimagesharegroupimageshares_test

import (
	"log"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/producerimagesharegroupimageshares/tmpl"
)

func TestAccDataSourceImageShareGroupImageShares_basic(t *testing.T) {
	t.Parallel()

	const dsAll = "data.linode_producer_image_share_group_image_shares.all"
	const dsByID = "data.linode_producer_image_share_group_image_shares.by_id"
	const dsByLabel = "data.linode_producer_image_share_group_image_shares.by_label"

	label := acctest.RandomWithPrefix("tf_test")
	instanceLabel := acctest.RandomWithPrefix("tf_test")

	instanceRegion, err := acceptance.GetRandomRegionWithCaps([]string{}, "core")
	if err != nil {
		log.Fatal(err)
	}

	imageLabel1 := acctest.RandomWithPrefix("tf-test")
	imageLabel2 := acctest.RandomWithPrefix("tf-test")
	shareGroupLabel := acctest.RandomWithPrefix("tf-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t, label, instanceLabel, instanceRegion, imageLabel1, imageLabel2, shareGroupLabel),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dsAll, tfjsonpath.New("image_shares"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue(dsAll, tfjsonpath.New("image_shares").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("image_one_label")),
					statecheck.ExpectKnownValue(dsAll, tfjsonpath.New("image_shares").AtSliceIndex(0).AtMapKey("description"), knownvalue.StringExact("image one description")),
					statecheck.ExpectKnownValue(dsAll, tfjsonpath.New("image_shares").AtSliceIndex(1).AtMapKey("label"), knownvalue.StringExact("image_two_label")),
					statecheck.ExpectKnownValue(dsAll, tfjsonpath.New("image_shares").AtSliceIndex(1).AtMapKey("description"), knownvalue.StringExact("image two description")),

					statecheck.ExpectKnownValue(dsByID, tfjsonpath.New("image_shares"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(dsByID, tfjsonpath.New("image_shares").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("image_one_label")),
					statecheck.ExpectKnownValue(dsByID, tfjsonpath.New("image_shares").AtSliceIndex(0).AtMapKey("description"), knownvalue.StringExact("image one description")),

					statecheck.ExpectKnownValue(dsByLabel, tfjsonpath.New("image_shares"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(dsByLabel, tfjsonpath.New("image_shares").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("image_two_label")),
					statecheck.ExpectKnownValue(dsByLabel, tfjsonpath.New("image_shares").AtSliceIndex(0).AtMapKey("description"), knownvalue.StringExact("image two description")),
				},
			},
		},
	})
}
