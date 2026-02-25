//go:build integration || regions

package regions_test

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/regions/tmpl"
)

func TestSmokeTests_firewall(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{"TestAccDataSourceRegions_basic_smoke", TestAccDataSourceRegions_basic_smoke},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestAccDataSourceRegions_basic_smoke(t *testing.T) {
	t.Parallel()

	resourceName := "data.linode_regions.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("country"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("label"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("status"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("site_type"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("resolvers").AtSliceIndex(0).AtMapKey("ipv4"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("resolvers").AtSliceIndex(0).AtMapKey("ipv6"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("placement_group_limits").AtSliceIndex(0).AtMapKey("maximum_pgs_per_customer"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("placement_group_limits").AtSliceIndex(0).AtMapKey("maximum_linodes_per_pg"),
						knownvalue.NotNull(),
					),
					acceptance.StateCheckResourceAttrGreaterThan(resourceName, "regions.0.capabilities.#", 0),
					acceptance.StateCheckResourceAttrGreaterThan(resourceName, "regions.#", 0),
				},
			},
		},
	})
}

func TestAccDataSourceRegions_filterByCountry(t *testing.T) {
	t.Parallel()

	resourceName := "data.linode_regions.foobar"

	client, err := acceptance.GetTestClient()
	if err != nil {
		t.Fail()
		t.Log("Failed to get testing client.")
	}

	regions, err := client.ListRegions(context.TODO(), nil)
	randIndex := rand.Intn(len(regions))
	region := regions[randIndex]

	country := region.Country
	status := region.Status
	capabilities := region.Capabilities

	randomCapability := capabilities[rand.Intn(len(capabilities))]

	if err != nil {
		t.Fail()
		t.Log("Failed to get testing region.")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataFilterCountry(t, country, status, randomCapability),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("country"), knownvalue.StringExact(country)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("label"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("status"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("site_type"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("resolvers").AtSliceIndex(0).AtMapKey("ipv4"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("resolvers").AtSliceIndex(0).AtMapKey("ipv6"), knownvalue.NotNull()),
					acceptance.StateCheckResourceAttrGreaterThan(resourceName, "regions.0.capabilities.#", 0),
					acceptance.StateCheckResourceAttrGreaterThan(resourceName, "regions.#", 0),
				},
			},
		},
	})
}

func TestAccDataSourceRegions_filterByStatus(t *testing.T) {
	t.Parallel()

	resourceName := "data.linode_regions.foobar"

	client, err := acceptance.GetTestClient()
	if err != nil {
		t.Fail()
		t.Log("Failed to get testing client.")
	}

	regions, err := client.ListRegions(context.TODO(), nil)
	randIndex := rand.Intn(len(regions))
	region := regions[randIndex]

	country := region.Country
	status := region.Status
	capabilities := region.Capabilities

	randomCapability := capabilities[rand.Intn(len(capabilities))]

	if err != nil {
		t.Fail()
		t.Log("Failed to get testing region.")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataFilterStatus(t, country, status, randomCapability),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("country"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("label"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("status"), knownvalue.StringExact(status)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("site_type"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("resolvers").AtSliceIndex(0).AtMapKey("ipv4"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("resolvers").AtSliceIndex(0).AtMapKey("ipv6"), knownvalue.NotNull()),
					acceptance.StateCheckResourceAttrGreaterThan(resourceName, "regions.0.capabilities.#", 0),
					acceptance.StateCheckResourceAttrGreaterThan(resourceName, "regions.#", 0),
				},
			},
		},
	})
}

func TestAccDataSourceRegions_filterByCapabilities(t *testing.T) {
	t.Parallel()

	resourceName := "data.linode_regions.foobar"

	client, err := acceptance.GetTestClient()
	if err != nil {
		t.Fail()
		t.Log("Failed to get testing client.")
	}

	regions, err := client.ListRegions(context.TODO(), nil)
	randIndex := rand.Intn(len(regions))
	region := regions[randIndex]

	country := region.Country
	status := region.Status
	capabilities := region.Capabilities

	randomCapability := capabilities[rand.Intn(len(capabilities))]

	if err != nil {
		t.Fail()
		t.Log("Failed to get testing region.")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataFilterCapabilities(t, country, status, randomCapability),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("country"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("label"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("status"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("site_type"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("resolvers").AtSliceIndex(0).AtMapKey("ipv4"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("regions").AtSliceIndex(0).AtMapKey("resolvers").AtSliceIndex(0).AtMapKey("ipv6"), knownvalue.NotNull()),
					acceptance.StateCheckResourceAttrGreaterThan(resourceName, "regions.0.capabilities.#", 0),
					acceptance.StateCheckResourceAttrGreaterThan(resourceName, "regions.#", 0),
					acceptance.StateCheckLoopThroughStringList(resourceName, "regions", func(resourceName, path string, resources []*tfjson.StateResource) error {
						for _, rc := range resources {
							if rc.Address != resourceName {
								continue
							}

							// path is like "regions.0", "regions.1", etc.
							// Extract the index to navigate the nested tfjson structure
							parts := strings.Split(path, ".")
							if len(parts) < 2 {
								return fmt.Errorf("invalid path: %s", path)
							}
							idx, err := strconv.Atoi(parts[len(parts)-1])
							if err != nil {
								return fmt.Errorf("error parsing index from path %s: %s", path, err)
							}

							regionsVal, ok := rc.AttributeValues["regions"]
							if !ok {
								return fmt.Errorf("attribute regions does not exist")
							}
							regionsList, ok := regionsVal.([]interface{})
							if !ok {
								return fmt.Errorf("attribute regions is not a list")
							}
							if idx >= len(regionsList) {
								return fmt.Errorf("index %d out of bounds for regions (len=%d)", idx, len(regionsList))
							}
							regionMap, ok := regionsList[idx].(map[string]interface{})
							if !ok {
								return fmt.Errorf("region element at index %d is not a map", idx)
							}
							capsVal, ok := regionMap["capabilities"]
							if !ok {
								return fmt.Errorf("capabilities not found in region %d", idx)
							}
							caps, ok := capsVal.([]interface{})
							if !ok {
								return fmt.Errorf("capabilities is not a list in region %d", idx)
							}
							for _, c := range caps {
								if fmt.Sprintf("%v", c) == randomCapability {
									return nil
								}
							}
							return fmt.Errorf("capability %s not found in region %d capabilities", randomCapability, idx)
						}
						return fmt.Errorf("Not found: %s", resourceName)
					}),
				},
			},
		},
	})
}
