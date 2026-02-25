//go:build integration || stackscript

package stackscript_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/stackscript/tmpl"
)

var basicStackScript = `#!/bin/bash
#<UDF name="name" label="Your name" example="Linus Torvalds" default="user">
# NAME=
echo "Hello, $NAME!"
`

func TestAccDataSourceStackscript_basic(t *testing.T) {
	t.Parallel()

	resourceName := "data.linode_stackscript.stackscript"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t, basicStackScript),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("deployments_active"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("deployments_total"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("username"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("created"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("updated"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("label"), knownvalue.StringExact("my_stackscript")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("description"), knownvalue.StringExact("test")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("is_public"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("rev_note"), knownvalue.StringExact("initial")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("script"), knownvalue.StringExact(basicStackScript)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("images"), knownvalue.ListSizeExact(2)),
					acceptance.StateCheckListContains(resourceName, "images", "linode/ubuntu24.04"),
					acceptance.StateCheckListContains(resourceName, "images", "linode/ubuntu22.04"),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("user_defined_fields"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("user_defined_fields").AtSliceIndex(0).AtMapKey("name"), knownvalue.StringExact("name")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("user_defined_fields").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("Your name")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("user_defined_fields").AtSliceIndex(0).AtMapKey("default"), knownvalue.StringExact("user")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("user_defined_fields").AtSliceIndex(0).AtMapKey("example"), knownvalue.StringExact("Linus Torvalds")),
				},
			},
		},
	})
}
