//go:build integration || instancedisk

package instancedisk_test

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/linodego"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/helper"
	"github.com/linode/terraform-provider-linode/v3/linode/instancedisk/tmpl"
)

var testRegion string

func init() {
	region, err := acceptance.GetRandomRegionWithCaps(nil, "core")
	if err != nil {
		log.Fatal(err)
	}

	testRegion = region
}

func TestSmokeTests_instancedisk(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{"TestAccResourceInstanceDisk_basic_smoke", TestAccResourceInstanceDisk_basic_smoke},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestAccResourceInstanceDisk_basic_smoke(t *testing.T) {
	t.Parallel()

	resName := "linode_instance_disk.foobar"
	label := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, label, testRegion, 2048),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckExists(resName, nil),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(label)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("size"), knownvalue.StringExact("2048")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("status"), knownvalue.StringExact("ready")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("linode_id"), knownvalue.NotNull()),
					statecheck.CompareValuePairs(
						resName, tfjsonpath.New("disk_encryption"),
						"linode_instance.foobar", tfjsonpath.New("disk_encryption"),
						compare.ValuesSame(),
					),
				},
			},
			// Resize up
			{
				Config: tmpl.Basic(t, label, testRegion, 2049),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckExists(resName, nil),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(label)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("size"), knownvalue.StringExact("2049")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("status"), knownvalue.StringExact("ready")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("linode_id"), knownvalue.NotNull()),
				},
			},
			// Resize down
			{
				Config: tmpl.Basic(t, label, testRegion, 2047),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckExists(resName, nil),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(label)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("size"), knownvalue.StringExact("2047")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("status"), knownvalue.StringExact("ready")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("linode_id"), knownvalue.NotNull()),
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

func TestAccResourceInstanceDisk_complex(t *testing.T) {
	t.Parallel()

	resName := "linode_instance_disk.foobar"
	label := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Complex(t, label, testRegion, 2048),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckExists(resName, nil),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(label)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("size"), knownvalue.StringExact("2048")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("filesystem"), knownvalue.StringExact("ext4")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("status"), knownvalue.StringExact("ready")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("linode_id"), knownvalue.NotNull()),
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

func TestAccResourceInstanceDisk_bootedResize(t *testing.T) {
	t.Parallel()

	resName := "linode_instance_disk.foobar"
	label := acctest.RandomWithPrefix("tf_test")
	rootPass := acctest.RandString(64)

	var instance linodego.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.BootedResize(t, label, testRegion, 2048, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckExists(resName, nil),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(label)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("size"), knownvalue.StringExact("2048")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("status"), knownvalue.StringExact("ready")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("linode_id"), knownvalue.NotNull()),
				},
			},
			// Resize up
			{
				Config: tmpl.BootedResize(t, label, testRegion, 2049, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckExists(resName, nil),
					acceptance.StateCheckInstanceExists("linode_instance.foobar", &instance),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(label)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("size"), knownvalue.StringExact("2049")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("status"), knownvalue.StringExact("ready")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("linode_id"), knownvalue.NotNull()),
				},
			},
			{
				PreConfig: func() {
					if instance.Status != linodego.InstanceRunning {
						t.Fatalf("expected instance to be running, found %s", instance.Status)
					}
				},
				Config: tmpl.BootedResize(t, label, testRegion, 2049, rootPass),
			},
			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateIdFunc:       resourceImportStateID,
				ImportStateVerifyIgnore: []string{"image", "root_pass"},
			},
		},
	})
}

func checkExists(name string, disk *linodego.InstanceDisk) resource.TestCheckFunc {
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

		found, err := client.GetInstanceDisk(context.Background(), linodeID, id)
		if err != nil {
			return fmt.Errorf("error retrieving state of disk %s: %s", rs.Primary.Attributes["label"], err)
		}

		if disk != nil {
			*disk = *found
		}

		return nil
	}
}

// stateCheckExists is the ConfigStateChecks-compatible equivalent of checkExists.
// It verifies that a Linode instance disk resource exists in the Terraform state and
// populates the provided disk pointer with the API response for downstream assertions.
func stateCheckExists(name string, disk *linodego.InstanceDisk) statecheck.StateCheck {
	return acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Address != name {
				continue
			}

			idVal, ok := rc.AttributeValues["id"]
			if !ok || idVal == nil {
				resp.Error = fmt.Errorf("No ID is set")
				return
			}

			id, err := strconv.Atoi(fmt.Sprintf("%v", idVal))
			if err != nil {
				resp.Error = fmt.Errorf("failed to parse disk id: %v", err)
				return
			}

			linodeIDVal, ok := rc.AttributeValues["linode_id"]
			if !ok || linodeIDVal == nil {
				resp.Error = fmt.Errorf("No linode_id is set")
				return
			}

			// linode_id is an Int64 attribute; in tfjson state it may appear as float64 or string
			linodeID, err := strconv.Atoi(fmt.Sprintf("%v", linodeIDVal))
			if err != nil {
				resp.Error = fmt.Errorf("failed to parse linode_id: %v", err)
				return
			}

			found, err := client.GetInstanceDisk(context.Background(), linodeID, id)
			if err != nil {
				resp.Error = fmt.Errorf("error retrieving state of disk %s: %s", fmt.Sprintf("%v", rc.AttributeValues["label"]), err)
				return
			}

			if disk != nil {
				*disk = *found
			}
			return
		}
		resp.Error = fmt.Errorf("Not found: %s", name)
	})
}

func checkDestroy(s *terraform.State) error {
	client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_instance_disk" {
			continue
		}

		linodeID, id, err := getResourceIDs(rs)
		if err != nil {
			return fmt.Errorf("failed to get disk info: %v", err)
		}

		_, err = client.GetInstanceDisk(context.Background(), linodeID, id)

		if err == nil {
			return fmt.Errorf("disk with id %d still exists", id)
		}

		if apiErr, ok := err.(*linodego.Error); ok && apiErr.Code != 404 {
			return fmt.Errorf("error requesting disk with id %d", id)
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
		if rs.Type != "linode_instance_disk" {
			continue
		}

		linodeID, id, err := getResourceIDs(rs)
		if err != nil {
			return "", fmt.Errorf("failed to get disk info: %v", err)
		}

		return fmt.Sprintf("%d,%d", linodeID, id), nil
	}

	return "", fmt.Errorf("Error finding linode_instance_disk")
}
