//go:build integration || vpc

package vpc_test

import (
	"context"
	"fmt"
	"log"
	"regexp"
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
	"github.com/linode/terraform-provider-linode/v3/linode/vpc/tmpl"
)

var testRegion string

func init() {
	resource.AddTestSweepers("linode_vpc", &resource.Sweeper{
		Name: "linode_vpc",
		F:    sweep,
	})

	var err error

	testRegion, err = acceptance.GetRandomRegionWithCaps([]string{"VPCs"}, "core")
	if err != nil {
		log.Fatal(fmt.Errorf("Error getting region: %s", err))
	}
}

func sweep(prefix string) error {
	client, err := acceptance.GetTestClient()
	if err != nil {
		log.Fatal(fmt.Errorf("Error getting client: %s", err))
	}

	listOpts := acceptance.SweeperListOptions(prefix, "label")
	vpcs, err := client.ListVPCs(context.Background(), listOpts)
	if err != nil {
		return fmt.Errorf("Error getting VPCs: %s", err)
	}

	for _, vpc := range vpcs {
		if !acceptance.ShouldSweep(prefix, vpc.Label) {
			continue
		}
		err := client.DeleteVPC(context.Background(), vpc.ID)
		if err != nil {
			return fmt.Errorf("Error destroying %s during sweep: %s", vpc.Label, err)
		}
	}

	return nil
}

func TestAccResourceVPC_basic(t *testing.T) {
	t.Parallel()

	resName := "linode_vpc.foobar"
	vpcLabel := acctest.RandomWithPrefix("tf-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkVPCDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, vpcLabel, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckVPCExists(),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("label"),
						knownvalue.StringExact(vpcLabel),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("description"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("region"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("created"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("updated"),
						knownvalue.NotNull(),
					),
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

func TestAccResourceVPC_update(t *testing.T) {
	t.Parallel()
	resName := "linode_vpc.foobar"
	vpcLabel := acctest.RandomWithPrefix("tf-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkVPCDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, vpcLabel, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckVPCExists(),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("label"),
						knownvalue.StringExact(vpcLabel),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("created"),
						knownvalue.NotNull(),
					),
				},
			},
			{
				Config: tmpl.Updates(t, vpcLabel, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckVPCExists(),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("label"),
						knownvalue.StringExact(fmt.Sprintf("%s-renamed", vpcLabel)),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("description"),
						knownvalue.StringExact("some description"),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("updated"),
						knownvalue.NotNull(),
					),
				},
			},
			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"updated"},
			},
		},
	})
}

func TestAccResourceVPC_dualStack(t *testing.T) {
	t.Parallel()
	resName := "linode_vpc.foobar"
	vpcLabel := acctest.RandomWithPrefix("tf-test")

	// TODO (VPC Dual Stack): Remove region hardcoding
	targetRegion := "no-osl-1"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkVPCDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DualStack(t, vpcLabel, targetRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckVPCExists(),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("label"),
						knownvalue.StringExact(vpcLabel),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("ipv6").AtSliceIndex(0).AtMapKey("range"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("ipv6").AtSliceIndex(0).AtMapKey("allocated_range"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("created"),
						knownvalue.NotNull(),
					),
				},
			},
			{
				Config: tmpl.DualStack(t, vpcLabel, targetRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckVPCExists(),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("label"),
						knownvalue.StringExact(vpcLabel),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("ipv6").AtSliceIndex(0).AtMapKey("range"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("ipv6").AtSliceIndex(0).AtMapKey("allocated_range"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("created"),
						knownvalue.NotNull(),
					),
				},
			},
			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"updated", "ipv6.0.range"},
			},
		},
	})
}

func TestAccResourceLinodeVPC_create_InvalidLabel(t *testing.T) {
	t.Parallel()

	vpcLabel := "tf-test_123"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      tmpl.Basic(t, vpcLabel, testRegion),
				ExpectError: regexp.MustCompile("Label must include only ASCII letters, numbers, and dashes"),
			},
		},
	})
}

func TestAccResourceLinodeVPC_update_InvalidLabel(t *testing.T) {
	t.Parallel()
	resName := "linode_vpc.foobar"
	vpcLabel := acctest.RandomWithPrefix("tf-test")

	invalidLabel := "tf-test_123"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkVPCDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, vpcLabel, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckVPCExists(),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("label"),
						knownvalue.StringExact(vpcLabel),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("created"),
						knownvalue.NotNull(),
					),
				},
			},
			{
				Config:      tmpl.Updates(t, invalidLabel, testRegion),
				ExpectError: regexp.MustCompile("Label must include only ASCII letters, numbers, and dashes"),
			},
			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func stateCheckVPCExists() statecheck.StateCheck {
	return acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Type != "linode_vpc" {
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

			_, err = client.GetVPC(context.Background(), id)
			if err != nil {
				labelVal := rc.AttributeValues["label"]
				resp.Error = fmt.Errorf("Error retrieving state of VPC %s: %s", labelVal, err)
				return
			}
		}
	})
}

func checkVPCExists(s *terraform.State) error {
	client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_vpc" {
			continue
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error parsing %v to int", rs.Primary.ID)
		}

		_, err = client.GetVPC(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Error retrieving state of VPC %s: %s", rs.Primary.Attributes["label"], err)
		}
	}

	return nil
}

func checkVPCDestroy(s *terraform.State) error {
	client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_vpc" {
			continue
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error parsing %v to int", rs.Primary.ID)
		}
		if id == 0 {
			return fmt.Errorf("Would have considered %v as %d", rs.Primary.ID, id)
		}

		_, err = client.GetVPC(context.Background(), id)

		if err == nil {
			return fmt.Errorf("Linode VPC with id %d still exists", id)
		}

		if apiErr, ok := err.(*linodego.Error); ok && apiErr.Code != 404 {
			return fmt.Errorf("Error requesting Linode VPC with id %d", id)
		}
	}

	return nil
}
