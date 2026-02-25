//go:build integration || sshkeys

package sshkeys_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/sshkeys/tmpl"
)

func TestAccDataSourceSSHKeys_basic(t *testing.T) {
	t.Parallel()

	testSSHKeyDataName := "data.linode_sshkeys.keys"

	keyLabel := acctest.RandomWithPrefix("tf_test")
	keySSH := acceptance.PublicKeyMaterial

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataFilterEmpty(t, keyLabel, keySSH),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(testSSHKeyDataName, tfjsonpath.New("sshkeys"), knownvalue.ListSizeExact(0)),
				},
			},
			{
				Config: tmpl.DataFilter(t, keyLabel, keySSH),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(testSSHKeyDataName, tfjsonpath.New("sshkeys"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testSSHKeyDataName, tfjsonpath.New("sshkeys").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact(keyLabel+"-0")),
					statecheck.ExpectKnownValue(testSSHKeyDataName, tfjsonpath.New("sshkeys").AtSliceIndex(0).AtMapKey("ssh_key"), knownvalue.StringExact(keySSH)),
					statecheck.ExpectKnownValue(testSSHKeyDataName, tfjsonpath.New("sshkeys").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testSSHKeyDataName, tfjsonpath.New("sshkeys").AtSliceIndex(0).AtMapKey("created"), knownvalue.NotNull()),
				},
			},
			{
				Config: tmpl.DataBasic(t, keyLabel, keySSH),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(testSSHKeyDataName, tfjsonpath.New("sshkeys"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(testSSHKeyDataName, tfjsonpath.New("sshkeys").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact(keyLabel+"-0")),
					statecheck.ExpectKnownValue(testSSHKeyDataName, tfjsonpath.New("sshkeys").AtSliceIndex(0).AtMapKey("ssh_key"), knownvalue.StringExact(keySSH)),
					statecheck.ExpectKnownValue(testSSHKeyDataName, tfjsonpath.New("sshkeys").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testSSHKeyDataName, tfjsonpath.New("sshkeys").AtSliceIndex(0).AtMapKey("created"), knownvalue.NotNull()),
				},
			},
			{
				Config: tmpl.DataAll(t, keyLabel, keySSH),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckResourceAttrGreaterThan(testSSHKeyDataName, "sshkeys.#", 1),
					statecheck.ExpectKnownValue(testSSHKeyDataName, tfjsonpath.New("sshkeys").AtSliceIndex(0).AtMapKey("label"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testSSHKeyDataName, tfjsonpath.New("sshkeys").AtSliceIndex(0).AtMapKey("ssh_key"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testSSHKeyDataName, tfjsonpath.New("sshkeys").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testSSHKeyDataName, tfjsonpath.New("sshkeys").AtSliceIndex(0).AtMapKey("created"), knownvalue.NotNull()),
				},
			},
		},
	})
}
