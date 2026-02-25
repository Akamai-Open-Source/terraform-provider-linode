package acceptance

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/linode/linodego"
	"github.com/linode/terraform-provider-linode/v3/linode/helper"
)

func CheckInstanceExists(name string, instance *linodego.Instance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error parsing %v to int", rs.Primary.ID)
		}

		found, err := client.GetInstance(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Error retrieving state of Instance %s: %s", rs.Primary.Attributes["label"], err)
		}

		*instance = *found

		return nil
	}
}

// StateCheckInstanceExists is the ConfigStateChecks-compatible equivalent of CheckInstanceExists.
// It verifies that a Linode instance resource exists in the Terraform state and populates
// the provided instance pointer with the API response for downstream assertions.
func StateCheckInstanceExists(name string, instance *linodego.Instance) statecheck.StateCheck {
	return CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		client := TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		// Find the resource in tfjson state
		var resourceID string
		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Address == name {
				idVal, ok := rc.AttributeValues["id"]
				if !ok {
					resp.Error = fmt.Errorf("No ID is set for %s", name)
					return
				}
				resourceID = fmt.Sprintf("%v", idVal)
				break
			}
		}

		if resourceID == "" {
			resp.Error = fmt.Errorf("Not found: %s", name)
			return
		}

		id, err := strconv.Atoi(resourceID)
		if err != nil {
			resp.Error = fmt.Errorf("Error parsing %v to int", resourceID)
			return
		}

		found, err := client.GetInstance(context.Background(), id)
		if err != nil {
			resp.Error = fmt.Errorf("Error retrieving state of Instance: %s", err)
			return
		}

		*instance = *found
	})
}

func CheckInstanceDestroy(s *terraform.State) error {
	client := TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_instance" {
			continue
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error parsing %v as int", rs.Primary.ID)
		}

		if id == 0 {
			return fmt.Errorf("should not have Linode ID 0")
		}

		_, err = client.GetInstance(context.Background(), id)

		if err == nil {
			return fmt.Errorf("should not find Linode ID %d existing after delete", id)
		}

		if apiErr, ok := err.(*linodego.Error); ok && apiErr.Code != 404 {
			return fmt.Errorf("Error getting Linode ID %d: %s", id, err)
		}
	}

	return nil
}

func AssertInstanceReboot(t testing.TB, shouldRestart bool, instance *linodego.Instance) func() {
	t.Helper()

	return func() {
		client := TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client
		eventFilter := fmt.Sprintf(
			`{"entity.type": "linode", "entity.id": %d, "action": "linode_reboot", "created": { "+gte": "%s" }}`,
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
