//go:build integration || images

package images_test

import (
	"log"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/images/tmpl"
)

var testRegion string

func init() {
	region, err := acceptance.GetRandomRegionWithCaps(nil, "core")
	if err != nil {
		log.Fatal(err)
	}

	testRegion = region
}

func TestAccDataSourceImages_basic_smoke(t *testing.T) {
	t.Parallel()

	imageName := acctest.RandomWithPrefix("tf_test")
	resourceName := "data.linode_images.foobar"
	label := imageName
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t, imageName, testRegion, label),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact(imageName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("description"), knownvalue.StringExact("descriptive text")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("is_public"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("is_shared"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("image_sharing").AtMapKey("shared_with").AtMapKey("sharegroup_count"), knownvalue.StringExact("0")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("image_sharing").AtMapKey("shared_with").AtMapKey("sharegroup_list_url"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("image_sharing").AtMapKey("shared_by"), knownvalue.Null()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("type"), knownvalue.StringExact("manual")),
					acceptance.StateCheckListContains(resourceName, "images.0.tags", "test"),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("created"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("created_by"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("size"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("deprecated"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("capabilities"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("total_size"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("replications"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(1).AtMapKey("label"), knownvalue.StringExact(imageName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(1).AtMapKey("description"), knownvalue.StringExact("descriptive text")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(1).AtMapKey("is_public"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(1).AtMapKey("is_shared"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(1).AtMapKey("image_sharing").AtMapKey("shared_with").AtMapKey("sharegroup_count"), knownvalue.StringExact("0")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(1).AtMapKey("image_sharing").AtMapKey("shared_with").AtMapKey("sharegroup_list_url"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(1).AtMapKey("image_sharing").AtMapKey("shared_by"), knownvalue.Null()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(1).AtMapKey("type"), knownvalue.StringExact("manual")),
					acceptance.StateCheckListContains(resourceName, "images.1.tags", "test"),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(1).AtMapKey("created"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(1).AtMapKey("created_by"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(1).AtMapKey("size"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(1).AtMapKey("deprecated"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(1).AtMapKey("capabilities"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(1).AtMapKey("total_size"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(1).AtMapKey("replications"), knownvalue.NotNull()),
				},
			},

			// These cases are all used in the same test to avoid recreating images unnecessarily
			{
				Config: tmpl.DataLatest(t, imageName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact(imageName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("description"), knownvalue.StringExact("descriptive text")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("is_public"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("type"), knownvalue.StringExact("manual")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("created"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("created_by"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("size"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("deprecated"), knownvalue.NotNull()),
				},
			},

			{
				Config: tmpl.DataLatestEmpty(t, imageName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images"), knownvalue.ListSizeExact(0)),
				},
			},

			{
				Config: tmpl.DataOrder(t, imageName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					// Ensure order is correctly appended to filter
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images"), knownvalue.ListSizeExact(2)),
				},
			},

			{
				Config: tmpl.DataSubstring(t, imageName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					// Ensure order is correctly appended to filter
					acceptance.StateCheckResourceAttrGreaterThan(resourceName, "images.#", 1),
					acceptance.StateCheckResourceAttrContains(resourceName, "images.0.label", "Alpine"),
				},
			},

			{
				Config: tmpl.DataClientFilter(t, imageName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images"), knownvalue.ListSizeExact(1)),
					acceptance.StateCheckResourceAttrContains(resourceName, "images.0.label", imageName),
				},
			},
		},
	})
}
