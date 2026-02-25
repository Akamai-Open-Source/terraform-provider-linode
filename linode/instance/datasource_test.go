//go:build integration || instance

package instance_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/linodego"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/instance/tmpl"
)

func TestAccDataSourceInstances_basic(t *testing.T) {
	t.Parallel()

	// Resolve a region with support for Maintenance Policy
	region, err := acceptance.GetRandomRegionWithCaps(
		[]string{"Linodes", "Maintenance Policy"},
		"core",
	)
	if err != nil {
		t.Fatal(err)
	}

	resName := "data.linode_instances.foobar"
	instanceName := acctest.RandomWithPrefix("tf_test")
	rootPass := acctest.RandString(64)
	maintenancePolicy := "linode/migrate"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t, instanceName, region, rootPass, maintenancePolicy),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances").AtSliceIndex(0).AtMapKey("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances").AtSliceIndex(0).AtMapKey("tags"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances").AtSliceIndex(0).AtMapKey("image"), knownvalue.StringExact(acceptance.TestImageLatest)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances").AtSliceIndex(0).AtMapKey("region"), knownvalue.StringExact(region)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances").AtSliceIndex(0).AtMapKey("maintenance_policy"), knownvalue.StringExact(maintenancePolicy)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances").AtSliceIndex(0).AtMapKey("group"), knownvalue.StringExact("tf_test")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances").AtSliceIndex(0).AtMapKey("swap_size"), knownvalue.StringExact("256")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances").AtSliceIndex(0).AtMapKey("disk_encryption"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances").AtSliceIndex(0).AtMapKey("ipv4"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances").AtSliceIndex(0).AtMapKey("ipv6"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances").AtSliceIndex(0).AtMapKey("host_uuid"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances").AtSliceIndex(0).AtMapKey("has_user_data"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances").AtSliceIndex(0).AtMapKey("disk"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances").AtSliceIndex(0).AtMapKey("config"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances").AtSliceIndex(0).AtMapKey("config").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances").AtSliceIndex(0).AtMapKey("placement_group"), knownvalue.ListSizeExact(0)),
				},
			},
		},
	})
}

func TestAccDataSourceInstances_withBlockStorageEncryption(t *testing.T) {
	t.Parallel()

	resName := "data.linode_instances.foobar"
	instanceName := acctest.RandomWithPrefix("tf_test")

	// Resolve a region with support for Block Storage Encryption
	targetRegion, err := acceptance.GetRandomRegionWithCaps(
		[]string{"Linodes", "Block Storage Encryption"},
		"core",
	)
	if err != nil {
		t.Fatal(err)
	}

	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.DataWithBlockStorageEncryption(t, instanceName, targetRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
						for _, rc := range req.State.Values.RootModule.Resources {
							if rc.Address != resName {
								continue
							}
							instances, ok := rc.AttributeValues["instances"].([]interface{})
							if !ok || len(instances) == 0 {
								resp.Error = fmt.Errorf("instances attribute not found or empty")
								return
							}
							inst := instances[0].(map[string]interface{})
							capabilities, ok := inst["capabilities"].([]interface{})
							if !ok {
								resp.Error = fmt.Errorf("capabilities attribute not found")
								return
							}
							for _, cap := range capabilities {
								if cap.(string) == "Block Storage Encryption" {
									return
								}
							}
							resp.Error = fmt.Errorf("Block Storage Encryption not found in capabilities")
							return
						}
						resp.Error = fmt.Errorf("resource %s not found", resName)
					}),
				},
			},
		},
	})
}

func TestAccDataSourceInstances_withPG(t *testing.T) {
	t.Parallel()

	resName := "data.linode_instances.foobar"
	instanceName := acctest.RandomWithPrefix("tf_test")

	pgIDs := []string{"foobar"}

	// Resolve a region with support for PGs
	targetRegion, err := acceptance.GetRandomRegionWithCaps(
		[]string{"Linodes", "Placement Group"},
		"core",
	)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.DataWithPG(t, instanceName, targetRegion, "foobar", pgIDs),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances").AtSliceIndex(0).AtMapKey("placement_group"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances").AtSliceIndex(0).AtMapKey("placement_group").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances").AtSliceIndex(0).AtMapKey("placement_group").AtSliceIndex(0).AtMapKey("placement_group_type"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances").AtSliceIndex(0).AtMapKey("placement_group").AtSliceIndex(0).AtMapKey("placement_group_policy"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func TestAccDataSourceInstances_multipleInstances(t *testing.T) {
	resName := "data.linode_instances.foobar"
	resNameDesc := "data.linode_instances.desc"
	resNameAsc := "data.linode_instances.asc"

	instanceName := acctest.RandomWithPrefix("tf_test")
	tagName := acctest.RandomWithPrefix("tf_test")
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataMultiple(t, instanceName, tagName, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances"), knownvalue.ListSizeExact(3)),
				},
			},
			{
				Config: tmpl.DataMultipleOrder(t, instanceName, tagName, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					// Ensure order is correctly appended to filter
					statecheck.ExpectKnownValue(resNameDesc, tfjsonpath.New("instances"), knownvalue.ListSizeExact(3)),
					statecheck.ExpectKnownValue(resNameAsc, tfjsonpath.New("instances"), knownvalue.ListSizeExact(3)),
				},
			},
			{
				Config: tmpl.DataMultipleRegex(t, instanceName, tagName, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances"), knownvalue.ListSizeExact(3)),
				},
			},
			{
				Config: tmpl.DataClientFilter(t, instanceName, tagName, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances").AtSliceIndex(0).AtMapKey("status"), knownvalue.StringExact("running")),
				},
			},
		},
	})
}

func TestAccDataSourceInstances_explicitInterfaceGeneration(t *testing.T) {
	t.Parallel()

	resName := "data.linode_instances.foobar"
	instanceName := acctest.RandomWithPrefix("tf_test")

	firstInstancePath := tfjsonpath.New("instances").AtSliceIndex(0)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.DataExplicitInterfaceGeneration(
					t,
					instanceName,
					testRegion,
					acceptance.TestImageLatest,
					linodego.GenerationLinode,
					false,
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("instances"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						resName,
						firstInstancePath.AtMapKey("label"),
						knownvalue.StringExact(instanceName),
					),
					statecheck.ExpectKnownValue(
						resName,
						firstInstancePath.AtMapKey("type"),
						knownvalue.StringExact("g6-nanode-1"),
					),
					statecheck.ExpectKnownValue(
						resName,
						firstInstancePath.AtMapKey("image"),
						knownvalue.StringExact(acceptance.TestImageLatest),
					),
					statecheck.ExpectKnownValue(
						resName,
						firstInstancePath.AtMapKey("region"),
						knownvalue.StringExact(testRegion),
					),
					statecheck.ExpectKnownValue(
						resName,
						firstInstancePath.AtMapKey("interface_generation"),
						knownvalue.StringExact(string(linodego.GenerationLinode)),
					),
				},
			},
		},
	})
}

func TestAccDataSourceInstance_interfaceVPCIPv6(t *testing.T) {
	t.Parallel()

	dataSourceName := "data.linode_instances.foobar"
	instanceName := acctest.RandomWithPrefix("tf-test")
	rootPass := acctest.RandString(64)

	// TODO (VPC Dual Stack): Remove region hardcoding
	targetRegion := "no-osl-1"

	ipv6Path := tfjsonpath.New("instances").
		AtSliceIndex(0).
		AtMapKey("config").
		AtSliceIndex(0).
		AtMapKey("interface").
		AtSliceIndex(0).
		AtMapKey("ipv6").
		AtSliceIndex(0)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataInterfacesVPCIPv6(
					t,
					instanceName,
					targetRegion,
					rootPass,
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						dataSourceName,
						ipv6Path.AtMapKey("slaac"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						dataSourceName,
						ipv6Path.
							AtMapKey("slaac").
							AtSliceIndex(0).
							AtMapKey("range"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						dataSourceName,
						ipv6Path.
							AtMapKey("slaac").
							AtSliceIndex(0).
							AtMapKey("assigned_range"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						dataSourceName,
						ipv6Path.
							AtMapKey("slaac").
							AtSliceIndex(0).
							AtMapKey("address"),
						knownvalue.NotNull(),
					),

					statecheck.ExpectKnownValue(
						dataSourceName,
						ipv6Path.AtMapKey("range"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						dataSourceName,
						ipv6Path.
							AtMapKey("range").
							AtSliceIndex(0).
							AtMapKey("assigned_range"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						dataSourceName,
						ipv6Path.
							AtMapKey("range").
							AtSliceIndex(0).
							AtMapKey("range"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccDataSourceInstances_withLock(t *testing.T) {
	t.Parallel()

	resName := "data.linode_instances.test"
	label := acctest.RandomWithPrefix("tf_test")
	lockType := "cannot_delete"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataWithLock(t, label, testRegion, lockType),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact(label)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("instances").AtSliceIndex(0).AtMapKey("locks"), knownvalue.ListSizeExact(1)),
					acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
						for _, rc := range req.State.Values.RootModule.Resources {
							if rc.Address != resName {
								continue
							}
							instances, ok := rc.AttributeValues["instances"].([]interface{})
							if !ok || len(instances) == 0 {
								resp.Error = fmt.Errorf("instances attribute not found or empty")
								return
							}
							inst := instances[0].(map[string]interface{})
							locks, ok := inst["locks"].([]interface{})
							if !ok {
								resp.Error = fmt.Errorf("locks attribute not found")
								return
							}
							for _, lock := range locks {
								if lock.(string) == lockType {
									return
								}
							}
							resp.Error = fmt.Errorf("%s not found in locks", lockType)
							return
						}
						resp.Error = fmt.Errorf("resource %s not found", resName)
					}),
				},
			},
		},
	})
}
