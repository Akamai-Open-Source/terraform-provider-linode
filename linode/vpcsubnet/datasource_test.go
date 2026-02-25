//go:build integration || vpcsubnet

package vpcsubnet_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/vpcsubnet/tmpl"
)

func TestAccDataSourceVPCSubnet_basic(t *testing.T) {
	t.Parallel()

	resourceName := "data.linode_vpc_subnet.foo"
	subnetLabel := acctest.RandomWithPrefix("tf-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t, subnetLabel, "10.0.0.0/24", testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("label"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("ipv4"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("created"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("updated"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("linodes"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("linodes").AtSliceIndex(0).AtMapKey("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("linodes").AtSliceIndex(0).AtMapKey("interfaces"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("linodes").AtSliceIndex(0).AtMapKey("interfaces").AtSliceIndex(0).AtMapKey("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("linodes").AtSliceIndex(0).AtMapKey("interfaces").AtSliceIndex(0).AtMapKey("active"),
						knownvalue.StringExact("false"),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("databases"),
						knownvalue.ListSizeExact(0),
					),
				},
			},
		},
	})
}

func TestAccDataSourceVPCSubnet_dualStack(t *testing.T) {
	t.Parallel()

	resourceName := "data.linode_vpc_subnet.foo"
	subnetLabel := acctest.RandomWithPrefix("tf-test")

	// TODO (VPC Dual Stack): Remove region hardcoding
	targetRegion := "no-osl-1"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataDualStack(t, subnetLabel, "10.0.0.0/24", targetRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("label"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("ipv4"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("created"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("updated"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("ipv6"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("ipv6").AtSliceIndex(0).AtMapKey("range"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}
