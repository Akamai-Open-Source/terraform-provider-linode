//go:build integration || producerimagesharegroup

package producerimagesharegroup_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/producerimagesharegroup/tmpl"
)

func TestAccResourceImageShareGroup_basic(t *testing.T) {
	t.Parallel()

	resourceName := "linode_producer_image_share_group.foobar"
	label := acctest.RandomWithPrefix("tf-test")
	description := "A cool description."

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, label, description),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("uuid"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("label"), knownvalue.StringExact(label)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("description"), knownvalue.StringExact(description)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("is_suspended"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images_count"), knownvalue.StringExact("0")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("members_count"), knownvalue.StringExact("0")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("created"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("updated"), knownvalue.Null()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("expiry"), knownvalue.Null()),
				},
			},
		},
	})
}

func TestAccResourceImageShareGroup_updates(t *testing.T) {
	t.Parallel()

	resourceName := "linode_producer_image_share_group.foobar"
	label := acctest.RandomWithPrefix("tf-test")
	testRegion, err := acceptance.GetRandomRegionWithCaps([]string{"Linodes"}, "core")
	if err != nil {
		t.Fatalf("failed to get test region: %s", err)
	}
	imageLabel1 := "image" + acctest.RandomWithPrefix("tf-test")
	imageLabel2 := "image" + acctest.RandomWithPrefix("tf-test")
	isgLabel := "sharegroup" + acctest.RandomWithPrefix("tf-test")
	isgDescription := "A cool description."
	isgLabelUpdated := isgLabel + "-updated"
	isgDescriptionUpdated := isgDescription + " updated"

	imagesStep2 := []tmpl.ShareGroupImageTemplate{
		{
			ID:          "${linode_image.foobar.id}",
			Label:       "Share-Image-1",
			Description: "Share Image 1 Description",
		},
	}

	imagesStep3 := []tmpl.ShareGroupImageTemplate{
		{
			ID:          "${linode_image.foobar.id}",
			Label:       "Share-Image-1-updated",
			Description: "Share Image 1 Description updated",
		},
		{
			ID:          "${linode_image.barfoo.id}",
			Label:       "Share-Image-2",
			Description: "Share Image 2 Description",
		},
	}

	imagesStep4 := []tmpl.ShareGroupImageTemplate{
		{
			ID:          "${linode_image.foobar.id}",
			Label:       "Share-Image-1-updated",
			Description: "Share Image 1 Description updated",
		},
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create empty share group
			{
				Config: tmpl.Updates(t, label, testRegion, imageLabel1, imageLabel2, isgLabel, isgDescription, nil),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("uuid"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("label"), knownvalue.StringExact(isgLabel)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("description"), knownvalue.StringExact(isgDescription)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("is_suspended"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images_count"), knownvalue.StringExact("0")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("members_count"), knownvalue.StringExact("0")),
				},
			},
			// Step 2: Add first image
			{
				Config: tmpl.Updates(t, label, testRegion, imageLabel1, imageLabel2, isgLabel, isgDescription, imagesStep2),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("uuid"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("label"), knownvalue.StringExact(isgLabel)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("description"), knownvalue.StringExact(isgDescription)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("is_suspended"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images_count"), knownvalue.StringExact("1")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("members_count"), knownvalue.StringExact("0")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("Share-Image-1")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("description"), knownvalue.StringExact("Share Image 1 Description")),
				},
			},
			// Step 3: Add second image and update first image
			{
				Config: tmpl.Updates(t, label, testRegion, imageLabel1, imageLabel2, isgLabel, isgDescription, imagesStep3),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("uuid"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("label"), knownvalue.StringExact(isgLabel)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("description"), knownvalue.StringExact(isgDescription)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("is_suspended"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images_count"), knownvalue.StringExact("2")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("members_count"), knownvalue.StringExact("0")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("Share-Image-1-updated")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("description"), knownvalue.StringExact("Share Image 1 Description updated")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(1).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(1).AtMapKey("label"), knownvalue.StringExact("Share-Image-2")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(1).AtMapKey("description"), knownvalue.StringExact("Share Image 2 Description")),
				},
			},
			// Step 4: Update the Share Group and remove the second image
			{
				Config: tmpl.Updates(t, label, testRegion, imageLabel1, imageLabel2, isgLabelUpdated, isgDescriptionUpdated, imagesStep4),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("uuid"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("label"), knownvalue.StringExact(isgLabel+"-updated")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("description"), knownvalue.StringExact(isgDescription+" updated")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("is_suspended"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images_count"), knownvalue.StringExact("1")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("members_count"), knownvalue.StringExact("0")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("Share-Image-1-updated")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("description"), knownvalue.StringExact("Share Image 1 Description updated")),
				},
			},
			// Step 5: Remove the first image
			{
				Config: tmpl.Updates(t, label, testRegion, imageLabel1, imageLabel2, isgLabelUpdated, isgDescriptionUpdated, nil),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("uuid"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("label"), knownvalue.StringExact(isgLabel+"-updated")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("description"), knownvalue.StringExact(isgDescription+" updated")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("is_suspended"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images_count"), knownvalue.StringExact("0")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("members_count"), knownvalue.StringExact("0")),
				},
			},
		},
	})
}
