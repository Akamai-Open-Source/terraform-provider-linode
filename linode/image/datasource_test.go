//go:build integration || image

package image_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/image/tmpl"
)

func TestAccDataSourceImage_basic(t *testing.T) {
	t.Parallel()

	imageID := "linode/debian8"
	resourceName := "data.linode_image.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t, imageID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("id"), knownvalue.StringExact(imageID)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("label"), knownvalue.StringExact("Debian 8")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("description"), knownvalue.StringExact("")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("is_public"), knownvalue.StringExact("true")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("is_shared"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("image_sharing").AtMapKey("shared_with"), knownvalue.Null()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("image_sharing").AtMapKey("shared_by"), knownvalue.Null()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("type"), knownvalue.StringExact("manual")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("size"), knownvalue.StringExact("1300")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("vendor"), knownvalue.StringExact("Debian")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("capabilities"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func TestAccDataSourceImage_replicate(t *testing.T) {
	t.Parallel()

	resourceName := "data.linode_image.foobar"
	imageName := acctest.RandomWithPrefix("tf_test")

	file, err := createTempFile("tf-test-image-data-replicate-file", testImageBytes)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,

		CheckDestroy: checkImageDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataReplicate(t, imageName, file.Name(), testRegion, testRegions[0]),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckImageExists(resourceName, nil),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("label"), knownvalue.StringExact(imageName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("description"), knownvalue.StringExact("really descriptive text")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("created"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("created_by"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("size"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("type"), knownvalue.StringExact("manual")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("is_public"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("deprecated"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("tags"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("total_size"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("replications"), knownvalue.ListSizeExact(2)),
				},
			},
		},
	})
}
