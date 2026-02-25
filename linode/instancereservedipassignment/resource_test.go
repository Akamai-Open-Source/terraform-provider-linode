//go:build integration || instancereservedipassignment

package instancereservedipassignment_test

import (
	"log"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/linodego"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/instancereservedipassignment/tmpl"
)

const testInstanceIPResName = "linode_reserved_ip_assignment.test"

var testRegion string

func init() {
	region, err := acceptance.GetRandomRegionWithCaps(nil, "core")
	if err != nil {
		log.Fatal(err)
	}
	testRegion = region
}

func TestAccInstanceIP_addReservedIP(t *testing.T) {
	t.Parallel()

	var instance linodego.Instance
	name := acctest.RandomWithPrefix("tf_test")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.AddReservedIP(t, name, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists("linode_instance.foobar", &instance),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("address"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("public"), knownvalue.StringExact("true")),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("linode_id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("gateway"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("subnet_mask"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("prefix"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("rdns"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("type"), knownvalue.StringExact("ipv4")),
				},
			},
		},
	})
}
