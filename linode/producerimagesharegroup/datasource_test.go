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

func TestAccDataSourceImageShareGroup_basic(t *testing.T) {
	t.Parallel()

	resourceName := "data.linode_producer_image_share_group.foobar"

	label := acctest.RandomWithPrefix("tf-test")
	description := "A cool description."

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t, label, description),
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
