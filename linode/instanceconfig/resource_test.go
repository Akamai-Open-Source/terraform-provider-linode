//go:build integration || instanceconfig

package instanceconfig_test

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
	"github.com/linode/terraform-provider-linode/v3/linode/instanceconfig/tmpl"
)

var testRegion string

func init() {
	region, err := acceptance.GetRandomRegionWithCaps([]string{"vlans", "VPCs"}, "core")
	if err != nil {
		log.Fatal(err)
	}

	testRegion = region
}

func TestAccResourceInstanceConfig_basic(t *testing.T) {
	t.Parallel()

	resName := "linode_instance_config.foobar"
	instanceName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, instanceName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckExists(resName, nil),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact("my-config")),
				},
			},
			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: resourceImportStateID,
			},
		},
	})
}

func TestAccResourceInstanceConfig_deviceBlock(t *testing.T) {
	t.Parallel()

	resName := "linode_instance_config.foobar"
	instanceName := acctest.RandomWithPrefix("tf_test")
	rootPass := acctest.RandString(64)

	devicesStateChecks := []statecheck.StateCheck{
		statecheck.ExpectKnownValue(resName, tfjsonpath.New("devices").AtSliceIndex(0).AtMapKey("sda").AtSliceIndex(0).AtMapKey("disk_id"), knownvalue.NotNull()),
		statecheck.ExpectKnownValue(resName, tfjsonpath.New("devices").AtSliceIndex(0).AtMapKey("sdb").AtSliceIndex(0).AtMapKey("disk_id"), knownvalue.NotNull()),
		statecheck.ExpectKnownValue(resName, tfjsonpath.New("device").AtSliceIndex(0).AtMapKey("disk_id"), knownvalue.NotNull()),
		statecheck.ExpectKnownValue(resName, tfjsonpath.New("device").AtSliceIndex(1).AtMapKey("disk_id"), knownvalue.NotNull()),
		statecheck.ExpectKnownValue(resName, tfjsonpath.New("device").AtSliceIndex(0).AtMapKey("device_name"), knownvalue.StringExact("sda")),
		statecheck.ExpectKnownValue(resName, tfjsonpath.New("device").AtSliceIndex(1).AtMapKey("device_name"), knownvalue.StringExact("sdb")),
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			// Ensure the provider doesn't panic when creating an instance
			// with the new `device` block.
			{
				Config: tmpl.DeviceBlock(t, instanceName, testRegion, rootPass),
				ConfigStateChecks: append([]statecheck.StateCheck{
					stateCheckExists(resName, nil),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact("my-config")),
				}, devicesStateChecks...),
			},
			{
				Config: tmpl.DeviceNamedBlock(t, instanceName, testRegion, rootPass),
				ConfigStateChecks: append([]statecheck.StateCheck{
					stateCheckExists(resName, nil),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact("my-config")),
				}, devicesStateChecks...),
			},
			{
				Config: tmpl.DeviceBlock(t, instanceName, testRegion, rootPass),
				ConfigStateChecks: append([]statecheck.StateCheck{
					stateCheckExists(resName, nil),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact("my-config")),
				}, devicesStateChecks...),
			},
			{
				Config: tmpl.DeviceNamedBlock(t, instanceName, testRegion, rootPass),
				ConfigStateChecks: append([]statecheck.StateCheck{
					stateCheckExists(resName, nil),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact("my-config")),
				}, devicesStateChecks...),
			},
			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: resourceImportStateID,
			},
		},
	})
}

func TestAccResourceInstanceConfig_complex(t *testing.T) {
	t.Parallel()

	resName := "linode_instance_config.foobar"
	var instance linodego.Instance
	instanceName := acctest.RandomWithPrefix("tf_test")
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Complex(t, instanceName, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckExists(resName, nil),
					acceptance.StateCheckInstanceExists("linode_instance.foobar", &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact("my-config")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("comments"), knownvalue.StringExact("cool")),

					statecheck.ExpectKnownValue(resName, tfjsonpath.New("helpers").AtSliceIndex(0).AtMapKey("devtmpfs_automount"), knownvalue.StringExact("true")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("helpers").AtSliceIndex(0).AtMapKey("distro"), knownvalue.StringExact("true")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("helpers").AtSliceIndex(0).AtMapKey("modules_dep"), knownvalue.StringExact("true")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("helpers").AtSliceIndex(0).AtMapKey("network"), knownvalue.StringExact("true")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("helpers").AtSliceIndex(0).AtMapKey("updatedb_disabled"), knownvalue.StringExact("true")),

					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface").AtSliceIndex(0).AtMapKey("purpose"), knownvalue.StringExact("public")),

					statecheck.ExpectKnownValue(resName, tfjsonpath.New("kernel"), knownvalue.StringExact("linode/latest-64bit")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("memory_limit"), knownvalue.StringExact("512")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("root_device"), knownvalue.StringExact("/dev/sda")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("virt_mode"), knownvalue.StringExact("paravirt")),

					statecheck.ExpectKnownValue(resName, tfjsonpath.New("booted"), knownvalue.StringExact("true")),

					statecheck.ExpectKnownValue(resName, tfjsonpath.New("devices").AtSliceIndex(0).AtMapKey("sda").AtSliceIndex(0).AtMapKey("disk_id"), knownvalue.NotNull()),
				},
			},
			{
				Config: tmpl.ComplexUpdates(t, instanceName, testRegion, rootPass, true),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckExists(resName, nil),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact("my-config-updated")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("comments"), knownvalue.StringExact("cool-updated")),

					statecheck.ExpectKnownValue(resName, tfjsonpath.New("helpers").AtSliceIndex(0).AtMapKey("devtmpfs_automount"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("helpers").AtSliceIndex(0).AtMapKey("distro"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("helpers").AtSliceIndex(0).AtMapKey("modules_dep"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("helpers").AtSliceIndex(0).AtMapKey("network"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("helpers").AtSliceIndex(0).AtMapKey("updatedb_disabled"), knownvalue.StringExact("false")),

					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface").AtSliceIndex(0).AtMapKey("purpose"), knownvalue.StringExact("vlan")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringExact("cooler")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface").AtSliceIndex(0).AtMapKey("ipam_address"), knownvalue.StringExact("10.0.0.3/24")),

					statecheck.ExpectKnownValue(resName, tfjsonpath.New("kernel"), knownvalue.StringExact("linode/latest-32bit")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("memory_limit"), knownvalue.StringExact("513")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("root_device"), knownvalue.StringExact("/dev/sdb")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("virt_mode"), knownvalue.StringExact("fullvirt")),

					statecheck.ExpectKnownValue(resName, tfjsonpath.New("booted"), knownvalue.StringExact("true")),

					statecheck.ExpectKnownValue(resName, tfjsonpath.New("devices").AtSliceIndex(0).AtMapKey("sdb").AtSliceIndex(0).AtMapKey("disk_id"), knownvalue.NotNull()),
				},
			},
			{
				PreConfig: acceptance.AssertInstanceReboot(t, true, &instance),
				Config:    tmpl.ComplexUpdates(t, instanceName, testRegion, rootPass, true),
			},
			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: resourceImportStateID,
				// Remove this ignorance when the TF SDK issue is fixed
				// https://github.com/hashicorp/terraform-plugin-sdk/issues/792
				ImportStateVerifyIgnore: []string{"device"},
			},
		},
	})
}

func TestAccResourceInstanceConfig_booted(t *testing.T) {
	t.Parallel()

	var instance linodego.Instance

	resName := "linode_instance_config.foobar"
	instanceName := acctest.RandomWithPrefix("tf_test")
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Booted(t, instanceName, testRegion, false, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists("linode_instance.foobar", &instance),
					stateCheckExists(resName, nil),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact("my-config")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("booted"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("devices").AtSliceIndex(0).AtMapKey("sda").AtSliceIndex(0).AtMapKey("disk_id"), knownvalue.NotNull()),
				},
			},
			{
				PreConfig: func() {
					if instance.Status != linodego.InstanceOffline {
						t.Fatalf("expected instance to be offline, got %s", instance.Status)
					}
				},
				Config: tmpl.Booted(t, instanceName, testRegion, true, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists("linode_instance.foobar", &instance),
					stateCheckExists(resName, nil),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact("my-config")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("booted"), knownvalue.StringExact("true")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("devices").AtSliceIndex(0).AtMapKey("sda").AtSliceIndex(0).AtMapKey("disk_id"), knownvalue.NotNull()),
				},
			},
			{
				PreConfig: func() {
					if instance.Status != linodego.InstanceRunning {
						t.Fatalf("expected instance to be running, got %s", instance.Status)
					}
				},
				Config: tmpl.Booted(t, instanceName, testRegion, true, rootPass),
			},
			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: resourceImportStateID,
			},
		},
	})
}

func TestAccResourceInstanceConfig_bootedSwap(t *testing.T) {
	t.Parallel()

	acceptance.RunTestWithRetries(t, 3, func(t *acceptance.WrappedT) {
		var instance linodego.Instance

		config1Name := "linode_instance_config.foobar1"
		config2Name := "linode_instance_config.foobar2"
		instanceName := acctest.RandomWithPrefix("tf_test")
		rootPass := acctest.RandString(64)

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { acceptance.PreCheck(t) },
			ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
			CheckDestroy:             checkDestroy,
			Steps: []resource.TestStep{
				{
					Config: tmpl.BootedSwap(t, instanceName, testRegion, false, rootPass),
					ConfigStateChecks: []statecheck.StateCheck{
						acceptance.StateCheckInstanceExists("linode_instance.foobar", &instance),
						stateCheckExists(config1Name, nil),
						stateCheckExists(config2Name, nil),
						statecheck.ExpectKnownValue(config1Name, tfjsonpath.New("booted"), knownvalue.StringExact("false")),
						statecheck.ExpectKnownValue(config2Name, tfjsonpath.New("booted"), knownvalue.StringExact("true")),
					},
				},
				{
					PreConfig: func() {
						if instance.Status != linodego.InstanceRunning {
							t.Fatalf("expected instance to be running, got %s", instance.Status)
						}
					},
					Config: tmpl.BootedSwap(t, instanceName, testRegion, true, rootPass),
					ConfigStateChecks: []statecheck.StateCheck{
						acceptance.StateCheckInstanceExists("linode_instance.foobar", &instance),
						stateCheckExists(config1Name, nil),
						stateCheckExists(config2Name, nil),
						statecheck.ExpectKnownValue(config1Name, tfjsonpath.New("booted"), knownvalue.StringExact("true")),
						statecheck.ExpectKnownValue(config2Name, tfjsonpath.New("booted"), knownvalue.StringExact("false")),
					},
				},
				{
					PreConfig: func() {
						if instance.Status != linodego.InstanceRunning {
							t.Fatalf("expected instance to be running, got %s", instance.Status)
						}
					},
					Config: tmpl.BootedSwap(t, instanceName, testRegion, true, rootPass),
				},
			},
		})
	})
}

func TestAccResourceInstanceConfig_provisioner(t *testing.T) {
	t.Parallel()

	var instance linodego.Instance

	resName := "linode_instance_config.foobar"
	instanceName := acctest.RandomWithPrefix("tf_test")
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Provisioner(t, instanceName, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists("linode_instance.foobar", &instance),
					stateCheckExists(resName, nil),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact("my-config")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("booted"), knownvalue.StringExact("true")),
				},
			},
			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: resourceImportStateID,
			},
		},
	})
}

func TestAccResourceInstanceConfig_vpcInterface(t *testing.T) {
	t.Parallel()

	resName := "linode_instance_config.foobar"
	networkDSName := "data.linode_instance_networking.foobar"
	instanceName := acctest.RandomWithPrefix("tf-test")
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.VPCInterface(t, instanceName, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckExists(resName, nil),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface").AtSliceIndex(0).AtMapKey("purpose"), knownvalue.StringExact("public")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface").AtSliceIndex(1).AtMapKey("purpose"), knownvalue.StringExact("vpc")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface").AtSliceIndex(1).AtMapKey("ipv4").AtSliceIndex(0).AtMapKey("vpc"), knownvalue.StringExact("10.0.4.250")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface").AtSliceIndex(1).AtMapKey("ip_ranges").AtSliceIndex(0), knownvalue.StringExact("10.0.4.101/32")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface").AtSliceIndex(1).AtMapKey("ipv4").AtSliceIndex(0).AtMapKey("nat_1_1"), knownvalue.NotNull()),

					statecheck.ExpectKnownValue(networkDSName, tfjsonpath.New("ipv4").AtSliceIndex(0).AtMapKey("public").AtSliceIndex(0).AtMapKey("vpc_nat_1_1").AtMapKey("address"), knownvalue.StringExact("10.0.4.250")),
					statecheck.ExpectKnownValue(networkDSName, tfjsonpath.New("ipv4").AtSliceIndex(0).AtMapKey("public").AtSliceIndex(0).AtMapKey("vpc_nat_1_1").AtMapKey("vpc_id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(networkDSName, tfjsonpath.New("ipv4").AtSliceIndex(0).AtMapKey("public").AtSliceIndex(0).AtMapKey("vpc_nat_1_1").AtMapKey("subnet_id"), knownvalue.NotNull()),
				},
			},
			{
				Config: tmpl.VPCInterfaceUpdated(t, instanceName, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckExists(resName, nil),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface").AtSliceIndex(0).AtMapKey("purpose"), knownvalue.StringExact("public")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface").AtSliceIndex(1).AtMapKey("purpose"), knownvalue.StringExact("vpc")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface").AtSliceIndex(1).AtMapKey("ipv4").AtSliceIndex(0).AtMapKey("vpc"), knownvalue.StringExact("10.0.4.249")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface").AtSliceIndex(1).AtMapKey("active"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface").AtSliceIndex(1).AtMapKey("ip_ranges").AtSliceIndex(0), knownvalue.StringExact("10.0.4.100/32")),

					statecheck.ExpectKnownValue(networkDSName, tfjsonpath.New("ipv4").AtSliceIndex(0).AtMapKey("public").AtSliceIndex(0).AtMapKey("vpc_nat_1_1"), knownvalue.ListSizeExact(0)),
				},
			},
			{
				Config: tmpl.VPCInterfaceSwapped(t, instanceName, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckExists(resName, nil),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface").AtSliceIndex(1).AtMapKey("purpose"), knownvalue.StringExact("public")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface").AtSliceIndex(0).AtMapKey("purpose"), knownvalue.StringExact("vpc")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface").AtSliceIndex(0).AtMapKey("ipv4").AtSliceIndex(0).AtMapKey("vpc"), knownvalue.StringExact("10.0.4.249")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface").AtSliceIndex(0).AtMapKey("active"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface").AtSliceIndex(0).AtMapKey("ip_ranges").AtSliceIndex(0), knownvalue.StringExact("10.0.4.100/32")),
				},
			},
			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: resourceImportStateID,
			},
			{
				Config: tmpl.VPCInterfaceOnly(t, instanceName, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckExists(resName, nil),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface").AtSliceIndex(0).AtMapKey("purpose"), knownvalue.StringExact("vpc")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface").AtSliceIndex(0).AtMapKey("ipv4").AtSliceIndex(0).AtMapKey("vpc"), knownvalue.StringExact("10.0.4.249")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface").AtSliceIndex(0).AtMapKey("active"), knownvalue.StringExact("false")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface").AtSliceIndex(0).AtMapKey("ip_ranges").AtSliceIndex(0), knownvalue.StringExact("10.0.4.100/32")),
				},
			},
			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: resourceImportStateID,
			},
			{
				Config: tmpl.VPCInterfaceRemoved(t, instanceName, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckExists(resName, nil),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("interface").AtSliceIndex(0).AtMapKey("purpose"), knownvalue.StringExact("public")),
				},
			},
			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: resourceImportStateID,
			},
		},
	})
}

// Test case to ensure instances manually booted into rescue mode
// will not crash the provider.
func TestAccResourceInstanceConfig_rescueBooted(t *testing.T) {
	t.Parallel()

	var instance linodego.Instance

	resName := "linode_instance_config.foobar"
	instanceResName := "linode_instance.foobar"

	instanceName := acctest.RandomWithPrefix("tf_test")
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Complex(t, instanceName, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckExists(resName, nil),
					acceptance.StateCheckInstanceExists(instanceResName, &instance),
				},
			},
			{
				PreConfig: func() {
					client, err := acceptance.GetTestClient()
					if err != nil {
						t.Fatal(err)
					}

					poller, err := client.NewEventPoller(context.Background(), instance.ID, linodego.EntityLinode, linodego.ActionLinodeReboot)
					if err != nil {
						t.Fatalf("failed to create event poller: %v", err)
					}

					if err := client.RescueInstance(context.Background(), instance.ID, linodego.InstanceRescueOptions{}); err != nil {
						t.Fatalf("failed to boot instance into rescue mode: %v", err)
					}

					if _, err := poller.WaitForFinished(context.Background(), 240); err != nil {
						t.Fatalf("failed to wait for instance to boot into rescue mode: %v", err)
					}
				},
				Config: tmpl.ComplexUpdates(t, instanceName, testRegion, rootPass, false),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckExists(resName, nil),
					acceptance.StateCheckInstanceExists(instanceResName, &instance),
					// The provider should not be booted into this config
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("booted"), knownvalue.StringExact("false")),
				},
			},
			{
				Config: tmpl.Complex(t, instanceName, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckExists(resName, nil),
					acceptance.StateCheckInstanceExists(instanceResName, &instance),
					// The provider should now have been rebooted into this config
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("booted"), knownvalue.StringExact("true")),
				},
			},
			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: resourceImportStateID,
				// Remove this ignorance when the TF SDK issue is fixed
				// https://github.com/hashicorp/terraform-plugin-sdk/issues/792
				ImportStateVerifyIgnore: []string{"device"},
			},
		},
	})
}

func TestAccResourceInstanceConfig_vpcInterfaceIPv6(t *testing.T) {
	t.Parallel()

	resName := "linode_instance_config.foobar"
	instanceName := acctest.RandomWithPrefix("tf-test")
	rootPass := acctest.RandString(64)

	// TODO (VPC Dual Stack): Remove region hardcoding
	targetRegion := "no-osl-1"

	interfacePath := tfjsonpath.New("interface").AtSliceIndex(0)
	ipv4Path := interfacePath.AtMapKey("ipv4").AtSliceIndex(0)
	ipv6Path := interfacePath.AtMapKey("ipv6").AtSliceIndex(0)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.VPCInterfaceIPv60(t, instanceName, targetRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckExists(resName, nil),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("interface"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						resName,
						interfacePath.AtMapKey("purpose"),
						knownvalue.StringExact("vpc"),
					),
					statecheck.ExpectKnownValue(
						resName,
						interfacePath.AtMapKey("subnet_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						interfacePath.AtMapKey("ip_ranges").AtSliceIndex(0),
						knownvalue.StringExact("10.0.4.101/32"),
					),

					statecheck.ExpectKnownValue(
						resName,
						ipv4Path.AtMapKey("vpc"),
						knownvalue.StringExact("10.0.4.250"),
					),
					statecheck.ExpectKnownValue(
						resName,
						ipv4Path.AtMapKey("nat_1_1"),
						knownvalue.NotNull(),
					),

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
						ipv6Path.AtMapKey("range"),
						knownvalue.ListSizeExact(1),
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
							AtSliceIndex(0).
							AtMapKey("assigned_range"),
						knownvalue.NotNull(),
					),
				},
			},
			{
				Config: tmpl.VPCInterfaceIPv61(t, instanceName, targetRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckExists(resName, nil),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("interface"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						resName,
						interfacePath.AtMapKey("purpose"),
						knownvalue.StringExact("vpc"),
					),
					statecheck.ExpectKnownValue(
						resName,
						interfacePath.AtMapKey("subnet_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						interfacePath.AtMapKey("ip_ranges").AtSliceIndex(0),
						knownvalue.StringExact("10.0.4.101/32"),
					),

					statecheck.ExpectKnownValue(
						resName,
						ipv4Path.AtMapKey("vpc"),
						knownvalue.StringExact("10.0.4.250"),
					),
					statecheck.ExpectKnownValue(
						resName,
						ipv4Path.AtMapKey("nat_1_1"),
						knownvalue.NotNull(),
					),

					// The full path is required due to a bug that causes reusing a path
					// to break bool checks under certain conditions.
					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.AtMapKey("is_public"),
						knownvalue.Bool(true),
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
							AtSliceIndex(0).AtMapKey("range"),
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
						ipv6Path.AtMapKey("range"),
						knownvalue.ListSizeExact(2),
					),
					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.AtMapKey("range").AtSliceIndex(0).AtMapKey("range"),
						knownvalue.NotNull(),
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
						ipv6Path.AtMapKey("range").AtSliceIndex(1).AtMapKey("range"),
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
				},
			},
			{
				Config: tmpl.VPCInterfaceIPv61(t, instanceName, targetRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckExists(resName, nil),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("interface"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						resName,
						interfacePath.AtMapKey("purpose"),
						knownvalue.StringExact("vpc"),
					),
					statecheck.ExpectKnownValue(
						resName,
						interfacePath.AtMapKey("subnet_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						interfacePath.AtMapKey("ip_ranges").AtSliceIndex(0),
						knownvalue.StringExact("10.0.4.101/32"),
					),

					statecheck.ExpectKnownValue(
						resName,
						ipv4Path.AtMapKey("vpc"),
						knownvalue.StringExact("10.0.4.250"),
					),
					statecheck.ExpectKnownValue(
						resName,
						ipv4Path.AtMapKey("nat_1_1"),
						knownvalue.NotNull(),
					),

					// The full path is required due to a bug that causes reusing a path
					// to break bool checks under certain conditions.
					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.AtMapKey("is_public"),
						knownvalue.Bool(true),
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
						ipv6Path.AtMapKey("range"),
						knownvalue.ListSizeExact(2),
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
							AtSliceIndex(0).
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
					statecheck.ExpectKnownValue(
						resName,
						ipv6Path.
							AtMapKey("range").
							AtSliceIndex(1).
							AtMapKey("assigned_range"),
						knownvalue.NotNull(),
					),
				},
			},
			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: resourceImportStateID,
			},
		},
	})
}

func checkExists(name string, config *linodego.InstanceConfig) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		linodeID, id, err := getResourceIDs(rs)
		if err != nil {
			return fmt.Errorf("failed to get disk info: %v", err)
		}

		found, err := client.GetInstanceConfig(context.Background(), linodeID, id)
		if err != nil {
			return fmt.Errorf("error retrieving state of config %s: %s", rs.Primary.Attributes["label"], err)
		}

		if config != nil {
			*config = *found
		}

		return nil
	}
}

func stateCheckExists(name string, config *linodego.InstanceConfig) statecheck.StateCheck {
	return acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		// Find the resource in tfjson state
		var found bool
		var idStr, linodeIDStr string
		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Address == name {
				found = true
				idVal, ok := rc.AttributeValues["id"]
				if !ok || idVal == nil {
					resp.Error = fmt.Errorf("No ID is set")
					return
				}
				idStr = fmt.Sprintf("%v", idVal)

				lidVal, ok := rc.AttributeValues["linode_id"]
				if !ok || lidVal == nil {
					resp.Error = fmt.Errorf("No linode_id is set")
					return
				}
				linodeIDStr = fmt.Sprintf("%v", lidVal)
				break
			}
		}

		if !found {
			resp.Error = fmt.Errorf("Not found: %s", name)
			return
		}

		id, err := strconv.Atoi(idStr)
		if err != nil {
			resp.Error = fmt.Errorf("failed to parse config ID: %v", err)
			return
		}

		linodeID, err := strconv.Atoi(linodeIDStr)
		if err != nil {
			resp.Error = fmt.Errorf("failed to parse linode_id: %v", err)
			return
		}

		foundConfig, err := client.GetInstanceConfig(context.Background(), linodeID, id)
		if err != nil {
			resp.Error = fmt.Errorf("error retrieving state of config: %s", err)
			return
		}

		if config != nil {
			*config = *foundConfig
		}
	})
}

func checkDestroy(s *terraform.State) error {
	client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_instance_config" {
			continue
		}

		linodeID, id, err := getResourceIDs(rs)
		if err != nil {
			return fmt.Errorf("failed to get disk info: %v", err)
		}

		_, err = client.GetInstanceConfig(context.Background(), linodeID, id)

		if err == nil {
			return fmt.Errorf("config with id %d still exists", id)
		}

		if apiErr, ok := err.(*linodego.Error); ok && apiErr.Code != 404 {
			return fmt.Errorf("error requesting config with id %d", id)
		}
	}

	return nil
}

func getResourceIDs(rs *terraform.ResourceState) (int, int, error) {
	id, err := strconv.Atoi(rs.Primary.ID)
	if err != nil {
		return 0, 0, err
	}

	linodeID, err := strconv.Atoi(rs.Primary.Attributes["linode_id"])
	if err != nil {
		return 0, 0, err
	}

	return linodeID, id, nil
}

func resourceImportStateID(s *terraform.State) (string, error) {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_instance_config" {
			continue
		}

		linodeID, id, err := getResourceIDs(rs)
		if err != nil {
			return "", fmt.Errorf("failed to get config info: %v", err)
		}

		return fmt.Sprintf("%d,%d", linodeID, id), nil
	}

	return "", fmt.Errorf("Error finding linode_instance_config")
}
