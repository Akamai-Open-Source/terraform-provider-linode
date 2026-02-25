//go:build integration || ipv6range

package ipv6range_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/linodego"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/helper"
	"github.com/linode/terraform-provider-linode/v3/linode/ipv6range/tmpl"
)

// TODO: don't hardcode this once IPv6 sharing has a proper capability string
const testRegion = "eu-central"

func TestAccIPv6Range_basic(t *testing.T) {
	t.Parallel()

	acceptance.RunTestWithRetries(t, 3, func(t *acceptance.WrappedT) {
		resName := "linode_ipv6_range.foobar"
		instLabel := acctest.RandomWithPrefix("tf_test")

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             checkIPv6RangeDestroy,

			Steps: []resource.TestStep{
				{
					Config: tmpl.Basic(t, instLabel, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						checkIPv6RangeExistsStateCheck(resName, nil),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("prefix_length"), knownvalue.Int64Exact(64)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("is_bgp"), knownvalue.Bool(false)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("range"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("linode_id"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("linodes").AtSliceIndex(0), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("route_target"), knownvalue.NotNull()),
					},
				},
				{
					ResourceName:            resName,
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"linode_id", "route_target", "firewall_id"},
				},
			},
		})
	})
}

func TestAccIPv6Range_routeTarget(t *testing.T) {
	t.Parallel()

	acceptance.RunTestWithRetries(t, 3, func(t *acceptance.WrappedT) {
		resName := "linode_ipv6_range.foobar"
		instLabel := acctest.RandomWithPrefix("tf_test")

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             checkIPv6RangeDestroy,

			Steps: []resource.TestStep{
				{
					Config: tmpl.RouteTarget(t, instLabel, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						checkIPv6RangeExistsStateCheck(resName, nil),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("prefix_length"), knownvalue.Int64Exact(64)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("is_bgp"), knownvalue.Bool(false)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("range"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("route_target"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("linodes").AtSliceIndex(0), knownvalue.NotNull()),
					},
				},
				{
					ResourceName:            resName,
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"linode_id", "route_target", "firewall_id"},
				},
			},
		})
	})
}

func TestAccIPv6Range_noID(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkIPv6RangeDestroy,
		Steps: []resource.TestStep{
			{
				Config:      tmpl.NoID(t),
				ExpectError: regexp.MustCompile("Either linode_id or route_target must be specified."),
			},
		},
	})
}

func TestAccIPv6Range_reassignment(t *testing.T) {
	t.Parallel()

	acceptance.RunTestWithRetries(t, 3, func(t *acceptance.WrappedT) {
		resName := "linode_ipv6_range.foobar"
		instance1ResName := "linode_instance.foobar"
		instance2ResName := "linode_instance.foobar2"

		instLabel := acctest.RandomWithPrefix("tf_test")

		var instance1 linodego.Instance
		var instance2 linodego.Instance

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             checkIPv6RangeDestroy,

			Steps: []resource.TestStep{
				{
					Config: tmpl.ReassignmentStep1(t, instLabel, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						checkIPv6RangeExistsStateCheck(resName, nil),
						acceptance.StateCheckInstanceExists(instance1ResName, &instance1),
						acceptance.StateCheckInstanceExists(instance2ResName, &instance2),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("prefix_length"), knownvalue.Int64Exact(64)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("is_bgp"), knownvalue.Bool(false)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("range"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("linode_id"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("linodes").AtSliceIndex(0), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("route_target"), knownvalue.NotNull()),
					},
				},
				{
					PreConfig: func() {
						validateInstanceIPv6Assignments(t, instance1.ID, instance2.ID)
					},
					Config: tmpl.ReassignmentStep2(t, instLabel, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						checkIPv6RangeExistsStateCheck(resName, nil),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("prefix_length"), knownvalue.Int64Exact(64)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("is_bgp"), knownvalue.Bool(false)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("range"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("linode_id"), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("linodes").AtSliceIndex(0), knownvalue.NotNull()),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("route_target"), knownvalue.NotNull()),
					},
				},
				{
					Config: tmpl.ReassignmentStep2(t, instLabel, testRegion),
					PreConfig: func() {
						validateInstanceIPv6Assignments(t, instance2.ID, instance1.ID)
					},
				},
			},
		})
	})
}

func TestAccIPv6Range_raceCondition(t *testing.T) {
	t.Parallel()

	// Occasionally IPv6 range deletions take a bit to replicate
	acceptance.RunTestWithRetries(t, 3, func(t *acceptance.WrappedT) {
		instLabel := acctest.RandomWithPrefix("tf_test")

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             checkIPv6RangeDestroy,

			Steps: []resource.TestStep{
				{
					Config: tmpl.RaceCondition(t, instLabel, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						checkIPv6RangeNoDuplicatesStateCheck,
					},
				},
			},
		})
	})
}

func checkIPv6RangeExists(name string, ipv6Range *linodego.IPv6Range) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		found, err := client.GetIPv6Range(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("failed to retrieve state of ipv6 range %s: %s", rs.Primary.Attributes["range"], err)
		}

		if ipv6Range != nil {
			*ipv6Range = *found
		}

		return nil
	}
}

// checkIPv6RangeExistsStateCheck is the ConfigStateChecks-compatible equivalent of checkIPv6RangeExists.
// It verifies that an IPv6 range resource exists in the Terraform state and populates
// the provided ipv6Range pointer with the API response for downstream assertions.
func checkIPv6RangeExistsStateCheck(name string, ipv6Range *linodego.IPv6Range) statecheck.StateCheck {
	return acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		var resourceID string
		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Address == name {
				idVal, ok := rc.AttributeValues["id"]
				if !ok {
					resp.Error = fmt.Errorf("no ID is set for %s", name)
					return
				}
				resourceID = idVal.(string)
				break
			}
		}

		if resourceID == "" {
			resp.Error = fmt.Errorf("not found: %s", name)
			return
		}

		found, err := client.GetIPv6Range(context.Background(), resourceID)
		if err != nil {
			resp.Error = fmt.Errorf("failed to retrieve state of ipv6 range %s: %s", resourceID, err)
			return
		}

		if ipv6Range != nil {
			*ipv6Range = *found
		}
	})
}

func checkIPv6RangeDestroy(s *terraform.State) error {
	client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

	// We should retry here as there is sometimes a delay between deletion request and
	// range deletion. This should significantly reduce the number of intermittent cleanup
	// failures we get.
	err := resource.RetryContext(context.Background(), 30*time.Second, func() *resource.RetryError {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "linode_ipv6_range" {
				continue
			}

			_, err := client.GetIPv6Range(context.Background(), rs.Primary.ID)
			if err == nil {
				return resource.RetryableError(fmt.Errorf("ipv6 range still exists: %s", err))
			}

			if apiErr, ok := err.(*linodego.Error); ok &&
				// Intermittent error codes
				apiErr.Code != 403 && apiErr.Code != 404 && apiErr.Code != 405 {
				return resource.NonRetryableError(
					fmt.Errorf("error requesting ipv6 range with id %s: %s", rs.Primary.ID, err))
			}
		}

		return nil
	})

	return err
}

func checkIPv6RangeNoDuplicates(s *terraform.State) error {
	existingRanges := make(map[string]bool)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_ipv6_range" {
			continue
		}

		if _, ok := existingRanges[rs.Primary.ID]; ok {
			return fmt.Errorf("duplicate range found: %s", rs.Primary.ID)
		}

		existingRanges[rs.Primary.ID] = true
	}

	return nil
}

// checkIPv6RangeNoDuplicatesStateCheck is the ConfigStateChecks-compatible equivalent of
// checkIPv6RangeNoDuplicates. It verifies that no duplicate IPv6 range resources exist
// in the Terraform state by checking for unique resource IDs.
var checkIPv6RangeNoDuplicatesStateCheck = acceptance.CustomStateCheck(
	func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		existingRanges := make(map[string]bool)

		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Type != "linode_ipv6_range" {
				continue
			}

			idVal, ok := rc.AttributeValues["id"]
			if !ok {
				continue
			}

			id := idVal.(string)
			if _, exists := existingRanges[id]; exists {
				resp.Error = fmt.Errorf("duplicate range found: %s", id)
				return
			}

			existingRanges[id] = true
		}
	},
)

func validateInstanceIPv6Assignments(t testing.TB, assignedID, unassignedID int) {
	client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

	assignedNetworking, err := client.GetInstanceIPAddresses(context.Background(), assignedID)
	if err != nil {
		t.Fatal(err)
	}

	unassignedNetworking, err := client.GetInstanceIPAddresses(context.Background(), unassignedID)
	if err != nil {
		t.Fatal(err)
	}

	if len(unassignedNetworking.IPv6.Global) > 0 {
		t.Fatalf("expected instance to have no attached ipv6 ranged, got %d", len(unassignedNetworking.IPv6.Global))
	}

	if len(assignedNetworking.IPv6.Global) < 1 {
		t.Fatalf("expected instance to have one attached ipv6 ranged, got %d", len(assignedNetworking.IPv6.Global))
	}
}
