//go:build integration || instance

package instance_test

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/linodego"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/helper"
	"github.com/linode/terraform-provider-linode/v3/linode/instance/tmpl"
	"github.com/stretchr/testify/require"
)

var testRegion string

func init() {
	resource.AddTestSweepers("linode_instance", &resource.Sweeper{
		Name: "linode_instance",
		F:    sweep,
	})

	region, err := acceptance.GetRandomRegionWithCaps([]string{
		linodego.CapabilityVlans, linodego.CapabilityVPCs, linodego.CapabilityDiskEncryption,
	}, "core")
	if err != nil {
		log.Fatal(err)
	}

	testRegion = region
}

func sweep(prefix string) error {
	client, err := acceptance.GetTestClient()
	if err != nil {
		return fmt.Errorf("Error getting client: %s", err)
	}

	listOpts := acceptance.SweeperListOptions(prefix, "label")
	instances, err := client.ListInstances(context.Background(), listOpts)
	if err != nil {
		return fmt.Errorf("Error getting instances: %s", err)
	}
	for _, instance := range instances {
		if !acceptance.ShouldSweep(prefix, instance.Label) {
			continue
		}
		err := client.DeleteInstance(context.Background(), instance.ID)
		if err != nil {
			return fmt.Errorf("Error destroying %s during sweep: %s", instance.Label, err)
		}
	}

	return nil
}

func TestSmokeTests_instance(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{"TestAccResourceInstance_basic_smoke", TestAccResourceInstance_basic_smoke},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestAccResourceInstance_basic_smoke(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, instanceName, acceptance.PublicKeyMaterial, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("image"), knownvalue.StringExact(acceptance.TestImageLatest)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("group"), knownvalue.StringExact("tf_test")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("swap_size"), knownvalue.StringExact("256")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("host_uuid"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("ipv6"), knownvalue.StringRegexp(regexp.MustCompile(`/128$`))),
				},
			},

			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"root_pass", "authorized_keys", "image", "resize_disk", "migration_type", "firewall_id", "capabilities"},
			},
		},
	})
}

func TestAccResourceInstance_vpu(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test_vpu")
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.VPU(t, instanceName, acceptance.PublicKeyMaterial, "us-lax", rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g1-accelerated-netint-vpu-t1u1-s")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("image"), knownvalue.StringExact(acceptance.TestImageLatest)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact("us-lax")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("group"), knownvalue.StringExact("tf_test")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("swap_size"), knownvalue.StringExact("256")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("host_uuid"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("specs").AtSliceIndex(0).AtMapKey("accelerated_devices"), knownvalue.NotNull()),
				},
			},

			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"root_pass", "authorized_keys", "image", "resize_disk", "migration_type", "firewall_id", "capabilities"},
			},
		},
	})
}

func TestAccResourceInstance_watchdogDisabled(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	instanceName := acctest.RandomWithPrefix("tf_test")
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.WatchdogDisabled(t, instanceName, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("watchdog_enabled"), knownvalue.StringExact("false")),
				},
			},
			{
				Config:   tmpl.WatchdogDisabled(t, instanceName, testRegion, rootPass),
				PlanOnly: true,
			},
		},
	})
}

func TestAccResourceInstance_authorizedUsers(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")
	rootPass := acctest.RandString(64)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.AuthorizedUsers(t, instanceName, acceptance.PublicKeyMaterial, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("image"), knownvalue.StringExact(acceptance.TestImageLatest)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("group"), knownvalue.StringExact("tf_test")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("swap_size"), knownvalue.StringExact("256")),
				},
			},

			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"root_pass", "authorized_users", "image", "resize_disk", "migration_type", "firewall_id", "capabilities"},
			},
		},
	})
}

func TestAccResourceInstance_validateAuthorizedKeys(t *testing.T) {
	t.Parallel()

	instanceName := acctest.RandomWithPrefix("tf_test")
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.AuthorizedKeysEmpty(t, instanceName, testRegion),
				ExpectError: regexp.MustCompile(
					"invalid input for authorized_keys"),
			},
			{
				Config: tmpl.DiskAuthorizedKeysEmpty(t, instanceName, testRegion, rootPass),
				ExpectError: regexp.MustCompile(
					"invalid input for disk authorized_keys"),
			},
		},
	})
}

func TestAccResourceInstance_interfaces(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.Interfaces(t, instanceName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("group"), knownvalue.StringExact("tf_test")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("image"), knownvalue.StringExact(acceptance.TestImageLatest)),

					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface"), knownvalue.ListSizeExact(1)),

					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface").AtSliceIndex(0).AtMapKey("purpose"), knownvalue.StringExact("vlan")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("tf-really-cool-vlan")),
				},
			},
			{
				Config: tmpl.InterfacesUpdate(t, instanceName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface"), knownvalue.ListSizeExact(2)),

					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface").AtSliceIndex(0).AtMapKey("purpose"), knownvalue.StringExact("public")),

					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface").AtSliceIndex(1).AtMapKey("purpose"), knownvalue.StringExact("vlan")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface").AtSliceIndex(1).AtMapKey("label"), knownvalue.StringExact("tf-really-cool-vlan")),
				},
			},
			{
				Config: tmpl.InterfacesUpdateEmpty(t, instanceName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface"), knownvalue.ListSizeExact(1)),
				},
			},
			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"image", "interface", "resize_disk", "migration_type", "firewall_id"},
			},
		},
	})
}

func TestAccResourceInstance_config(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.WithConfig(t, instanceName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					// resource.TestCheckResourceAttr(resName, "kernel", "linode/latest-64bit"),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("group"), knownvalue.StringExact("tf_test")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("swap_size"), knownvalue.StringExact("0")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("alerts").AtSliceIndex(0).AtMapKey("cpu"), knownvalue.StringExact("60")),

					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("run_level"), knownvalue.StringExact("binbash")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("virt_mode"), knownvalue.StringExact("fullvirt")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("memory_limit"), knownvalue.StringExact("1024")),

					stateCheckComputeInstanceConfigs(&instance, testConfig("config", testConfigKernel("linode/latest-64bit"))),
				},
			},

			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"resize_disk", "migration_type", "firewall_id"},
			},
		},
	})
}

func TestAccResourceInstance_configPair(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.MultipleConfigs(t, instanceName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					// resource.TestCheckResourceAttr(resName, "kernel", "linode/latest-64bit"),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("group"), knownvalue.StringExact("tf_test")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("swap_size"), knownvalue.StringExact("0")),
					stateCheckComputeInstanceConfigs(&instance, testConfig("configa", testConfigKernel("linode/latest-64bit"))),
					stateCheckComputeInstanceConfigs(&instance, testConfig("configb", testConfigKernel("linode/latest-32bit"))),
				},
			},

			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"boot_config_label", "resize_disk", "migration_type", "firewall_id"},
			},
		},
	})
}

func TestAccResourceInstance_configInterfaces(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.ConfigInterfaces(t, instanceName, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("group"), knownvalue.StringExact("tf_test")),

					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface").AtSliceIndex(0).AtMapKey("purpose"), knownvalue.StringExact("vlan")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("tf-really-cool-vlan")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("config")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("boot_config_label"), knownvalue.StringExact("config")),
				},
			},
			{
				Config: tmpl.ConfigInterfacesMultiple(t, instanceName, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface").AtSliceIndex(0).AtMapKey("purpose"), knownvalue.StringExact("vlan")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("tf-really-cool-vlan")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("config")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(1).AtMapKey("interface"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("boot_config_label"), knownvalue.StringExact("config")),
				},
			},
			{
				PreConfig: testAccAssertReboot(t, true, &instance),
				Config:    tmpl.ConfigInterfacesUpdate(t, instanceName, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("config")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("boot_config_label"), knownvalue.StringExact("config")),
				},
			},
			{
				PreConfig: testAccAssertReboot(t, true, &instance),
				Config:    tmpl.ConfigInterfacesUpdateEmpty(t, instanceName, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface"), knownvalue.ListSizeExact(0)),
				},
			},
			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"resize_disk", "migration_type", "firewall_id"},
			},
		},
	})
}

func TestAccResourceInstance_configInterfacesNoReboot(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.ConfigInterfaces(t, instanceName, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("group"), knownvalue.StringExact("tf_test")),

					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface").AtSliceIndex(0).AtMapKey("purpose"), knownvalue.StringExact("vlan")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("tf-really-cool-vlan")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("config")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("boot_config_label"), knownvalue.StringExact("config")),
				},
			},
			{
				Config: tmpl.ConfigInterfacesUpdateNoReboot(t, instanceName, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface"), knownvalue.ListSizeExact(0)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("config")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("boot_config_label"), knownvalue.StringExact("config")),
				},
			},
			{
				PreConfig: testAccAssertReboot(t, false, &instance),
				Config:    tmpl.ConfigInterfacesUpdateNoReboot(t, instanceName, testRegion, rootPass),
			},
			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"resize_disk", "boot_config_label", "migration_type", "firewall_id"},
			},
		},
	})
}

func testAccAssertReboot(t *testing.T, shouldRestart bool, instance *linodego.Instance) func() {
	return func() {
		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client
		eventFilter := fmt.Sprintf(`{"entity.type": "linode", "entity.id": %d, "action": "linode_reboot", "created": { "+gte": "%s" }}`,
			instance.ID, instance.Created.Format("2006-01-02T15:04:05"))

		events, err := client.ListEvents(context.Background(), &linodego.ListOptions{Filter: eventFilter})
		if err != nil {
			t.Fail()
		}

		if len(events) == 0 && shouldRestart {
			t.Fatal("expected instance to have been rebooted")
		}

		if len(events) > 0 && !shouldRestart {
			t.Fatal("expected instance to not have been rebooted")
		}
	}
}

func TestAccResourceInstance_disk(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.RawDisk(t, instanceName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("group"), knownvalue.StringExact("tf_test")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("swap_size"), knownvalue.StringExact("0")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("status"), knownvalue.StringExact("offline")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config"), knownvalue.ListSizeExact(0)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk").AtSliceIndex(0).AtMapKey("size"), knownvalue.StringExact("3000")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("disk")),
					stateCheckComputeInstanceDisk(&instance, "disk", 3000),
				},
			},
			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"resize_disk", "migration_type", "firewall_id"},
			},
		},
	})
}

func TestAccResourceInstance_diskImage(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.Disk(t, instanceName, acceptance.PublicKeyMaterial, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					// resource.TestCheckResourceAttr(resName, "kernel", "linode/latest-64bit"),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("group"), knownvalue.StringExact("tf_test")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("swap_size"), knownvalue.StringExact("0")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk").AtSliceIndex(0).AtMapKey("size"), knownvalue.StringExact("3000")),
					stateCheckComputeInstanceDisk(&instance, "disk", 3000),
				},
			},

			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"resize_disk", "migration_type", "firewall_id"},
			},
		},
	})
}

func TestAccResourceInstance_diskPair(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	var instanceDisk linodego.InstanceDisk
	instanceName := acctest.RandomWithPrefix("tf_test")
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.DiskMultiple(t, instanceName, acceptance.PublicKeyMaterial, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					// resource.TestCheckResourceAttr(resName, "kernel", "linode/latest-64bit"),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("group"), knownvalue.StringExact("tf_test")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("swap_size"), knownvalue.StringExact("512")),
					stateCheckInstanceDisks(&instance, testDisk("diska", testDiskSize(3000), testDiskExists(&instanceDisk)), testDisk("diskb", testDiskSize(512)), ),
				},
			},

			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"resize_disk", "migration_type", "firewall_id", "available"},
			},
		},
	})
}

func TestAccResourceInstance_diskAndConfig(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.DiskConfig(t, instanceName, acceptance.PublicKeyMaterial, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					// resource.TestCheckResourceAttr(resName, "kernel", "linode/latest-64bit"),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("group"), knownvalue.StringExact("tf_test")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("swap_size"), knownvalue.StringExact("0")),
					stateCheckComputeInstanceConfigs(&instance, testConfig("config", testConfigKernel("linode/latest-64bit")), ),
					stateCheckComputeInstanceDisk(&instance, "disk", 3000),
				},
			},

			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"resize_disk", "migration_type", "firewall_id"},
			},
		},
	})
}

func TestAccResourceInstance_disksAndConfigs(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	var instanceDisk linodego.InstanceDisk

	instanceName := acctest.RandomWithPrefix("tf_test")

	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			acceptance.CheckInstanceDestroy,
			acceptance.CheckVolumeDestroy,
		),
		ExternalProviders: acceptance.HttpExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DiskConfigMultiple(t, instanceName, acceptance.PublicKeyMaterial, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					// resource.TestCheckResourceAttr(resName, "kernel", "linode/latest-64bit"),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("group"), knownvalue.StringExact("tf_test")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("swap_size"), knownvalue.StringExact("512")),
					stateCheckInstanceDiskExists(&instance, "diska", &instanceDisk),
					// TODO(displague) create checkInstanceDisks helper (like Configs)
					stateCheckComputeInstanceDisk(&instance, "diska", 3000),
					stateCheckComputeInstanceDisk(&instance, "diskb", 512),
					stateCheckComputeInstanceConfigs(&instance, testConfig("configa", testConfigKernel("linode/latest-64bit"), testConfigSDADisk(&instanceDisk)), testConfig("configb", testConfigKernel("linode/grub2"), testConfigComments("won't boot"), testConfigSDBDisk(&instanceDisk)), ),
				},
			},

			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"boot_config_label", "resize_disk", "migration_type", "firewall_id"},
			},
		},
	})
}

func TestAccResourceInstance_volumeAndConfig(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	volName := "linode_volume.foo"

	var instance linodego.Instance
	var instanceDisk linodego.InstanceDisk
	var volume linodego.Volume
	instanceName := acctest.RandomWithPrefix("tf_test")
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.VolumeConfig(t, instanceName, acceptance.PublicKeyMaterial, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					acceptance.StateCheckVolumeExists(volName, &volume),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					// resource.TestCheckResourceAttr(resName, "kernel", "linode/latest-64bit"),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("group"), knownvalue.StringExact("tf_test")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("boot_config_label"), knownvalue.StringExact("config")),
					stateCheckInstanceDiskExists(&instance, "disk", &instanceDisk),
					// TODO(displague) create checkInstanceDisks helper (like Configs)
					stateCheckComputeInstanceDisk(&instance, "disk", 3000),
					stateCheckComputeInstanceConfigs(&instance, testConfig("config", testConfigKernel("linode/latest-64bit"), testConfigSDADisk(&instanceDisk), testConfigSDBVolume(&volume)), ),
				},
			},

			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"resize_disk", "migration_type", "firewall_id"},
			},
		},
	})
}

func TestAccResourceInstance_privateImage(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.PrivateImage(t, instanceName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("group"), knownvalue.StringExact("tf_test")),
					stateCheckInstanceDisks(&instance, testDisk("boot", testDiskSize(1000)), testDisk("swap", testDiskSize(800)), testDisk("logs", testDiskSize(600)), ),
				},
			},

			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"resize_disk", "migration_type", "firewall_id"},
			},
		},
	})
}

func TestAccResourceInstance_noImage(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.NoImage(t, instanceName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("group"), knownvalue.StringExact("tf_test")),
				},
			},

			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"resize_disk", "migration_type", "firewall_id"},
			},
		},
	})
}

func TestAccResourceInstance_updateSimple(t *testing.T) {
	t.Parallel()
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")
	resName := "linode_instance.foobar"
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, instanceName, acceptance.PublicKeyMaterial, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("group"), knownvalue.StringExact("tf_test")),
				},
			},
			{
				Config: tmpl.Updates(t, instanceName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(fmt.Sprintf("%s_r", instanceName))),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("group"), knownvalue.StringExact("tf_test_r")),
				},
			},
		},
	})
}

func TestAccResourceInstance_updateMaintenancePolicy(t *testing.T) {
	t.Parallel()
	var instance linodego.Instance

	region, err := acceptance.GetRandomRegionWithCaps([]string{"Linodes", "Maintenance Policy"}, "core")
	require.NoError(t, err)

	instanceName := acctest.RandomWithPrefix("tf_test")
	resName := "linode_instance.foobar"
	rootPass := acctest.RandString(64)
	maintenancePolicyMigrate := "linode/migrate"
	maintenancePolicyPowerOnOff := "linode/power_off_on"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.MaintenancePolicy(t, instanceName, acceptance.PublicKeyMaterial, region, rootPass, maintenancePolicyMigrate),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("maintenance_policy"), knownvalue.StringExact(maintenancePolicyMigrate)),
				},
			},
			{
				Config: tmpl.MaintenancePolicy(t, instanceName, acceptance.PublicKeyMaterial, region, rootPass, maintenancePolicyPowerOnOff),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("maintenance_policy"), knownvalue.StringExact(maintenancePolicyPowerOnOff)),
				},
			},
		},
	})
}

func TestAccResourceInstance_configUpdate(t *testing.T) {
	t.Parallel()
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")
	resName := "linode_instance.foobar"

	// This test can occasionally fail while running the entire test suite in parallel
	acceptance.RunTestWithRetries(t, 3, func(t *acceptance.WrappedT) {
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             acceptance.CheckInstanceDestroy,

			Steps: []resource.TestStep{
				{
					Config: tmpl.WithConfig(t, instanceName, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						acceptance.StateCheckInstanceExists(resName, &instance),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("group"), knownvalue.StringExact("tf_test")),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("kernel"), knownvalue.StringExact("linode/latest-64bit")),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("root_device"), knownvalue.StringExact("/dev/sda")),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("helpers").AtSliceIndex(0).AtMapKey("network"), knownvalue.StringExact("true")),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("alerts").AtSliceIndex(0).AtMapKey("cpu"), knownvalue.StringExact("60")),
					},
				},
				{
					Config: tmpl.ConfigUpdates(t, instanceName, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						acceptance.StateCheckInstanceExists(resName, &instance),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(fmt.Sprintf("%s_r", instanceName))),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("group"), knownvalue.StringExact("tf_test_r")),
						// changed kernel, not label
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("config")),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("kernel"), knownvalue.StringExact("linode/latest-32bit")),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("root_device"), knownvalue.StringExact("/dev/sda")),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("helpers").AtSliceIndex(0).AtMapKey("network"), knownvalue.StringExact("false")),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("alerts").AtSliceIndex(0).AtMapKey("cpu"), knownvalue.StringExact("80")),
					},
				},
			},
		})
	})
}

func TestAccResourceInstance_configPairUpdate(t *testing.T) {
	t.Parallel()

	config := linodego.InstanceConfig{}
	configA := linodego.InstanceConfig{}
	configB := linodego.InstanceConfig{}

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.WithConfig(t, instanceName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("group"), knownvalue.StringExact("tf_test")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("config")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("kernel"), knownvalue.StringExact("linode/latest-64bit")),
					stateCheckComputeInstanceConfigs(&instance, testConfig("config", testConfigExists(&config), testConfigKernel("linode/latest-64bit")), ),
				},
			},
			{
				Config: tmpl.MultipleConfigs(t, instanceName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					// resource.TestCheckResourceAttr(resName, "kernel", "linode/latest-64bit"),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("configa")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("kernel"), knownvalue.StringExact("linode/latest-64bit")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(1).AtMapKey("label"), knownvalue.StringExact("configb")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(1).AtMapKey("kernel"), knownvalue.StringExact("linode/latest-32bit")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("group"), knownvalue.StringExact("tf_test")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("swap_size"), knownvalue.StringExact("0")),
					stateCheckComputeInstanceConfigs(&instance, testConfig("configa", testConfigExists(&configA), testConfigKernel("linode/latest-64bit")), testConfig("configb", testConfigExists(&configB), testConfigKernel("linode/latest-32bit")), ),
				},
			},
			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"boot_config_label", "status", "resize_disk", "migration_type", "firewall_id"},
			},
			{
				Config: tmpl.WithConfig(t, instanceName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("group"), knownvalue.StringExact("tf_test")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("config")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("kernel"), knownvalue.StringExact("linode/latest-64bit")),
					stateCheckComputeInstanceConfigs(&instance, testConfig("config", testConfigExists(&config), testConfigKernel("linode/latest-64bit")), ),
				},
			},
			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"boot_config_label", "status", "resize_disk", "migration_type", "firewall_id"},
			},
			{
				Config: tmpl.ConfigsAllUpdated(t, instanceName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					// resource.TestCheckResourceAttr(resName, "kernel", "linode/latest-64bit"),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("group"), knownvalue.StringExact("tf_test")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("swap_size"), knownvalue.StringExact("0")),
					stateCheckComputeInstanceConfigs(&instance, testConfig("configb", testConfigKernel("linode/latest-64bit")), testConfig("configa", testConfigKernel("linode/latest-32bit")), testConfig("configc", testConfigKernel("linode/latest-64bit")), ),
				},
			},
		},
	})
}

func TestAccResourceInstance_upsizeWithoutDisk(t *testing.T) {
	t.Parallel()
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")
	resName := "linode_instance.foobar"
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.WithType(t, instanceName, acceptance.PublicKeyMaterial, "g6-nanode-1", testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("specs").AtSliceIndex(0).AtMapKey("disk"), knownvalue.StringExact("25600")),
					stateCheckInstanceDisks(&instance, testDiskByFS(linodego.FilesystemExt4, testDiskSize(25344)), testDiskByFS(linodego.FilesystemSwap, testDiskSize(256)), ),
				},
			},
			{
				Config: tmpl.WithType(t, instanceName, acceptance.PublicKeyMaterial, "g6-standard-1", testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("specs").AtSliceIndex(0).AtMapKey("disk"), knownvalue.StringExact("51200")),
					stateCheckInstanceDisks(&instance, testDiskByFS(linodego.FilesystemExt4, testDiskSize(25344)), testDiskByFS(linodego.FilesystemSwap, testDiskSize(256)), ),
				},
			},
		},
	})
}

func TestAccResourceInstance_diskRawResize(t *testing.T) {
	t.Parallel()
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")
	resName := "linode_instance.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			// Start off with a Linode 1024
			{
				Config: tmpl.RawDisk(t, instanceName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("specs").AtSliceIndex(0).AtMapKey("disk"), knownvalue.StringExact("25600")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config"), knownvalue.ListSizeExact(0)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk").AtSliceIndex(0).AtMapKey("size"), knownvalue.StringExact("3000")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("disk")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("swap_size"), knownvalue.StringExact("0")),
					stateCheckInstanceDisks(&instance, testDisk("disk", testDiskSize(3000))),
				},
			},
			// Bump it to a 2048, and expand the disk
			{
				Config: tmpl.RawDiskExpanded(t, instanceName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("specs").AtSliceIndex(0).AtMapKey("disk"), knownvalue.StringExact("51200")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config"), knownvalue.ListSizeExact(0)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk").AtSliceIndex(0).AtMapKey("size"), knownvalue.StringExact("6000")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("disk")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-standard-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("swap_size"), knownvalue.StringExact("0")),
					stateCheckInstanceDisks(&instance, testDisk("disk", testDiskSize(6000))),
				},
			},
		},
	})
}

func TestAccResourceInstance_tag(t *testing.T) {
	t.Parallel()
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")
	resName := "linode_instance.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			// Start off with a single tag
			{
				Config: tmpl.Tag(t, instanceName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("tags"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("tags").AtSliceIndex(0), knownvalue.StringExact("tf_test")),
				},
			},
			// Apply updated tags
			{
				Config: tmpl.TagUpdate(t, instanceName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("tags"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("tags").AtSliceIndex(0), knownvalue.StringExact("tf_test")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("tags").AtSliceIndex(1), knownvalue.StringExact("tf_test_2")),
				},
			},
			// Reapply with different case, expect no planned changes
			{
				Config:   tmpl.TagUpdateCaseChange(t, instanceName, testRegion),
				PlanOnly: true,
			},
			// Update the tags again, expect changes
			{
				Config: tmpl.Tag(t, instanceName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("tags"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("tags").AtSliceIndex(0), knownvalue.StringExact("tf_test")),
				},
			},
		},
	})
}

func TestAccResourceInstance_tagWithVolume(t *testing.T) {
	t.Parallel()

	var instance linodego.Instance

	label := acctest.RandomWithPrefix("tf_test")

	instanceResName := "linode_instance.foobar"
	volumeResName := "linode_volume.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.TagVolume(t, label, "tf_test", testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(instanceResName, &instance),
					statecheck.ExpectKnownValue(instanceResName, tfjsonpath.New("tags"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(instanceResName, tfjsonpath.New("tags").AtSliceIndex(0), knownvalue.StringExact("tf_test")),
				},
			},
			{
				Config: tmpl.TagVolume(t, label, "tf_test_updated", testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					// Ensure the volume is not detached
					acceptance.StateCheckEventAbsent(volumeResName, "volume", linodego.ActionVolumeDetach),

					acceptance.StateCheckInstanceExists(instanceResName, &instance),
					statecheck.ExpectKnownValue(instanceResName, tfjsonpath.New("tags"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(instanceResName, tfjsonpath.New("tags").AtSliceIndex(0), knownvalue.StringExact("tf_test_updated")),
				},
			},
		},
	})
}

func TestAccResourceInstance_diskResize(t *testing.T) {
	t.Parallel()
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")
	resName := "linode_instance.foobar"
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			// Start off with a Linode 1024
			{
				Config: tmpl.DiskConfig(t, instanceName, acceptance.PublicKeyMaterial, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("specs").AtSliceIndex(0).AtMapKey("disk"), knownvalue.StringExact("25600")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("swap_size"), knownvalue.StringExact("0")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk").AtSliceIndex(0).AtMapKey("size"), knownvalue.StringExact("3000")),
					stateCheckComputeInstanceConfigs(&instance, testConfig("config", testConfigKernel("linode/latest-64bit"))),
					stateCheckInstanceDisks(&instance, testDisk("disk", testDiskSize(3000))),
				},
			},
			// Increase disk size
			{
				Config: tmpl.DiskConfigResized(t, instanceName, acceptance.PublicKeyMaterial, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("specs").AtSliceIndex(0).AtMapKey("disk"), knownvalue.StringExact("25600")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("swap_size"), knownvalue.StringExact("0")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk").AtSliceIndex(0).AtMapKey("size"), knownvalue.StringExact("6000")),
					stateCheckComputeInstanceConfigs(&instance, testConfig("config", testConfigKernel("linode/latest-64bit"))),
					stateCheckInstanceDisks(&instance, testDisk("disk", testDiskSize(6000))),
				},
			},
		},
	})
}

func TestAccResourceInstance_withDiskLinodeUpsize(t *testing.T) {
	t.Parallel()
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")
	resName := "linode_instance.foobar"
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			// Start with g6-nanode-1
			{
				Config: tmpl.DiskConfig(t, instanceName, acceptance.PublicKeyMaterial, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("specs").AtSliceIndex(0).AtMapKey("disk"), knownvalue.StringExact("25600")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("swap_size"), knownvalue.StringExact("0")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk").AtSliceIndex(0).AtMapKey("size"), knownvalue.StringExact("3000")),
					stateCheckComputeInstanceConfigs(&instance, testConfig("config", testConfigKernel("linode/latest-64bit"))),
					stateCheckInstanceDisks(&instance, testDisk("disk", testDiskSize(3000))),
				},
			},
			// Upsize to g6-standard-1 with fully allocated disk
			{
				Config: tmpl.DiskConfigExpanded(t, instanceName, acceptance.PublicKeyMaterial, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("specs").AtSliceIndex(0).AtMapKey("disk"), knownvalue.StringExact("51200")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-standard-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("swap_size"), knownvalue.StringExact("0")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk").AtSliceIndex(0).AtMapKey("size"), knownvalue.StringExact("51200")),
					stateCheckComputeInstanceConfigs(&instance, testConfig("config", testConfigKernel("linode/latest-64bit"))),
					stateCheckInstanceDisks(&instance, testDisk("disk", testDiskSize(51200))),
				},
			},
		},
	})
}

func TestAccResourceInstance_withDiskLinodeDownsize(t *testing.T) {
	t.Parallel()
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")
	resName := "linode_instance.foobar"
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			// Start with g6-standard-1 with fully allocated disk
			{
				Config: tmpl.DiskConfigExpanded(t, instanceName, acceptance.PublicKeyMaterial, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("specs").AtSliceIndex(0).AtMapKey("disk"), knownvalue.StringExact("51200")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-standard-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("swap_size"), knownvalue.StringExact("0")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk").AtSliceIndex(0).AtMapKey("size"), knownvalue.StringExact("51200")),
					stateCheckComputeInstanceConfigs(&instance, testConfig("config", testConfigKernel("linode/latest-64bit"))),
					stateCheckInstanceDisks(&instance, testDisk("disk", testDiskSize(51200))),
				},
			},
			// Downsize to g6-nanode-1
			{
				Config: tmpl.DiskConfig(t, instanceName, acceptance.PublicKeyMaterial, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("specs").AtSliceIndex(0).AtMapKey("disk"), knownvalue.StringExact("25600")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("swap_size"), knownvalue.StringExact("0")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk").AtSliceIndex(0).AtMapKey("size"), knownvalue.StringExact("3000")),
					stateCheckComputeInstanceConfigs(&instance, testConfig("config", testConfigKernel("linode/latest-64bit"))),
					stateCheckInstanceDisks(&instance, testDisk("disk", testDiskSize(3000))),
				},
			},
		},
	})
}

func TestAccResourceInstance_downsizeWithoutDisk(t *testing.T) {
	t.Parallel()

	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")
	resName := "linode_instance.foobar"
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.WithType(t, instanceName, acceptance.PublicKeyMaterial, "g6-standard-1", testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					stateCheckInstanceDisks(&instance, testDiskByFS(linodego.FilesystemExt4, testDiskSize(50944)), testDiskByFS(linodego.FilesystemSwap, testDiskSize(256)), ),
				},
			},
			{
				Config: tmpl.WithType(t, instanceName, acceptance.PublicKeyMaterial, "g6-nanode-1", testRegion, rootPass),
				ExpectError: regexp.MustCompile(
					"insufficient disk capacity"),
			},
		},
	})
}

func TestAccResourceInstance_fullDiskSwapUpsize(t *testing.T) {
	t.Parallel()

	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")
	stackScriptName := acctest.RandomWithPrefix("tf_test")
	resName := "linode_instance.foobar"
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.FullDisk(t, instanceName, acceptance.PublicKeyMaterial, stackScriptName, testRegion, 256, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					stateCheckInstanceDisks(&instance, testDiskByFS(linodego.FilesystemExt4, testDiskSize(25344)), testDiskByFS(linodego.FilesystemSwap, testDiskSize(256)), ),
				},
			},
			{
				PreConfig: func() {
					ctx := context.Background()
					client := acceptance.GetSSHClient(t, "root", instance.IPv4[0].String())

					defer client.Close()
					ss, err := client.NewSession()
					if err != nil {
						t.Fatalf("failed to establish SSH session: %s", err)
					}

					ctx, cancel := context.WithTimeout(ctx, time.Minute)
					defer cancel()

					ticker := time.NewTicker(500 * time.Millisecond)
					defer ticker.Stop()

					for {
						select {
						case <-ticker.C:
							buf := new(bytes.Buffer)
							ss.Stdout = buf
							ss.Run("[[ $(df /dev/sda --block-size=1 | tail -n-1 | awk '{print $5}') == '100%' ]] && echo 1 || echo 0")

							if buf.String() == "1" {
								return
							}

						case <-ctx.Done():
							return
						}
					}
				},
				Config:      tmpl.FullDisk(t, instanceName, acceptance.PublicKeyMaterial, stackScriptName, testRegion, 512, rootPass),
				ExpectError: regexp.MustCompile("Error waiting for resize of Instance \\d+ Disk \\d+"),
			},
		},
	})
}

func TestAccResourceInstance_swapUpsize(t *testing.T) {
	t.Parallel()

	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")
	resName := "linode_instance.foobar"
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.WithSwapSize(t, instanceName, acceptance.PublicKeyMaterial, testRegion, 256, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					stateCheckInstanceDisks(&instance, testDiskByFS(linodego.FilesystemExt4, testDiskSize(25344)), testDiskByFS(linodego.FilesystemSwap, testDiskSize(256)), ),
				},
			},
			{
				Config: tmpl.WithSwapSize(t, instanceName, acceptance.PublicKeyMaterial, testRegion, 512, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					stateCheckInstanceDisks(&instance, testDiskByFS(linodego.FilesystemExt4, testDiskSize(25088)), testDiskByFS(linodego.FilesystemSwap, testDiskSize(512)), ),
				},
			},
		},
	})
}

func TestAccResourceInstance_swapDownsize(t *testing.T) {
	t.Parallel()

	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")
	resName := "linode_instance.foobar"

	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.WithSwapSize(t, instanceName, acceptance.PublicKeyMaterial, testRegion, 512, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					stateCheckInstanceDisks(&instance, testDiskByFS(linodego.FilesystemExt4, testDiskSize(25088)), testDiskByFS(linodego.FilesystemSwap, testDiskSize(512)), ),
				},
			},
			{
				Config: tmpl.WithSwapSize(t, instanceName, acceptance.PublicKeyMaterial, testRegion, 256, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					stateCheckInstanceDisks(&instance, testDiskByFS(linodego.FilesystemExt4, testDiskSize(25344)), testDiskByFS(linodego.FilesystemSwap, testDiskSize(256)), ),
				},
			},
		},
	})
}

func TestAccResourceInstance_diskResizeAndExpanded(t *testing.T) {
	t.Parallel()
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")
	resName := "linode_instance.foobar"
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			// Start off with a Linode 1024
			{
				Config: tmpl.DiskConfig(t, instanceName, acceptance.PublicKeyMaterial, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("specs").AtSliceIndex(0).AtMapKey("disk"), knownvalue.StringExact("25600")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("swap_size"), knownvalue.StringExact("0")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk").AtSliceIndex(0).AtMapKey("size"), knownvalue.StringExact("3000")),
					stateCheckComputeInstanceConfigs(&instance, testConfig("config", testConfigKernel("linode/latest-64bit"))),
					stateCheckInstanceDisks(&instance, testDisk("disk", testDiskSize(3000))),
				},
			},

			// Bump to 2048 and expand disk
			{
				Config: tmpl.DiskConfigResizedExpanded(t, instanceName, acceptance.PublicKeyMaterial, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("specs").AtSliceIndex(0).AtMapKey("disk"), knownvalue.StringExact("51200")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-standard-1")),

					statecheck.ExpectKnownValue(resName, tfjsonpath.New("swap_size"), knownvalue.StringExact("0")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk").AtSliceIndex(0).AtMapKey("size"), knownvalue.StringExact("6000")),

					stateCheckComputeInstanceConfigs(&instance, testConfig("config", testConfigKernel("linode/latest-64bit"))),
					stateCheckInstanceDisks(&instance, testDisk("disk", testDiskSize(6000))),
				},
			},
		},
	})
}

func TestAccResourceInstance_diskSlotReorder(t *testing.T) {
	t.Parallel()
	var (
		instance     linodego.Instance
		instanceDisk linodego.InstanceDisk
	)
	instanceName := acctest.RandomWithPrefix("tf_test")
	resName := "linode_instance.foobar"
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			// Start off with a Linode 1024
			{
				Config: tmpl.DiskConfig(t, instanceName, acceptance.PublicKeyMaterial, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("specs").AtSliceIndex(0).AtMapKey("disk"), knownvalue.StringExact("25600")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					stateCheckInstanceDisks(&instance, testDisk("disk", testDiskExists(&instanceDisk), testDiskSize(3000))),
					stateCheckComputeInstanceConfigs(&instance, testConfig("config", testConfigKernel("linode/latest-64bit"), testConfigSDADisk(&instanceDisk))),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("devices").AtSliceIndex(0).AtMapKey("sda").AtSliceIndex(0).AtMapKey("disk_id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("devices").AtSliceIndex(0).AtMapKey("sdb"), knownvalue.ListSizeExact(0)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("swap_size"), knownvalue.StringExact("0")),
					stateCheckComputeInstanceConfigs(&instance, testConfig("config", testConfigKernel("linode/latest-64bit"))),
				},
			},
			// Add a disk, reorder the disks
			{
				Config: tmpl.DiskConfigReordered(t, instanceName, acceptance.PublicKeyMaterial, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("specs").AtSliceIndex(0).AtMapKey("disk"), knownvalue.StringExact("51200")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-standard-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk").AtSliceIndex(0).AtMapKey("size"), knownvalue.StringExact("3000")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("disk")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk").AtSliceIndex(1).AtMapKey("size"), knownvalue.StringExact("3000")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk").AtSliceIndex(1).AtMapKey("label"), knownvalue.StringExact("diskb")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk").AtSliceIndex(1).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("config")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("kernel"), knownvalue.StringExact("linode/latest-64bit")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("devices").AtSliceIndex(0).AtMapKey("sda").AtSliceIndex(0).AtMapKey("disk_id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("devices").AtSliceIndex(0).AtMapKey("sdb").AtSliceIndex(0).AtMapKey("disk_id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("devices").AtSliceIndex(0).AtMapKey("sdc"), knownvalue.ListSizeExact(0)),
					statecheck.CompareValuePairs(
						resName,
						tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("devices").AtSliceIndex(0).AtMapKey("sda").AtSliceIndex(0).AtMapKey("disk_id"),
						resName,
						tfjsonpath.New("disk").AtSliceIndex(1).AtMapKey("id"),
						compare.ValuesSame(),
					),
					statecheck.CompareValuePairs(
						resName,
						tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("devices").AtSliceIndex(0).AtMapKey("sdb").AtSliceIndex(0).AtMapKey("disk_id"),
						resName,
						tfjsonpath.New("disk").AtSliceIndex(0).AtMapKey("id"),
						compare.ValuesSame(),
					),

					statecheck.ExpectKnownValue(resName, tfjsonpath.New("swap_size"), knownvalue.StringExact("0")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("status"), knownvalue.StringExact("running")),
				},
			},
		},
	})
}

func TestAccResourceInstance_privateNetworking(t *testing.T) {
	t.Parallel()
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")
	resName := "linode_instance.foobar"
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.PrivateNetworking(t, instanceName, acceptance.PublicKeyMaterial, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					stateCheckInstancePrivateNetworkAttributes("linode_instance.foobar"),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("private_ip"), knownvalue.StringExact("true")),
				},
			},
		},
	})
}

func TestAccResourceInstance_stackScriptInstance(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.StackScript(t, instanceName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("image"), knownvalue.StringExact(acceptance.TestImageLatest)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("group"), knownvalue.StringExact("tf_test")),
				},
			},

			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       false,
				ImportStateVerifyIgnore: []string{"root_pass", "authorized_keys", "image", "resize_disk", "migration_type", "firewall_id"},
			},
		},
	})
}

func TestAccResourceInstance_diskImageUpdate(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.DiskBootImage(t, instanceName, acceptance.TestImagePrevious, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
				},
			},
			{
				Config: tmpl.DiskBootImage(t, instanceName, acceptance.TestImageLatest, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					// resource was tainted for recreation due to change of disk.0.image, marked
					// with ForceNew.
					acceptance.StateCheckResourceAttrNotEqual(resName, "id", strconv.Itoa(instance.ID)),
				},
			},

			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       false,
				ImportStateVerifyIgnore: []string{"root_pass", "authorized_keys", "image", "resize_disk", "migration_type", "firewall_id"},
			},
		},
	})
}

func TestAccResourceInstance_stackScriptDisk(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.DiskStackScript(t, instanceName, acceptance.PublicKeyMaterial, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
				},
			},
		},
	})
}

func TestAccResourceInstance_typeChangeDiskImplicit(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"

	var instance linodego.Instance
	// oldDiskSize := 0

	instanceName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			// Create an initial instance
			{
				Config: tmpl.TypeChangeDisk(t, instanceName, "g6-nanode-1", testRegion, true),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
				},
			},
			// Upsize the instance and disk
			{
				Config: tmpl.TypeChangeDisk(t, instanceName, "g6-standard-1", testRegion, true),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-standard-1")),
				},
			},
			// Attempt a downsize
			{
				Config:      tmpl.TypeChangeDisk(t, instanceName, "g6-nanode-1", testRegion, true),
				ExpectError: regexp.MustCompile("Did you try to resize a linode with implicit"),
			},
		},
	})
}

func TestAccResourceInstance_typeChangeDiskExplicit(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			// Create an instance with explicit disks
			{
				Config: tmpl.TypeChangeDiskExplicit(t, instanceName, "g6-nanode-1", testRegion, false),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
				},
			},
			// Attempt to resize the instance and disk and expect an error
			{
				Config:      tmpl.TypeChangeDiskExplicit(t, instanceName, "g6-standard-1", testRegion, true),
				ExpectError: regexp.MustCompile("all of `image,resize_disk` must be specified"),
			},
			// Resize only the instance
			{
				Config: tmpl.TypeChangeDiskExplicit(t, instanceName, "g6-standard-1", testRegion, false),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-standard-1")),
				},
			},
		},
	})
}

func TestAccResourceInstance_typeChangeNoDisks(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			// Create an instance with explicit disks
			{
				Config: tmpl.TypeChangeDiskNone(t, instanceName, "g6-nanode-1", testRegion, false),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
				},
			},
			// Attempt to resize the instance
			{
				Config: tmpl.TypeChangeDiskNone(t, instanceName, "g6-standard-1", testRegion, false),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-standard-1")),
				},
			},
			// Attempt to downsize the instance
			{
				Config: tmpl.TypeChangeDiskNone(t, instanceName, "g6-nanode-1", testRegion, false),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
				},
			},
		},
	})
}

func TestAccResourceInstance_powerStateUpdates(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.BootState(t, instanceName, testRegion, false),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("status"), knownvalue.StringExact("offline")),
				},
			},
			{
				Config: tmpl.BootState(t, instanceName, testRegion, true),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("status"), knownvalue.StringExact("running")),
				},
			},
			{
				Config: tmpl.BootState(t, instanceName, testRegion, false),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("status"), knownvalue.StringExact("offline")),
				},
			},
			// Ensure an implicit reboot isn't triggered when booted == false
			{
				Config: tmpl.BootStateInterface(t, instanceName, testRegion, false),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("status"), knownvalue.StringExact("offline")),
				},
			},
			{
				PreConfig: func() {
					client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

					filter := &linodego.Filter{}
					filter.AddField(linodego.Eq, "action", linodego.ActionLinodeReboot)
					filter.AddField(linodego.Eq, "entity.id", instance.ID)
					filter.AddField(linodego.Eq, "entity.type", linodego.EntityLinode)
					jsonData, err := filter.MarshalJSON()
					if err != nil {
						t.Fatal(err)
					}

					events, err := client.ListEvents(context.Background(), &linodego.ListOptions{Filter: string(jsonData)})
					if err != nil {
						t.Fatal(err)
					}

					if len(events) > 0 {
						t.Fatal("found reboot event when no reboot was expected")
					}
				},
				Config: tmpl.BootStateInterface(t, instanceName, testRegion, false),
			},
		},
	})
}

func TestAccResourceInstance_powerStateConfigUpdates(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.BootStateConfig(t, instanceName, testRegion, false, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("status"), knownvalue.StringExact("offline")),
				},
			},
			{
				Config: tmpl.BootStateConfig(t, instanceName, testRegion, true, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("status"), knownvalue.StringExact("running")),
				},
			},
			{
				Config: tmpl.BootStateConfig(t, instanceName, testRegion, false, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("status"), knownvalue.StringExact("offline")),
				},
			},
		},
	})
}

func TestAccResourceInstance_powerStateConfigBooted(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.BootStateConfig(t, instanceName, testRegion, true, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("status"), knownvalue.StringExact("running")),
				},
			},
		},
	})
}

func TestAccResourceInstance_powerStateBooted(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.BootState(t, instanceName, testRegion, true),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("status"), knownvalue.StringExact("running")),
				},
			},
		},
	})
}

func TestAccResourceInstance_powerStateNoImage(t *testing.T) {
	t.Parallel()

	instanceName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config:      tmpl.BootStateNoImage(t, instanceName, testRegion, true),
				ExpectError: regexp.MustCompile("booted requires an image or disk/config be defined"),
			},
		},
	})
}

func TestAccResourceInstance_ipv4Sharing(t *testing.T) {
	t.Parallel()

	// We need to manually override the region as IP sharing capabilities aren't
	// explicitly mentioned by the API.
	const region = "us-west"

	failoverResName := "linode_instance.failover"

	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config:      tmpl.IPv4SharingBadInput(t, instanceName, region),
				ExpectError: regexp.MustCompile("expected ipv4 address, got"),
			},
			{
				Config: tmpl.IPv4Sharing(t, instanceName, region),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(failoverResName, &instance),
					statecheck.ExpectKnownValue(failoverResName, tfjsonpath.New("shared_ipv4"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(failoverResName, tfjsonpath.New("shared_ipv4").AtSliceIndex(0), knownvalue.NotNull()),
				},
			},
			{
				Config: tmpl.IPv4SharingAllocation(t, instanceName, region),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(failoverResName, &instance),
					statecheck.ExpectKnownValue(failoverResName, tfjsonpath.New("shared_ipv4"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(failoverResName, tfjsonpath.New("shared_ipv4").AtSliceIndex(0), knownvalue.NotNull()),
				},
			},
			{
				Config: tmpl.IPv4SharingEmpty(t, instanceName, region),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(failoverResName, &instance),
					statecheck.ExpectKnownValue(failoverResName, tfjsonpath.New("shared_ipv4"), knownvalue.ListSizeExact(0)),
				},
			},
		},
	})
}

func TestAccResourceInstance_userData(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")

	region, err := acceptance.GetRandomRegionWithCaps([]string{"Metadata"}, "core")
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
				Config: tmpl.UserData(t, instanceName, region, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("image"), knownvalue.StringExact(acceptance.TestImageLatest)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(region)),

					statecheck.ExpectKnownValue(resName, tfjsonpath.New("has_user_data"), knownvalue.StringExact("true")),
				},
			},
			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"root_pass", "authorized_keys", "image", "resize_disk", "metadata", "migration_type", "firewall_id"},
			},
		},
	})
}

func TestAccResourceInstance_requestQuantity(t *testing.T) {
	t.Skip("firewall no longer available in old test provider")
	t.Parallel()

	const maxRequestsPerSecond = 3.0

	// We need to make sure we're not running into a race condition here
	var numRequestsLock sync.Mutex
	numRequests := 0
	var startTime time.Time

	instanceName := acctest.RandomWithPrefix("tf_test")

	provider, providerMap := acceptance.CreateTestProvider()

	rootPass := acctest.RandString(64)

	acceptance.ModifyProviderMeta(provider,
		func(ctx context.Context, _ *schema.ResourceData, config *helper.ProviderMeta) error {
			config.Client.OnBeforeRequest(func(request *linodego.Request) error {
				if startTime.IsZero() {
					startTime = time.Now()
				}

				numRequestsLock.Lock()
				defer numRequestsLock.Unlock()
				numRequests++

				return nil
			})

			return nil
		})

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { acceptance.PreCheck(t) },
		Providers:         providerMap,
		ExternalProviders: acceptance.HttpExternalProviders,
		Steps: []resource.TestStep{
			{
				// Provision a bunch of Linodes and wait for them to boot into an image
				Config: tmpl.ManyLinodes(t, instanceName, acceptance.PublicKeyMaterial, testRegion, rootPass),
			},
			{
				PreConfig: func() {
					requestsPerSecond := (float64(numRequests) / float64(time.Since(startTime).Seconds()))

					t.Logf("\n[INFO] results from 12 linode parallel creation:\n"+
						"total requests: %d\nfrequency: ~%f requests/second\n", numRequests, requestsPerSecond)

					if requestsPerSecond > maxRequestsPerSecond {
						t.Fatalf("too many requests: %f > %f", requestsPerSecond, maxRequestsPerSecond)
					}
				},
				Config: tmpl.ManyLinodes(t, instanceName, acceptance.PublicKeyMaterial, testRegion, rootPass),
			},
		},
	})
}

func TestAccResourceInstance_firewallOnCreation(t *testing.T) {
	t.Parallel()

	instanceResourceName := "linode_instance.foobar"
	firewallResourceName := "linode_firewall.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")

	region, err := acceptance.GetRandomRegionWithCaps([]string{"Cloud Firewall"}, "core")
	rootPass := acctest.RandString(64)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.FirewallOnCreation(t, instanceName, region, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(instanceResourceName, &instance),
				},
			},
			{
				RefreshState: true,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.CompareValuePairs(
						firewallResourceName,
						tfjsonpath.New("devices").AtSliceIndex(0).AtMapKey("label"),
						instanceResourceName,
						tfjsonpath.New("label"),
						compare.ValuesSame(),
					),
				},
			},
			{
				ResourceName:            instanceResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"root_pass", "authorized_keys", "image", "resize_disk", "firewall_id", "migration_type"},
			},
		},
	})
}

func TestAccResourceInstance_VPCInterface(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf-test")

	acceptance.RunTestWithRetries(t, 3, func(t *acceptance.WrappedT) {
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             acceptance.CheckInstanceDestroy,

			Steps: []resource.TestStep{
				{
					Config: tmpl.VPCInterface(t, instanceName, testRegion),
					ConfigStateChecks: []statecheck.StateCheck{
						acceptance.StateCheckInstanceExists(resName, &instance),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("image"), knownvalue.StringExact(acceptance.TestImageLatest)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface").AtSliceIndex(0).AtMapKey("purpose"), knownvalue.StringExact("vpc")),

						statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface").AtSliceIndex(0).AtMapKey("ipv4").AtSliceIndex(0).AtMapKey("vpc"), knownvalue.StringExact("10.0.4.150")),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface").AtSliceIndex(0).AtMapKey("ip_ranges").AtSliceIndex(0), knownvalue.StringExact("10.0.4.100/32")),
						statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface").AtSliceIndex(0).AtMapKey("ipv4").AtSliceIndex(0).AtMapKey("nat_1_1"), knownvalue.NotNull()),
					},
				},
				{
					ResourceName:      resName,
					ImportState:       true,
					ImportStateVerify: true,
					ImportStateVerifyIgnore: []string{
						"image",
						"interface",
						"resize_disk",
						"migration_type",
						"firewall_id",
						"capabilities.#",
						"capabilities.0",
						"capabilities.1",
					},
				},
			},
		})
	})
}

func TestAccResourceInstance_VPCPublicInterfacesAddRemoveSwap(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.PublicInterface(t, instanceName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("image"), knownvalue.StringExact(acceptance.TestImageLatest)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface").AtSliceIndex(0).AtMapKey("purpose"), knownvalue.StringExact("public")),
				},
			},
			{
				Config: tmpl.PublicAndVPCInterfaces(t, instanceName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("image"), knownvalue.StringExact(acceptance.TestImageLatest)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface").AtSliceIndex(0).AtMapKey("purpose"), knownvalue.StringExact("public")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface").AtSliceIndex(1).AtMapKey("purpose"), knownvalue.StringExact("vpc")),
				},
			},
			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"image", "interface", "resize_disk", "migration_type", "firewall_id"},
			},
			{
				Config: tmpl.VPCAndPublicInterfaces(t, instanceName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("image"), knownvalue.StringExact(acceptance.TestImageLatest)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface").AtSliceIndex(0).AtMapKey("purpose"), knownvalue.StringExact("vpc")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface").AtSliceIndex(1).AtMapKey("purpose"), knownvalue.StringExact("public")),
				},
			},
			{
				Config: tmpl.PublicInterface(t, instanceName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("image"), knownvalue.StringExact(acceptance.TestImageLatest)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("config").AtSliceIndex(0).AtMapKey("interface").AtSliceIndex(0).AtMapKey("purpose"), knownvalue.StringExact("public")),
				},
			},
			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"image", "interface", "resize_disk", "migration_type", "firewall_id"},
			},
		},
	})
}

func TestAccResourceInstance_migration(t *testing.T) {
	acceptance.LongRunningTest(t)

	t.Parallel()

	rootPass := acctest.RandString(64)

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")

	// Resolve a region to migrate to
	targetRegion, err := acceptance.GetRandomRegionWithCaps(
		[]string{"Linodes"}, "core",
		func(v linodego.Region) bool {
			return v.ID != testRegion
		},
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
				Config: tmpl.Basic(t, instanceName, acceptance.PublicKeyMaterial, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("image"), knownvalue.StringExact(acceptance.TestImageLatest)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
				},
			},
			{
				Config: tmpl.Basic(t, instanceName, acceptance.PublicKeyMaterial, targetRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("image"), knownvalue.StringExact(acceptance.TestImageLatest)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(targetRegion)),
				},
			},
			// TODO: Add logic for testing warm migrations once possible
			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"root_pass", "authorized_keys", "image", "resize_disk", "metadata", "migration_type", "firewall_id"},
			},
		},
	})
}

func TestAccResourceInstance_withPG(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	testLabel := acctest.RandomWithPrefix("tf_test")

	pgIDs := []string{"g1"}

	// Resolve a region with support for PGs
	targetRegion, err := acceptance.GetRandomRegionWithCaps(
		[]string{"Linodes", "Placement Group"}, "core",
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
				Config: tmpl.WithPG(t, testLabel, targetRegion, "g1", pgIDs),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(testLabel)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(targetRegion)),

					statecheck.ExpectKnownValue(resName, tfjsonpath.New("placement_group"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("placement_group").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("placement_group").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact(testLabel+"-g1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("placement_group").AtSliceIndex(0).AtMapKey("placement_group_type"), knownvalue.StringExact("anti_affinity:local")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("placement_group").AtSliceIndex(0).AtMapKey("placement_group_policy"), knownvalue.StringExact("flexible")),
				},
			},
			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"root_pass", "authorized_keys", "image", "resize_disk", "metadata", "migration_type"},
			},
		},
	})
}

func TestAccResourceInstance_pgAssignment(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	testLabel := acctest.RandomWithPrefix("tf_test")

	pgIDs := []string{"g1", "g2"}

	// Resolve a region with support for PGs
	testRegion, err := acceptance.GetRandomRegionWithCaps(
		[]string{"Linodes", "Placement Group"}, "core",
	)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,
		Steps: []resource.TestStep{
			// Create the instance with a PG
			{
				Config: tmpl.WithPG(t, testLabel, testRegion, "", pgIDs),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(testLabel)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),

					statecheck.ExpectKnownValue(resName, tfjsonpath.New("placement_group"), knownvalue.ListSizeExact(0)),
				},
			},

			// Assign the instance to a PG
			{
				Config: tmpl.WithPG(t, testLabel, testRegion, "g1", pgIDs),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(testLabel)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),

					statecheck.ExpectKnownValue(resName, tfjsonpath.New("placement_group"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("placement_group").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("placement_group").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact(testLabel+"-g1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("placement_group").AtSliceIndex(0).AtMapKey("placement_group_type"), knownvalue.StringExact("anti_affinity:local")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("placement_group").AtSliceIndex(0).AtMapKey("placement_group_policy"), knownvalue.StringExact("flexible")),
				},
			},

			// Reassign the instance to another PG
			{
				Config: tmpl.WithPG(t, testLabel, testRegion, "g2", pgIDs),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(testLabel)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),

					statecheck.ExpectKnownValue(resName, tfjsonpath.New("placement_group"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("placement_group").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("placement_group").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact(testLabel+"-g2")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("placement_group").AtSliceIndex(0).AtMapKey("placement_group_type"), knownvalue.StringExact("anti_affinity:local")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("placement_group").AtSliceIndex(0).AtMapKey("placement_group_policy"), knownvalue.StringExact("flexible")),
				},
			},

			// Unassign the instance from the PG
			{
				Config: tmpl.WithPG(t, testLabel, testRegion, "", pgIDs),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(testLabel)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),

					statecheck.ExpectKnownValue(resName, tfjsonpath.New("placement_group"), knownvalue.ListSizeExact(0)),
				},
			},
			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"root_pass", "authorized_keys", "image", "resize_disk", "metadata", "migration_type"},
			},
		},
	})
}

func TestAccResourceInstance_diskEncryption(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")
	rootPass := acctest.RandString(64)

	// Resolve a region that supports disk encryption
	targetRegion, err := acceptance.GetRandomRegionWithCaps(
		[]string{linodego.CapabilityLinodes, linodego.CapabilityDiskEncryption}, "core",
	)
	if err != nil {
		t.Fatal(err)
	}

	encryptionEnabled := linodego.InstanceDiskEncryptionEnabled
	encryptionDisabled := linodego.InstanceDiskEncryptionDisabled

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DiskEncryption(
					t,
					instanceName,
					targetRegion,
					rootPass,
					&encryptionEnabled,
				),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("image"), knownvalue.StringExact(acceptance.TestImageLatest)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(targetRegion)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk_encryption"), knownvalue.StringExact("enabled")),
				},
			},
			{
				Config: tmpl.DiskEncryption(
					t,
					instanceName,
					targetRegion,
					rootPass,
					&encryptionDisabled,
				),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("image"), knownvalue.StringExact(acceptance.TestImageLatest)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(targetRegion)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk_encryption"), knownvalue.StringExact("disabled")),
				},
			},

			// Make sure the instance is not recreated when disk_encryption is not explicitly set.
			// This is necessary to prevent instances created pre-disk-encryption from being recreated.
			{
				Config: tmpl.DiskEncryption(
					t,
					instanceName,
					targetRegion,
					rootPass,
					nil,
				),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("type"), knownvalue.StringExact("g6-nanode-1")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("image"), knownvalue.StringExact(acceptance.TestImageLatest)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("region"), knownvalue.StringExact(targetRegion)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("disk_encryption"), knownvalue.StringExact("disabled")),
				},
			},

			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"root_pass", "authorized_keys", "image", "resize_disk", "migration_type"},
			},
		},
	})
}

func TestAccResourceInstance_interfaceGenerationLegacy(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	instanceName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.ExplicitInterfaceGeneration(
					t,
					instanceName,
					testRegion,
					true,
					linodego.GenerationLegacyConfig,
					nil,
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("label"),
						knownvalue.StringExact(instanceName),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("type"),
						knownvalue.StringExact("g6-nanode-1"),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("image"),
						knownvalue.StringExact(acceptance.TestImageLatest),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("region"),
						knownvalue.StringExact(testRegion),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("interface_generation"),
						knownvalue.StringExact(string(linodego.GenerationLegacyConfig)),
					),
				},
			},

			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"root_pass", "authorized_keys", "image", "resize_disk", "migration_type", "firewall_id"},
			},
		},
	})
}

func TestAccResourceInstance_interfaceGenerationLinode(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	instanceName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.ExplicitInterfaceGeneration(
					t,
					instanceName,
					testRegion,
					true,
					linodego.GenerationLinode,
					linodego.Pointer(true),
				),
				ExpectError: regexp.MustCompile(
					"The Linode must have at least 1 interface defined to boot",
				),
			},
			{
				Config: tmpl.ExplicitInterfaceGeneration(
					t,
					instanceName,
					testRegion,
					false,
					linodego.GenerationLinode,
					linodego.Pointer(true),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("label"),
						knownvalue.StringExact(instanceName),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("type"),
						knownvalue.StringExact("g6-nanode-1"),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("image"),
						knownvalue.StringExact(acceptance.TestImageLatest),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("region"),
						knownvalue.StringExact(testRegion),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("interface_generation"),
						knownvalue.StringExact(string(linodego.GenerationLinode)),
					),
				},
			},
			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"root_pass", "authorized_keys", "image", "resize_disk", "migration_type", "firewall_id", "network_helper"},
			},
		},
	})
}

func TestAccResourceInstance_interfaceVPCIPv6(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf-test")
	rootPass := acctest.RandString(64)

	// TODO (VPC Dual Stack): Remove region hardcoding
	targetRegion := "no-osl-1"

	ipv6Path := tfjsonpath.New("interface").
		AtSliceIndex(0).
		AtMapKey("ipv6").
		AtSliceIndex(0)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.InterfacesVPCIPv60(
					t,
					instanceName,
					targetRegion,
					rootPass,
				),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.AtMapKey("is_public"),
						knownvalue.Bool(false),
					),

					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.AtMapKey("slaac"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.
							AtMapKey("slaac").
							AtSliceIndex(0).
							AtMapKey("range"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.
							AtMapKey("slaac").
							AtSliceIndex(0).
							AtMapKey("assigned_range"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.
							AtMapKey("slaac").
							AtSliceIndex(0).
							AtMapKey("address"),
						knownvalue.NotNull(),
					),

					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.
							AtMapKey("range"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.
							AtMapKey("range").
							AtSliceIndex(0).
							AtMapKey("assigned_range"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.
							AtMapKey("range").
							AtSliceIndex(0).
							AtMapKey("range"),
						knownvalue.NotNull(),
					),
				},
			},
			{
				Config: tmpl.InterfacesVPCIPv61(
					t,
					instanceName,
					targetRegion,
					rootPass,
				),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					// interface[0].ipv6[0] public flag
					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.
							AtMapKey("is_public"),
						knownvalue.Bool(true),
					),

					// slaac block (exactly one) with fields
					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.AtMapKey("slaac"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.
							AtMapKey("slaac").
							AtSliceIndex(0).
							AtMapKey("range"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.
							AtMapKey("slaac").
							AtSliceIndex(0).
							AtMapKey("assigned_range"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.
							AtMapKey("slaac").
							AtSliceIndex(0).
							AtMapKey("address"),
						knownvalue.NotNull(),
					),

					// range list (exactly two) with fields
					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.
							AtMapKey("range"),
						knownvalue.ListSizeExact(2),
					),
					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.
							AtMapKey("range").
							AtSliceIndex(0).
							AtMapKey("assigned_range"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.
							AtMapKey("range").
							AtSliceIndex(0).
							AtMapKey("range"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.
							AtMapKey("range").
							AtSliceIndex(1).
							AtMapKey("assigned_range"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.
							AtMapKey("range").
							AtSliceIndex(1).
							AtMapKey("range"),
						knownvalue.NotNull(),
					),
				},
			},
			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"root_pass", "authorized_keys", "image", "resize_disk", "migration_type", "interface", "firewall_id"},
			},
		},
	})
}

func TestAccResourceInstance_configInterfaceVPCIPv6(t *testing.T) {
	t.Parallel()

	resName := "linode_instance.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf-test")
	rootPass := acctest.RandString(64)

	// TODO (VPC Dual Stack): Remove region hardcoding
	targetRegion := "no-osl-1"

	ipv6Path := tfjsonpath.New("config").
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
				Config: tmpl.ConfigInterfacesVPCIPv6(
					t,
					instanceName,
					targetRegion,
					rootPass,
				),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.AtMapKey("slaac"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.AtMapKey("slaac").AtSliceIndex(0).AtMapKey("range"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.AtMapKey("slaac").AtSliceIndex(0).AtMapKey("assigned_range"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.AtMapKey("slaac").AtSliceIndex(0).AtMapKey("address"),
						knownvalue.NotNull(),
					),

					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.AtMapKey("range"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.AtMapKey("range").AtSliceIndex(0).AtMapKey("assigned_range"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.AtMapKey("range").AtSliceIndex(0).AtMapKey("range"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						// There seems to be a bug that causes reusing a path
						// to break bool checks under certain conditions.
						tfjsonpath.New("config").
							AtSliceIndex(0).
							AtMapKey("interface").
							AtSliceIndex(0).
							AtMapKey("ipv6").
							AtSliceIndex(0).
							AtMapKey("is_public"),
						knownvalue.Bool(true),
					),
				},
			},
			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"backups",
					"root_pass",
					"authorized_keys",
					"image",
					"resize_disk",
					"migration_type",
					"interface",
					"firewall_id",
				},
			},
		},
	})
}

func checkInstancePrivateNetworkAttributes(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("should have found linode_instance resource %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("should have a Linode ID")
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("should have an integer Linode ID: %s", err)
		}

		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		instanceIPs, err := client.GetInstanceIPAddresses(context.Background(), id)
		if err != nil {
			return err
		}
		if len(instanceIPs.IPv4.Private) == 0 {
			return fmt.Errorf("should have a private ip on Linode ID %d", id)
		}
		return nil
	}
}

type (
	testDiskFunc  func(disk linodego.InstanceDisk) error
	testDisksFunc func(disk []linodego.InstanceDisk) error
)

func testDisk(label string, diskTests ...testDiskFunc) testDisksFunc {
	return func(disks []linodego.InstanceDisk) error {
		for _, disk := range disks {
			if disk.Label == label {
				for _, test := range diskTests {
					if err := test(disk); err != nil {
						return err
					}
				}
				return nil
			}
		}
		return fmt.Errorf("should have found Instance disk with label: %s", label)
	}
}

func testDiskByFS(fs linodego.DiskFilesystem, diskTests ...testDiskFunc) testDisksFunc {
	return func(disks []linodego.InstanceDisk) error {
		for _, disk := range disks {
			if disk.Filesystem == fs {
				for _, test := range diskTests {
					if err := test(disk); err != nil {
						return err
					}
				}
				return nil
			}
		}
		return fmt.Errorf("should have found Instance disk with filesystem: %s", fs)
	}
}

func testDiskExists(diskPtr *linodego.InstanceDisk) testDiskFunc {
	return func(disk linodego.InstanceDisk) error {
		*diskPtr = disk
		return nil
	}
}

func testDiskSize(size int) testDiskFunc {
	return func(disk linodego.InstanceDisk) error {
		if disk.Size != size {
			return fmt.Errorf("should have matching sizes: %d != %d", disk.Size, size)
		}
		return nil
	}
}

func checkInstanceDisks(instance *linodego.Instance, disksTests ...testDisksFunc) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		if instance == nil || instance.ID == 0 {
			return fmt.Errorf("Error fetching disks: invalid Instance argument")
		}

		instanceDisks, err := client.ListInstanceDisks(context.Background(), instance.ID, nil)
		if err != nil {
			return fmt.Errorf("Error fetching disks: %s", err)
		}

		if len(instanceDisks) == 0 {
			return fmt.Errorf("No disks")
		}

		for _, tests := range disksTests {
			if err := tests(instanceDisks); err != nil {
				return err
			}
		}

		return nil
	}
}

type (
	testConfigFunc  func(config linodego.InstanceConfig) error
	testConfigsFunc func(config []linodego.InstanceConfig) error
)

// testConfig verifies a labeled config exists and runs many tests against that config
func testConfig(label string, configTests ...testConfigFunc) testConfigsFunc {
	return func(configs []linodego.InstanceConfig) error {
		for _, config := range configs {
			if config.Label == label {
				for _, test := range configTests {
					if err := test(config); err != nil {
						return err
					}
				}
				return nil
			}
		}
		return fmt.Errorf("should have found Instance config with label: %s", label)
	}
}

func testConfigExists(configPtr *linodego.InstanceConfig) testConfigFunc {
	return func(config linodego.InstanceConfig) error {
		*configPtr = config
		return nil
	}
}

func testConfigLabel(label string) testConfigFunc {
	return func(config linodego.InstanceConfig) error {
		if config.Label != label {
			return fmt.Errorf("should have matching labels: %s != %s", config.Label, label)
		}
		return nil
	}
}

func testConfigKernel(kernel string) testConfigFunc {
	return func(config linodego.InstanceConfig) error {
		if config.Kernel != kernel {
			return fmt.Errorf("should have matching kernels: %s != %s", config.Kernel, kernel)
		}
		return nil
	}
}

func testConfigComments(comments string) testConfigFunc {
	return func(config linodego.InstanceConfig) error {
		if config.Comments != comments {
			return fmt.Errorf("should have matching comments: %s != %s", config.Comments, comments)
		}
		return nil
	}
}

func testConfigSDADisk(disk *linodego.InstanceDisk) testConfigFunc {
	return func(config linodego.InstanceConfig) error {
		if disk == nil || config.Devices == nil || config.Devices.SDA == nil || config.Devices.SDA.DiskID != disk.ID {
			return fmt.Errorf("should have SDA with expected disk id")
		}
		return nil
	}
}

func testConfigSDBDisk(disk *linodego.InstanceDisk) testConfigFunc {
	return func(config linodego.InstanceConfig) error {
		if disk == nil || config.Devices == nil || config.Devices.SDB == nil || config.Devices.SDB.DiskID != disk.ID {
			return fmt.Errorf("should have SDB with expected disk id")
		}
		return nil
	}
}

func testConfigSDBVolume(volume *linodego.Volume) testConfigFunc {
	return func(config linodego.InstanceConfig) error {
		if volume == nil || config.Devices == nil || config.Devices.SDB == nil || config.Devices.SDB.VolumeID != volume.ID {
			return fmt.Errorf("should have SDB with expected volume id")
		}
		return nil
	}
}

func instanceDiskID(disk *linodego.InstanceDisk) string {
	return strconv.Itoa(disk.ID)
}

// checkComputeInstanceConfigs verifies any configs exist and runs config specific tests against a target instance
func checkComputeInstanceConfigs(instance *linodego.Instance, configsTests ...testConfigsFunc) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		if instance == nil || instance.ID == 0 {
			return fmt.Errorf("Error fetching configs: invalid Instance argument")
		}

		instanceConfigs, err := client.ListInstanceConfigs(context.Background(), instance.ID, nil)
		if err != nil {
			return fmt.Errorf("Error fetching configs: %s", err)
		}

		if len(instanceConfigs) == 0 {
			return fmt.Errorf("No configs")
		}

		for _, tests := range configsTests {
			if err := tests(instanceConfigs); err != nil {
				return err
			}
		}

		return nil
	}
}

func checkInstanceDiskExists(instance *linodego.Instance, label string, instanceDisk *linodego.InstanceDisk) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		if instance == nil || instance.ID == 0 {
			return fmt.Errorf("Error fetching disks: invalid Instance argument")
		}

		instanceDisks, err := client.ListInstanceDisks(context.Background(), instance.ID, nil)
		if err != nil {
			return fmt.Errorf("Error fetching disks: %s", err)
		}

		if len(instanceDisks) == 0 {
			return fmt.Errorf("No disks")
		}

		for _, disk := range instanceDisks {
			if disk.Label == label {
				*instanceDisk = disk
				return nil
			}
		}

		return fmt.Errorf("Disk not found: %s", label)
	}
}

func checkComputeInstanceDisk(instance *linodego.Instance, label string, size int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		if instance == nil || instance.ID == 0 {
			return fmt.Errorf("Error fetching disks: invalid Instance argument")
		}

		instanceDisks, err := client.ListInstanceDisks(context.Background(), instance.ID, nil)
		if err != nil {
			return fmt.Errorf("Error fetching disks: %s", err)
		}

		if len(instanceDisks) == 0 {
			return fmt.Errorf("No disks")
		}

		for _, disk := range instanceDisks {
			if disk.Label == label && disk.Size == size {
				return nil
			}
		}

		return fmt.Errorf("Disk not found: %s", label)
	}
}

// stateCheckComputeInstanceConfigs is the StateCheck equivalent of checkComputeInstanceConfigs.
// IMPORTANT: This relies on the instance pointer being populated by StateCheckInstanceExists
// which must appear EARLIER in the ConfigStateChecks slice.
func stateCheckComputeInstanceConfigs(instance *linodego.Instance, configsTests ...testConfigsFunc) statecheck.StateCheck {
	return acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		if instance == nil || instance.ID == 0 {
			resp.Error = fmt.Errorf("Error fetching configs: invalid Instance argument")
			return
		}

		instanceConfigs, err := client.ListInstanceConfigs(context.Background(), instance.ID, nil)
		if err != nil {
			resp.Error = fmt.Errorf("Error fetching configs: %s", err)
			return
		}

		if len(instanceConfigs) == 0 {
			resp.Error = fmt.Errorf("No configs")
			return
		}

		for _, tests := range configsTests {
			if err := tests(instanceConfigs); err != nil {
				resp.Error = err
				return
			}
		}
	})
}

// stateCheckComputeInstanceDisk is the StateCheck equivalent of checkComputeInstanceDisk.
func stateCheckComputeInstanceDisk(instance *linodego.Instance, label string, size int) statecheck.StateCheck {
	return acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		if instance == nil || instance.ID == 0 {
			resp.Error = fmt.Errorf("Error fetching disks: invalid Instance argument")
			return
		}

		instanceDisks, err := client.ListInstanceDisks(context.Background(), instance.ID, nil)
		if err != nil {
			resp.Error = fmt.Errorf("Error fetching disks: %s", err)
			return
		}

		if len(instanceDisks) == 0 {
			resp.Error = fmt.Errorf("No disks")
			return
		}

		for _, disk := range instanceDisks {
			if disk.Label == label && disk.Size == size {
				return
			}
		}

		resp.Error = fmt.Errorf("Disk not found: %s", label)
	})
}

// stateCheckInstanceDisks is the StateCheck equivalent of checkInstanceDisks.
func stateCheckInstanceDisks(instance *linodego.Instance, disksTests ...testDisksFunc) statecheck.StateCheck {
	return acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		if instance == nil || instance.ID == 0 {
			resp.Error = fmt.Errorf("Error fetching disks: invalid Instance argument")
			return
		}

		instanceDisks, err := client.ListInstanceDisks(context.Background(), instance.ID, nil)
		if err != nil {
			resp.Error = fmt.Errorf("Error fetching disks: %s", err)
			return
		}

		if len(instanceDisks) == 0 {
			resp.Error = fmt.Errorf("No disks")
			return
		}

		for _, tests := range disksTests {
			if err := tests(instanceDisks); err != nil {
				resp.Error = err
				return
			}
		}
	})
}

// stateCheckInstanceDiskExists is the StateCheck equivalent of checkInstanceDiskExists.
func stateCheckInstanceDiskExists(instance *linodego.Instance, label string, instanceDisk *linodego.InstanceDisk) statecheck.StateCheck {
	return acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		if instance == nil || instance.ID == 0 {
			resp.Error = fmt.Errorf("Error fetching disks: invalid Instance argument")
			return
		}

		instanceDisks, err := client.ListInstanceDisks(context.Background(), instance.ID, nil)
		if err != nil {
			resp.Error = fmt.Errorf("Error fetching disks: %s", err)
			return
		}

		if len(instanceDisks) == 0 {
			resp.Error = fmt.Errorf("No disks")
			return
		}

		for _, disk := range instanceDisks {
			if disk.Label == label {
				*instanceDisk = disk
				return
			}
		}

		resp.Error = fmt.Errorf("Disk not found: %s", label)
	})
}

// stateCheckInstancePrivateNetworkAttributes is the StateCheck equivalent of checkInstancePrivateNetworkAttributes.
func stateCheckInstancePrivateNetworkAttributes(n string) statecheck.StateCheck {
	return acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		var resourceID string
		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Address == n {
				idVal, ok := rc.AttributeValues["id"]
				if !ok {
					resp.Error = fmt.Errorf("should have a Linode ID")
					return
				}
				resourceID = idVal.(string)
				break
			}
		}

		if resourceID == "" {
			resp.Error = fmt.Errorf("should have found linode_instance resource %s", n)
			return
		}

		id, err := strconv.Atoi(resourceID)
		if err != nil {
			resp.Error = fmt.Errorf("should have an integer Linode ID: %s", err)
			return
		}

		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client
		instanceIPs, err := client.GetInstanceIPAddresses(context.Background(), id)
		if err != nil {
			resp.Error = err
			return
		}
		if len(instanceIPs.IPv4.Private) == 0 {
			resp.Error = fmt.Errorf("should have a private ip on Linode ID %d", id)
		}
	})
}

func TestAccResourceInstance_withReservedIP(t *testing.T) {
	t.Parallel()

	var instance linodego.Instance
	resourceName := "linode_instance.foobar"
	instanceName := acctest.RandomWithPrefix("tf_test")
	rootPass := acctest.RandString(16)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.WithReservedIP(t, instanceName, acceptance.PublicKeyMaterial, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resourceName, &instance),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("ipv4"), knownvalue.ListSizeExact(1)),
				},
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"root_pass", "authorized_keys", "image", "migration_type", "resize_disk", "firewall_id"},
			},
		},
	})
}

func TestAccResourceInstance_deleteWithReservedIP(t *testing.T) {
	t.Parallel()
	var instance linodego.Instance
	resourceName := "linode_instance.foobar"
	testRegion := "us-east"
	reservedIP := ""
	instanceName := acctest.RandomWithPrefix("tf_test")
	ipResourceName := "linode_networking_ip.test"
	rootPass := acctest.RandString(16)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.WithReservedIP(t, instanceName, acceptance.PublicKeyMaterial, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resourceName, &instance),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("label"), knownvalue.StringExact(instanceName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("ipv4"), knownvalue.ListSizeExact(1)),
					acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
						for _, rc := range req.State.Values.RootModule.Resources {
							if rc.Address == ipResourceName {
								addrVal, ok := rc.AttributeValues["address"]
								if !ok {
									resp.Error = fmt.Errorf("address attribute not found on %s", ipResourceName)
									return
								}
								reservedIP = addrVal.(string)
								return
							}
						}
						resp.Error = fmt.Errorf("Not found: %s", ipResourceName)
					}),
				},
			},
			{
				Config: tmpl.OnlyReservedIP(t, testRegion), // This config only includes the reserved IP resource
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
						client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

						// Check if the instance is deleted
						_, err := client.GetInstance(context.Background(), instance.ID)
						if err == nil {
							resp.Error = fmt.Errorf("Linode instance %d still exists", instance.ID)
							return
						}
						if apiErr, ok := err.(*linodego.Error); ok && apiErr.Code != 404 {
							resp.Error = fmt.Errorf("Error requesting Linode instance %d: %s", instance.ID, err)
							return
						}

						// Check if the Reserved IP still exists and is reserved
						ip, err := client.GetIPAddress(context.Background(), reservedIP)
						if err != nil {
							resp.Error = fmt.Errorf("Error checking if Reserved IP exists: %s", err)
							return
						}
						if !ip.Reserved {
							resp.Error = fmt.Errorf("Reserved IP %s is no longer reserved after instance deletion", reservedIP)
						}
					}),
				},
			},
		},
	})
}

func TestAccResourceInstance_withLock(t *testing.T) {
	var instance linodego.Instance
	label := acctest.RandomWithPrefix("tf_test")
	lockType := "cannot_delete"
	resName := "linode_instance.my-inst"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.WithLock(t, label, testRegion, lockType),
			},
			{
				RefreshState: true,
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists(resName, &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(label)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("locks"), knownvalue.ListSizeExact(1)),
					acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
						for _, rc := range req.State.Values.RootModule.Resources {
							if rc.Address != resName {
								continue
							}
							locks, ok := rc.AttributeValues["locks"].([]interface{})
							if !ok {
								resp.Error = fmt.Errorf("locks attribute not found or not a list")
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
