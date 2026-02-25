//go:build integration || volume

package volume_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/volume/tmpl"
)

func TestAccDataSourceVolume_basic(t *testing.T) {
	t.Parallel()

	volumeName := acctest.RandomWithPrefix("tf_test")
	resourceName := "data.linode_volume.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t, volumeName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("size"), knownvalue.Int64Exact(20)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("label"), knownvalue.StringExact(volumeName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("tags").AtSliceIndex(0), knownvalue.StringExact("tf_test")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("linode_id"), knownvalue.Int64Exact(0)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("created"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("updated"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("encryption"), knownvalue.StringExact("enabled")),
				},
			},
		},
	})
}

func TestAccDataSourceVolume_withBlockStorageEncryption(t *testing.T) {
	t.Parallel()

	volumeName := acctest.RandomWithPrefix("tf_test")
	resourceName := "data.linode_volume.foobar"

	// Resolve a region with support for Block Storage Encryption
	targetRegion, err := acceptance.GetRandomRegionWithCaps(
		[]string{"Linodes", "Block Storage Encryption"},
		"core",
	)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataWithBlockStorageEncryption(t, volumeName, targetRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("region"), knownvalue.StringExact(targetRegion)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("encryption"), knownvalue.StringExact("enabled")),
				},
			},
		},
	})
}
