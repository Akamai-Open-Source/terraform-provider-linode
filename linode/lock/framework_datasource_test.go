//go:build integration || lock

package lock_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/linodego"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/lock/tmpl"
)

const (
	testLockDataName = "data.linode_lock.test"
)

func TestAccDataSourceLock_basic(t *testing.T) {
	t.Parallel()

	testRegion, err := acceptance.GetRandomRegionWithCaps([]string{linodego.CapabilityLinodes}, "core")
	if err != nil {
		t.Fatal(err)
	}

	instanceLabel := acctest.RandomWithPrefix("tf_test")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t, instanceLabel, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(testLockDataName, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testLockDataName, tfjsonpath.New("entity_id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testLockDataName, tfjsonpath.New("entity_type"), knownvalue.StringExact("linode")),
					statecheck.ExpectKnownValue(testLockDataName, tfjsonpath.New("lock_type"), knownvalue.StringExact("cannot_delete")),
					statecheck.ExpectKnownValue(testLockDataName, tfjsonpath.New("entity_label"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testLockDataName, tfjsonpath.New("entity_url"), knownvalue.NotNull()),
				},
			},
		},
	})
}
