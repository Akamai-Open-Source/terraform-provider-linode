//go:build integration || stackscript

package stackscript_test

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
	"github.com/linode/terraform-provider-linode/v3/linode/stackscript/tmpl"
)

func init() {
	resource.AddTestSweepers("linode_stackscript", &resource.Sweeper{
		Name: "linode_stackscript",
		F:    sweep,
	})
}

func sweep(prefix string) error {
	client, err := acceptance.GetTestClient()
	if err != nil {
		return fmt.Errorf("Error getting client: %s", err)
	}

	listOpts := acceptance.SweeperListOptions(prefix, "label")
	stackscripts, err := client.ListStackscripts(context.Background(), listOpts)
	if err != nil {
		return fmt.Errorf("Error getting stackscripts: %s", err)
	}
	for _, stackscript := range stackscripts {
		if !acceptance.ShouldSweep(prefix, stackscript.Label) {
			continue
		}
		err := client.DeleteStackscript(context.Background(), stackscript.ID)
		if err != nil {
			return fmt.Errorf("Error destroying %s during sweep: %s", stackscript.Label, err)
		}
	}

	return nil
}

func TestSmokeTests_stackscript(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{"TestAccResourceStackscript_basic_smoke", TestAccResourceStackscript_basic_smoke},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestAccResourceStackscript_basic_smoke(t *testing.T) {
	t.Parallel()

	resName := "linode_stackscript.foobar"
	stackscriptName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkStackscriptDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, stackscriptName),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckStackscriptExists,
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("description"), knownvalue.StringExact("tf_test stackscript")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("rev_note"), knownvalue.StringExact("initial")),
					acceptance.StateCheckListContains(resName, "images", "linode/ubuntu24.04"),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(stackscriptName)),
				},
			},

			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created", "updated"}, // Ignore strict comparison for these attributes
			},
		},
	})
}

func TestAccResourceStackscript_update(t *testing.T) {
	t.Parallel()

	stackscriptName := acctest.RandomWithPrefix("tf_test")
	resName := "linode_stackscript.foobar"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkStackscriptDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, stackscriptName),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckStackscriptExists,
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("description"), knownvalue.StringExact("tf_test stackscript")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("rev_note"), knownvalue.StringExact("initial")),
					acceptance.StateCheckListContains(resName, "images", "linode/ubuntu24.04"),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(stackscriptName)),
				},
			},
			{
				Config: tmpl.Basic(t, stackscriptName+"_renamed"),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckStackscriptExists,
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("description"), knownvalue.StringExact("tf_test stackscript")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("rev_note"), knownvalue.StringExact("initial")),
					acceptance.StateCheckListContains(resName, "images", "linode/ubuntu24.04"),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(fmt.Sprintf("%s_renamed", stackscriptName))),
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

func TestAccResourceStackscript_codeChange(t *testing.T) {
	t.Parallel()

	stackscriptName := acctest.RandomWithPrefix("tf_test")
	resName := "linode_stackscript.foobar"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkStackscriptDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, stackscriptName),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckStackscriptExists,
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("description"), knownvalue.StringExact("tf_test stackscript")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("rev_note"), knownvalue.StringExact("initial")),
					acceptance.StateCheckListContains(resName, "images", "linode/ubuntu24.04"),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("script"), knownvalue.StringExact("#!/bin/bash\necho hello\n")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("user_defined_fields"), knownvalue.ListSizeExact(0)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(stackscriptName)),
				},
			},
			{
				Config: tmpl.CodeChange(t, stackscriptName),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckStackscriptExists,
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("description"), knownvalue.StringExact("tf_test stackscript")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("rev_note"), knownvalue.StringExact("second")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("script"), knownvalue.StringExact("#!/bin/bash\n# <UDF name=\"hasudf\" label=\"a label\" example=\"an example\" default=\"a default\">\necho bye\n")),
					acceptance.StateCheckListContains(resName, "images", "linode/ubuntu24.04"),
					acceptance.StateCheckListContains(resName, "images", "linode/ubuntu22.04"),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("user_defined_fields"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("user_defined_fields").AtSliceIndex(0).AtMapKey("name"), knownvalue.StringExact("hasudf")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("user_defined_fields").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("a label")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("user_defined_fields").AtSliceIndex(0).AtMapKey("default"), knownvalue.StringExact("a default")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("user_defined_fields").AtSliceIndex(0).AtMapKey("example"), knownvalue.StringExact("an example")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(stackscriptName)),
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

func checkStackscriptExists(s *terraform.State) error {
	client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_stackscript" {
			continue
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error parsing %v to int", rs.Primary.ID)
		}

		_, err = client.GetStackscript(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Error retrieving state of Stackscript %s: %s", rs.Primary.Attributes["label"], err)
		}
	}

	return nil
}

var stateCheckStackscriptExists = acceptance.CustomStateCheck(
	func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Type != "linode_stackscript" {
				continue
			}

			idVal, ok := rc.AttributeValues["id"]
			if !ok {
				resp.Error = fmt.Errorf("No ID is set")
				return
			}

			id, err := strconv.Atoi(idVal.(string))
			if err != nil {
				resp.Error = fmt.Errorf("Error parsing %v to int", idVal)
				return
			}

			_, err = client.GetStackscript(context.Background(), id)
			if err != nil {
				resp.Error = fmt.Errorf("Error retrieving state of Stackscript %v: %s", rc.AttributeValues["label"], err)
				return
			}
		}
	},
)

func checkStackscriptDestroy(s *terraform.State) error {
	client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_stackscript" {
			continue
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error parsing %v to int", rs.Primary.ID)
		}
		if id == 0 {
			return fmt.Errorf("Would have considered %v as %d", rs.Primary.ID, id)
		}

		_, err = client.GetStackscript(context.Background(), id)

		if err == nil {
			return fmt.Errorf("Linode Stackscript with id %d still exists", id)
		}

		if apiErr, ok := err.(*linodego.Error); ok && apiErr.Code != 404 {
			return fmt.Errorf("error requesting Linode Stackscript with id %d: %s", id, apiErr)
		}
	}

	return nil
}
