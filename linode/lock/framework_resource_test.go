//go:build integration || lock

package lock_test

import (
	"context"
	"fmt"
	"log"
	"strconv"
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
	"github.com/linode/terraform-provider-linode/v3/linode/lock/tmpl"
)

var testRegion string

func init() {
	region, err := acceptance.GetRandomRegionWithCaps([]string{linodego.CapabilityLinodes}, "core")
	if err != nil {
		log.Fatal(err)
	}

	testRegion = region
}

func TestAccResourceLock_basic(t *testing.T) {
	t.Parallel()

	instanceName := "linode_instance.test"
	lockName := "linode_lock.test"

	var instance linodego.Instance

	label := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreventPostDestroyRefresh: true,
		PreCheck:                  func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories:  acceptance.ProtoV6ProviderFactories,
		CheckDestroy:              checkLockDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, label, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(instanceName, &instance),
					statecheck.ExpectKnownValue(lockName, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(lockName, tfjsonpath.New("entity_id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(lockName, tfjsonpath.New("entity_type"), knownvalue.StringExact("linode")),
					statecheck.ExpectKnownValue(lockName, tfjsonpath.New("lock_type"), knownvalue.StringExact("cannot_delete")),
					statecheck.ExpectKnownValue(lockName, tfjsonpath.New("entity_label"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(lockName, tfjsonpath.New("entity_url"), knownvalue.NotNull()),
				},
			},
			{
				ResourceName:      lockName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccResourceLock_withSubresources(t *testing.T) {
	t.Parallel()

	instanceName := "linode_instance.test"
	lockName := "linode_lock.test"

	var instance linodego.Instance

	label := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreventPostDestroyRefresh: true,
		PreCheck:                  func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories:  acceptance.ProtoV6ProviderFactories,
		CheckDestroy:              checkLockDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.WithSubresources(t, label, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(instanceName, &instance),
					statecheck.ExpectKnownValue(lockName, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(lockName, tfjsonpath.New("entity_id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(lockName, tfjsonpath.New("entity_type"), knownvalue.StringExact("linode")),
					statecheck.ExpectKnownValue(lockName, tfjsonpath.New("lock_type"), knownvalue.StringExact("cannot_delete_with_subresources")),
					statecheck.ExpectKnownValue(lockName, tfjsonpath.New("entity_label"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(lockName, tfjsonpath.New("entity_url"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func checkLockDestroy(s *terraform.State) error {
	client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_lock" {
			continue
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error parsing %v to int", rs.Primary.ID)
		}

		_, err = client.GetLock(context.Background(), id)
		if err == nil {
			return fmt.Errorf("Lock with id %d still exists", id)
		}

		if apiErr, ok := err.(*linodego.Error); ok && apiErr.Code != 404 {
			return fmt.Errorf("Error requesting Lock with id %d: %s", id, err)
		}
	}

	return nil
}
