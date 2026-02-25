//go:build integration || user

package user_test

import (
	"context"
	"fmt"
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
	"github.com/linode/terraform-provider-linode/v3/linode/user/tmpl"
)

const testUserResName = "linode_user.test"

func TestAccResourceUser_basic(t *testing.T) {
	t.Parallel()

	username := acctest.RandomWithPrefix("tf-test")
	email := username + "@example.com"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, username, email, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("email"), knownvalue.StringExact(email)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("username"), knownvalue.StringExact(username)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("restricted"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("user_type"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("ssh_keys"), knownvalue.ListSizeExact(0)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("tfa_enabled"), knownvalue.Bool(false)),
				},
			},
		},
	})
}

func TestAccResourceUser_updates(t *testing.T) {
	t.Parallel()

	username := acctest.RandomWithPrefix("tf-test")
	updatedUsername := acctest.RandomWithPrefix("tf-test")
	email := username + "@example.com"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, username, email, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("email"), knownvalue.StringExact(email)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("username"), knownvalue.StringExact(username)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("restricted"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("ssh_keys"), knownvalue.ListSizeExact(0)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("tfa_enabled"), knownvalue.Bool(false)),
				},
			},
			{
				Config: tmpl.Basic(t, updatedUsername, email, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("email"), knownvalue.StringExact(email)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("username"), knownvalue.StringExact(updatedUsername)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("restricted"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("ssh_keys"), knownvalue.ListSizeExact(0)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("tfa_enabled"), knownvalue.Bool(false)),
				},
			},
		},
	})
}

func TestAccResourceUser_grants(t *testing.T) {
	t.Parallel()

	username := acctest.RandomWithPrefix("tf-test")
	instance := acctest.RandomWithPrefix("tf-test")

	email := username + "@example.com"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Grants(t, username, email),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("global_grants").AtSliceIndex(0).AtMapKey("account_access"), knownvalue.StringExact("")),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("global_grants").AtSliceIndex(0).AtMapKey("add_domains"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("global_grants").AtSliceIndex(0).AtMapKey("add_databases"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("global_grants").AtSliceIndex(0).AtMapKey("add_firewalls"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("global_grants").AtSliceIndex(0).AtMapKey("add_images"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("global_grants").AtSliceIndex(0).AtMapKey("add_linodes"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("global_grants").AtSliceIndex(0).AtMapKey("add_longview"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("global_grants").AtSliceIndex(0).AtMapKey("add_nodebalancers"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("global_grants").AtSliceIndex(0).AtMapKey("add_stackscripts"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("global_grants").AtSliceIndex(0).AtMapKey("add_volumes"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("global_grants").AtSliceIndex(0).AtMapKey("add_vpcs"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("global_grants").AtSliceIndex(0).AtMapKey("cancel_account"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("global_grants").AtSliceIndex(0).AtMapKey("longview_subscription"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("linode_grant"), knownvalue.SetSizeExact(0)),
				},
			},
			{
				Config: tmpl.GrantsUpdate(t, username, email, instance),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("global_grants").AtSliceIndex(0).AtMapKey("account_access"), knownvalue.StringExact("read_only")),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("global_grants").AtSliceIndex(0).AtMapKey("add_domains"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("global_grants").AtSliceIndex(0).AtMapKey("add_databases"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("global_grants").AtSliceIndex(0).AtMapKey("add_firewalls"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("global_grants").AtSliceIndex(0).AtMapKey("add_images"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("global_grants").AtSliceIndex(0).AtMapKey("add_linodes"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("global_grants").AtSliceIndex(0).AtMapKey("add_longview"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("global_grants").AtSliceIndex(0).AtMapKey("add_nodebalancers"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("global_grants").AtSliceIndex(0).AtMapKey("add_stackscripts"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("global_grants").AtSliceIndex(0).AtMapKey("add_volumes"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("global_grants").AtSliceIndex(0).AtMapKey("add_vpcs"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("global_grants").AtSliceIndex(0).AtMapKey("cancel_account"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("global_grants").AtSliceIndex(0).AtMapKey("longview_subscription"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("linode_grant"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue(testUserResName, tfjsonpath.New("linode_grant").AtSliceIndex(0).AtMapKey("permissions"), knownvalue.StringExact("read_write")),
				},
			},
		},
	})
}

func checkUserDestroy(s *terraform.State) error {
	client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_user" {
			continue
		}

		username := rs.Primary.ID
		_, err := client.GetUser(context.TODO(), username)

		if err == nil {
			return fmt.Errorf("should not find user %s existing after delete", username)
		}

		if apiErr, ok := err.(*linodego.Error); ok && apiErr.Code != 404 {
			return fmt.Errorf("Error getting user %s: %s", username, err)
		}
	}
	return nil
}
