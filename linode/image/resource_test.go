//go:build integration || image

package image_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/linodego"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/helper"
	"github.com/linode/terraform-provider-linode/v3/linode/image/tmpl"
)

// testImageBytes is a minimal Gzipped image.
// This is necessary because the API will reject invalid images.
var testImageBytes = []byte{
	0x1f, 0x8b, 0x08, 0x08, 0xbd, 0x5c, 0x91, 0x60,
	0x00, 0x03, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x69, 0x6d, 0x67, 0x00, 0x03, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

// MD5 digest of testImageBytes
const testImageMD5 = "0cf442194905e7be019a11660df8164f"

var testImageBytesNew = []byte{
	0x1f, 0x8b, 0x08, 0x08, 0x53, 0x13, 0x94, 0x60,
	0x00, 0x03, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x69, 0x6d, 0x67, 0x00, 0xcb, 0xc8,
	0xe4, 0x02, 0x00, 0x7a, 0x7a, 0x6f, 0xed, 0x03, 0x00, 0x00, 0x00,
}

var (
	// This is necessary because the API does not currently expose
	// a capability for regions that allow custom image uploads.
	//
	// In the future, we should remove this if the API exposes a custom images capability or
	//	if all Object Storage regions support custom images.
	disallowedImageRegions = map[string]bool{
		"gb-lon":   true,
		"au-mel":   true,
		"sg-sin-2": true,
		"jp-tyo-3": true,
		"no-osl-1": true,
	}

	testRegion  string
	testRegions []string
)

func init() {
	resource.AddTestSweepers("linode_image", &resource.Sweeper{
		Name: "linode_image",
		F:    sweep,
	})

	regions, err := acceptance.GetRegionsWithCaps([]string{"Object Storage", "Linodes"}, "core")
	if err != nil {
		log.Fatal(err)
	}

	testRegions = helper.FilterSlice(regions, func(region string) bool {
		isDisallowed, ok := disallowedImageRegions[region]
		return !ok || !isDisallowed
	})

	testRegion = testRegions[1]
}

func sweep(prefix string) error {
	client, err := acceptance.GetTestClient()
	if err != nil {
		return fmt.Errorf("Error getting client: %s", err)
	}

	listOpts := acceptance.SweeperListOptions(prefix, "label")
	images, err := client.ListImages(context.Background(), listOpts)
	if err != nil {
		return fmt.Errorf("Error getting images: %s", err)
	}
	for _, image := range images {
		if !acceptance.ShouldSweep(prefix, image.Label) {
			continue
		}
		err := client.DeleteImage(context.Background(), image.ID)
		if err != nil {
			return fmt.Errorf("Error destroying %s during sweep: %s", image.Label, err)
		}
	}

	return nil
}

func TestAccImage_basic(t *testing.T) {
	t.Parallel()

	resName := "linode_image.foobar"
	imageName := acctest.RandomWithPrefix("tf_test")
	label := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,

		CheckDestroy: checkImageDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, imageName, testRegion, label, "test-tag"),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckImageExists(resName, nil),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(imageName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("description"), knownvalue.StringExact("descriptive text")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("created"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("created_by"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("size"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("manual")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("is_public"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("is_shared"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("image_sharing").AtMapKey("shared_with").AtMapKey("sharegroup_count"), knownvalue.StringExact("0")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("image_sharing").AtMapKey("shared_with").AtMapKey("sharegroup_list_url"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("capabilities").AtSliceIndex(0), knownvalue.StringExact("cloud-init")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("deprecated"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("tags"), knownvalue.ListSizeExact(1)),
				},
			},
			{
				ResourceName: resName,
				ImportState:  true,
				// ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"linode_id", "disk_id", "firewall_id"},
			},
		},
	})
}

func TestAccImage_update(t *testing.T) {
	t.Parallel()

	imageName := acctest.RandomWithPrefix("tf_test")
	resName := "linode_image.foobar"
	label := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkImageDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, imageName, testRegion, label, "test-tag"),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckImageExists(resName, nil),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(imageName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("description"), knownvalue.StringExact("descriptive text")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("capabilities"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("tags"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("tags").AtSliceIndex(0), knownvalue.StringExact("test-tag")),
				},
			},
			{
				Config: tmpl.Updates(t, imageName, testRegion, label, "updated-tag"),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckImageExists(resName, nil),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(fmt.Sprintf("%s_renamed", imageName))),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("description"), knownvalue.StringExact("more descriptive text")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("created"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("created_by"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("size"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("manual")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("is_public"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("deprecated"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("tags"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("tags").AtSliceIndex(0), knownvalue.StringExact("updated-tag")),
				},
			},
			{
				ResourceName: resName,
				ImportState:  true,
				// ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"linode_id", "disk_id", "firewall_id"},
			},
		},
	})
}

func TestAccImage_uploadFile(t *testing.T) {
	t.Parallel()

	resName := "linode_image.foobar"
	imageName := acctest.RandomWithPrefix("tf_test")

	file, err := createTempFile("tf-test-image-upload-file", testImageBytes)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	var image linodego.Image

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkImageDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Upload(t, imageName, file.Name(), testRegion, "test-tag"),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckImageExists(resName, &image),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(imageName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("description"), knownvalue.StringExact("really descriptive text")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("created"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("created_by"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("size"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("manual")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("is_public"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("deprecated"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("file_hash"), knownvalue.StringExact(testImageMD5)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("status"), knownvalue.StringExact(string(linodego.ImageStatusAvailable))),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("tags"), knownvalue.ListSizeExact(1)),
				},
			},
			{
				PreConfig: func() {
					file.Write(testImageBytesNew)
				},
				Config: tmpl.Upload(t, imageName, file.Name(), testRegion, "test-tag"),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckImageExists(resName, &image),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("status"), knownvalue.StringExact(string(linodego.ImageStatusAvailable))),
				},
			},
		},
	})
}

func TestAccImage_replicate(t *testing.T) {
	t.Parallel()

	resName := "linode_image.foobar"
	imageName := acctest.RandomWithPrefix("tf_test")

	if len(testRegions) < 4 {
		t.Skipf("Not enough number of capable regions for image replication test. Skipping now...")
	}

	file, err := createTempFile("tf-test-image-replicate-file", testImageBytes)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	var image linodego.Image

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkImageDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Replicate(t, imageName, file.Name(), testRegion, testRegions[0]),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckImageExists(resName, &image),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(imageName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("replications"), knownvalue.ListSizeExact(2)),
				},
			},
			{
				// Remove the one of the available region and replicate the image in a new region
				Config: tmpl.Replicate(t, imageName, file.Name(), testRegion, testRegions[2]),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckImageExists(resName, &image),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(imageName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("replications"), knownvalue.ListSizeExact(2)),
				},
			},
			{
				// Remove all available region and replicate the image in new regions
				Config: tmpl.Replicate(t, imageName, file.Name(), testRegions[0], testRegions[3]),
				ExpectError: regexp.MustCompile(
					"At least one available region must be specified"),
			},
			{
				// Remove all available region
				Config: tmpl.NoReplicaRegions(t, imageName, file.Name(), testRegion),
				ExpectError: regexp.MustCompile(
					"At least one available region must be specified"),
			},
		},
	})
}

func checkImageExists(name string, image *linodego.Image) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		found, err := client.GetImage(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("failed to retrieve state of image %s: %s", rs.Primary.Attributes["label"], err)
		}

		if image != nil {
			*image = *found
		}

		return nil
	}
}

func stateCheckImageExists(name string, image *linodego.Image) statecheck.StateCheck {
	return acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		var resourceID string
		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Address == name {
				idVal, ok := rc.AttributeValues["id"]
				if !ok {
					resp.Error = fmt.Errorf("no ID is set")
					return
				}
				resourceID = idVal.(string)
				break
			}
		}

		if resourceID == "" {
			resp.Error = fmt.Errorf("not found: %s", name)
			return
		}

		found, err := client.GetImage(context.Background(), resourceID)
		if err != nil {
			resp.Error = fmt.Errorf("failed to retrieve state of image %s: %s", resourceID, err)
			return
		}

		if image != nil {
			*image = *found
		}
	})
}

func checkImageDestroy(s *terraform.State) error {
	client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_Image" {
			continue
		}

		_, err := client.GetImage(context.Background(), rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Linode Image with id %s still exists", rs.Primary.ID)
		}

		if apiErr, ok := err.(*linodego.Error); ok && apiErr.Code != 404 {
			return fmt.Errorf("Error requesting Linode Image with id %s", rs.Primary.ID)
		}
	}

	return nil
}

func createTempFile(name string, content []byte) (*os.File, error) {
	file, err := os.CreateTemp(os.TempDir(), name)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %s", err)
	}

	if _, err := file.Write(content); err != nil {
		return nil, fmt.Errorf("failed to write to temp file: %s", err)
	}

	return file, nil
}
