//go:build integration || instancetypes

package instancetypes_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/instancetypes/tmpl"
)

func TestAccDataSourceInstanceTypes_basic(t *testing.T) {
	t.Parallel()

	resourceName := "data.linode_instance_types.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("types"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("types").AtSliceIndex(0).AtMapKey("id"), knownvalue.StringExact("g6-standard-2")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("types").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("Linode 4GB")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("types").AtSliceIndex(0).AtMapKey("class"), knownvalue.StringExact("standard")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("types").AtSliceIndex(0).AtMapKey("disk"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("types").AtSliceIndex(0).AtMapKey("network_out"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("types").AtSliceIndex(0).AtMapKey("memory"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("types").AtSliceIndex(0).AtMapKey("transfer"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("types").AtSliceIndex(0).AtMapKey("vcpus"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("types").AtSliceIndex(0).AtMapKey("accelerated_devices"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("types").AtSliceIndex(0).AtMapKey("price").AtSliceIndex(0).AtMapKey("hourly"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("types").AtSliceIndex(0).AtMapKey("price").AtSliceIndex(0).AtMapKey("monthly"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("types").AtSliceIndex(0).AtMapKey("addons").AtSliceIndex(0).AtMapKey("backups").AtSliceIndex(0).AtMapKey("price").AtSliceIndex(0).AtMapKey("hourly"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("types").AtSliceIndex(0).AtMapKey("addons").AtSliceIndex(0).AtMapKey("backups").AtSliceIndex(0).AtMapKey("price").AtSliceIndex(0).AtMapKey("monthly"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func TestAccDataSourceInstanceTypes_substring(t *testing.T) {
	t.Parallel()

	resourceName := "data.linode_instance_types.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataSubstring(t),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckResourceAttrGreaterThan(resourceName, "types.#", 1),
					acceptance.StateCheckResourceAttrContains(resourceName, "types.0.label", "Linode"),
				},
			},
		},
	})
}

func TestAccDataSourceInstanceTypes_regex(t *testing.T) {
	t.Parallel()

	resourceName := "data.linode_instance_types.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataRegex(t),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckResourceAttrGreaterThan(resourceName, "types.#", 1),
					acceptance.StateCheckResourceAttrContains(resourceName, "types.0.label", "Dedicated"),
				},
			},
		},
	})
}

func TestAccDataSourceInstanceTypes_byClass(t *testing.T) {
	t.Parallel()

	resourceName := "data.linode_instance_types.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataByClass(t),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckResourceAttrGreaterThan(resourceName, "types.#", 0),
					acceptance.StateCheckResourceAttrContains(resourceName, "types.0.label", "Linode"),
				},
			},
		},
	})
}
