//go:build integration || stackscripts

package stackscripts_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/stackscripts/tmpl"
)

var basicStackScript = `#!/bin/bash
#<UDF name="name" label="Your name" example="Linus Torvalds" default="user">
# NAME=
echo "Hello, $NAME!"
`

func TestSmokeTests_stackscripts(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{"TestAccDataSourceStackscripts_basic_smoke", TestAccDataSourceStackscripts_basic_smoke},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestAccDataSourceStackscripts_basic_smoke(t *testing.T) {
	t.Parallel()

	stackScriptName := acctest.RandomWithPrefix("tf_test")

	resourceName := "data.linode_stackscripts.stackscript"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:            tmpl.DataBasic(t, stackScriptName, basicStackScript),
				ConfigStateChecks: validateStackscriptStateChecks(resourceName, stackScriptName),
			},
			{
				Config:            tmpl.DataSubString(t, stackScriptName, basicStackScript),
				ConfigStateChecks: validateStackscriptStateChecks(resourceName, stackScriptName),
			},
			{
				Config: tmpl.DataLatest(t, stackScriptName, basicStackScript),
				ConfigStateChecks: append(
					[]statecheck.StateCheck{
						statecheck.ExpectKnownValue(
							resourceName,
							tfjsonpath.New("stackscripts"),
							knownvalue.ListSizeExact(1),
						),
					},
					validateStackscriptStateChecks(resourceName, stackScriptName)...,
				),
			},
			{
				Config: tmpl.DataClientFilter(t, stackScriptName, basicStackScript),
				ConfigStateChecks: append(
					[]statecheck.StateCheck{
						statecheck.ExpectKnownValue(
							resourceName,
							tfjsonpath.New("stackscripts"),
							knownvalue.ListSizeExact(1),
						),
					},
					validateStackscriptStateChecks(resourceName, stackScriptName)...,
				),
			},
		},
	})
}

func validateStackscriptStateChecks(resourceName, stackScriptName string) []statecheck.StateCheck {
	return []statecheck.StateCheck{
		// TestCheckResourceAttrSet → ExpectKnownValue with NotNull()
		statecheck.ExpectKnownValue(
			resourceName,
			tfjsonpath.New("stackscripts").AtSliceIndex(0).AtMapKey("id"),
			knownvalue.NotNull(),
		),
		statecheck.ExpectKnownValue(
			resourceName,
			tfjsonpath.New("stackscripts").AtSliceIndex(0).AtMapKey("deployments_active"),
			knownvalue.NotNull(),
		),
		statecheck.ExpectKnownValue(
			resourceName,
			tfjsonpath.New("stackscripts").AtSliceIndex(0).AtMapKey("deployments_total"),
			knownvalue.NotNull(),
		),
		statecheck.ExpectKnownValue(
			resourceName,
			tfjsonpath.New("stackscripts").AtSliceIndex(0).AtMapKey("username"),
			knownvalue.NotNull(),
		),
		statecheck.ExpectKnownValue(
			resourceName,
			tfjsonpath.New("stackscripts").AtSliceIndex(0).AtMapKey("created"),
			knownvalue.NotNull(),
		),
		statecheck.ExpectKnownValue(
			resourceName,
			tfjsonpath.New("stackscripts").AtSliceIndex(0).AtMapKey("updated"),
			knownvalue.NotNull(),
		),
		// TestCheckResourceAttr → ExpectKnownValue with StringExact()
		statecheck.ExpectKnownValue(
			resourceName,
			tfjsonpath.New("stackscripts").AtSliceIndex(0).AtMapKey("label"),
			knownvalue.StringExact(stackScriptName),
		),
		statecheck.ExpectKnownValue(
			resourceName,
			tfjsonpath.New("stackscripts").AtSliceIndex(0).AtMapKey("description"),
			knownvalue.StringExact("test"),
		),
		// is_public is BoolAttribute in schema → use knownvalue.Bool(false)
		statecheck.ExpectKnownValue(
			resourceName,
			tfjsonpath.New("stackscripts").AtSliceIndex(0).AtMapKey("is_public"),
			knownvalue.Bool(false),
		),
		statecheck.ExpectKnownValue(
			resourceName,
			tfjsonpath.New("stackscripts").AtSliceIndex(0).AtMapKey("rev_note"),
			knownvalue.StringExact("initial"),
		),
		statecheck.ExpectKnownValue(
			resourceName,
			tfjsonpath.New("stackscripts").AtSliceIndex(0).AtMapKey("script"),
			knownvalue.StringExact(basicStackScript),
		),
		// images is schema.SetAttribute → use SetSizeExact for ".#" count check
		statecheck.ExpectKnownValue(
			resourceName,
			tfjsonpath.New("stackscripts").AtSliceIndex(0).AtMapKey("images"),
			knownvalue.SetSizeExact(2),
		),
		// Custom check: CheckListContains → StateCheckListContains
		acceptance.StateCheckListContains(resourceName, "stackscripts.0.images", "linode/ubuntu24.04"),
		acceptance.StateCheckListContains(resourceName, "stackscripts.0.images", "linode/ubuntu22.04"),
		// user_defined_fields is schema.ListAttribute → use ListSizeExact for ".#" count
		statecheck.ExpectKnownValue(
			resourceName,
			tfjsonpath.New("stackscripts").AtSliceIndex(0).AtMapKey("user_defined_fields"),
			knownvalue.ListSizeExact(1),
		),
		// Nested UDF attributes: "stackscripts.0.user_defined_fields.0.X"
		statecheck.ExpectKnownValue(
			resourceName,
			tfjsonpath.New("stackscripts").AtSliceIndex(0).AtMapKey("user_defined_fields").AtSliceIndex(0).AtMapKey("name"),
			knownvalue.StringExact("name"),
		),
		statecheck.ExpectKnownValue(
			resourceName,
			tfjsonpath.New("stackscripts").AtSliceIndex(0).AtMapKey("user_defined_fields").AtSliceIndex(0).AtMapKey("label"),
			knownvalue.StringExact("Your name"),
		),
		statecheck.ExpectKnownValue(
			resourceName,
			tfjsonpath.New("stackscripts").AtSliceIndex(0).AtMapKey("user_defined_fields").AtSliceIndex(0).AtMapKey("default"),
			knownvalue.StringExact("user"),
		),
		statecheck.ExpectKnownValue(
			resourceName,
			tfjsonpath.New("stackscripts").AtSliceIndex(0).AtMapKey("user_defined_fields").AtSliceIndex(0).AtMapKey("example"),
			knownvalue.StringExact("Linus Torvalds"),
		),
	}
}
