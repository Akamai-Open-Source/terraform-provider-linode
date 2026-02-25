//go:build integration || sshkey

package sshkey_test

import (
	"context"
	"fmt"
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
	"github.com/linode/terraform-provider-linode/v3/linode/sshkey/tmpl"
)

func init() {
	resource.AddTestSweepers("linode_sshkey", &resource.Sweeper{
		Name: "linode_sshkey",
		F:    sweep,
	})
}

func sweep(prefix string) error {
	client, err := acceptance.GetTestClient()
	if err != nil {
		return fmt.Errorf("Error getting client: %s", err)
	}

	listOpts := acceptance.SweeperListOptions(prefix, "label")
	sshkeys, err := client.ListSSHKeys(context.Background(), listOpts)
	if err != nil {
		return fmt.Errorf("Error getting sshkeys: %s", err)
	}
	for _, sshkey := range sshkeys {
		if !acceptance.ShouldSweep(prefix, sshkey.Label) {
			continue
		}
		err := client.DeleteSSHKey(context.Background(), sshkey.ID)
		if err != nil {
			return fmt.Errorf("Error destroying %s during sweep: %s", sshkey.Label, err)
		}
	}

	return nil
}

func TestAccResourceSSHKey_basic(t *testing.T) {
	t.Parallel()

	resName := "linode_sshkey.foobar"
	sshkeyName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkSSHKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, sshkeyName, acceptance.PublicKeyMaterial),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckSSHKeyExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(sshkeyName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("ssh_key"), knownvalue.StringExact(acceptance.PublicKeyMaterial)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("created"), knownvalue.NotNull()),
				},
			},

			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccResourceSSHKey_space_in_label(t *testing.T) {
	t.Parallel()

	resName := "linode_sshkey.foobar"
	sshkeyName := acctest.RandomWithPrefix("tf_test") + " "

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkSSHKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, sshkeyName, acceptance.PublicKeyMaterial),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckSSHKeyExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(sshkeyName)),
				},
			},
			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccResourceSSHKey_update(t *testing.T) {
	t.Parallel()
	resName := "linode_sshkey.foobar"
	sshkeyName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkSSHKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, sshkeyName, acceptance.PublicKeyMaterial),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckSSHKeyExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(sshkeyName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("ssh_key"), knownvalue.StringExact(acceptance.PublicKeyMaterial)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("created"), knownvalue.NotNull()),
				},
			},
			{
				Config: tmpl.Updates(t, sshkeyName, acceptance.PublicKeyMaterial),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckSSHKeyExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(fmt.Sprintf("%s_renamed", sshkeyName))),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("ssh_key"), knownvalue.StringExact(acceptance.PublicKeyMaterial)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("created"), knownvalue.NotNull()),
				},
			},
			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func stateCheckSSHKeyExists() statecheck.StateCheck {
	return acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Type != "linode_sshkey" {
				continue
			}

			idVal, ok := rc.AttributeValues["id"]
			if !ok {
				resp.Error = fmt.Errorf("No ID is set")
				return
			}

			idStr, ok := idVal.(string)
			if !ok {
				resp.Error = fmt.Errorf("id is not a string")
				return
			}

			id, err := strconv.Atoi(idStr)
			if err != nil {
				resp.Error = fmt.Errorf("Error parsing %v to int", idVal)
				return
			}

			_, err = client.GetSSHKey(context.Background(), id)
			if err != nil {
				label, _ := rc.AttributeValues["label"].(string)
				resp.Error = fmt.Errorf("Error retrieving state of SSHKey %s: %s", label, err)
				return
			}
		}
	})
}

func checkSSHKeyDestroy(s *terraform.State) error {
	client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_sshkey" {
			continue
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error parsing %v to int", rs.Primary.ID)
		}
		if id == 0 {
			return fmt.Errorf("Would have considered %v as %d", rs.Primary.ID, id)
		}

		_, err = client.GetSSHKey(context.Background(), id)

		if err == nil {
			return fmt.Errorf("Linode SSH Key with id %d still exists", id)
		}

		if apiErr, ok := err.(*linodego.Error); ok && apiErr.Code != 404 {
			return fmt.Errorf("Error requesting Linode SSH Key with id %d", id)
		}
	}

	return nil
}
