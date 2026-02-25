package acceptance

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/linode/linodego"
	"github.com/linode/terraform-provider-linode/v3/linode/helper"
)

func CheckMySQLDatabaseExists(name string, db *linodego.MySQLDatabase) resource.TestCheckFunc {
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

		found, err := client.GetMySQLDatabase(context.Background(), id)
		if err != nil {
			return fmt.Errorf("error retrieving state of mysql database %s: %s", rs.Primary.Attributes["label"], err)
		}

		if db != nil {
			*db = *found
		}

		return nil
	}
}

// StateCheckMySQLDatabaseExists returns a statecheck.StateCheck that verifies
// a MySQL database resource exists in the Terraform state and in the Linode API.
// It populates the provided db pointer with the API response for downstream assertions.
func StateCheckMySQLDatabaseExists(name string, db *linodego.MySQLDatabase) statecheck.StateCheck {
	return CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		client := TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		// Find the resource in tfjson state
		var resourceID string
		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Address == name {
				// Extract the ID from the attribute values
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

		found, err := client.GetMySQLDatabase(context.Background(), id)
		if err != nil {
			resp.Error = fmt.Errorf("error retrieving state of mysql database: %s", err)
			return
		}

		if db != nil {
			*db = *found
		}
	})
}

func CheckPostgresDatabaseExists(name string, db *linodego.PostgresDatabase) resource.TestCheckFunc {
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

		found, err := client.GetPostgresDatabase(context.Background(), id)
		if err != nil {
			return fmt.Errorf("error retrieving state of postgres database %s: %s", rs.Primary.Attributes["label"], err)
		}

		if db != nil {
			*db = *found
		}

		return nil
	}
}

// StateCheckPostgresDatabaseExists returns a statecheck.StateCheck that verifies
// a PostgreSQL database resource exists in the Terraform state and in the Linode API.
// It populates the provided db pointer with the API response for downstream assertions.
func StateCheckPostgresDatabaseExists(name string, db *linodego.PostgresDatabase) statecheck.StateCheck {
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

		found, err := client.GetPostgresDatabase(context.Background(), id)
		if err != nil {
			resp.Error = fmt.Errorf("error retrieving state of postgres database: %s", err)
			return
		}

		if db != nil {
			*db = *found
		}
	})
}

func CheckMySQLDatabaseV2Destroy(s *terraform.State) error {
	client := TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_database_mysql_v2" {
			continue
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("failed to parse %v as int", rs.Primary.ID)
		}

		if id == 0 {
			return fmt.Errorf("should not have Linode ID 0")
		}

		_, err = client.GetMySQLDatabase(context.Background(), id)

		if err == nil {
			return fmt.Errorf("should not find database ID %d existing after delete", id)
		}

		if apiErr, ok := err.(*linodego.Error); ok && !linodego.IsNotFound(apiErr) {
			return fmt.Errorf("failed to get database ID %d: %s", id, err)
		}
	}

	return nil
}

func CheckPostgreSQLDatabaseV2Destroy(s *terraform.State) error {
	client := TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_database_postgresql_v2" {
			continue
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("failed to parse %v as int", rs.Primary.ID)
		}

		if id == 0 {
			return fmt.Errorf("should not have Linode ID 0")
		}

		_, err = client.GetPostgresDatabase(context.Background(), id)

		if err == nil {
			return fmt.Errorf("should not find database ID %d existing after delete", id)
		}

		if apiErr, ok := err.(*linodego.Error); ok && !linodego.IsNotFound(apiErr) {
			return fmt.Errorf("failed to get database ID %d: %s", id, err)
		}
	}

	return nil
}
