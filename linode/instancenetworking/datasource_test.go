//go:build integration || instancenetworking

package instancenetworking_test

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
	"github.com/linode/terraform-provider-linode/v3/linode/instancenetworking/tmpl"
)

const testInstanceNetworkResName = "data.linode_instance_networking.test"

var testRegion string

func init() {
	region, err := acceptance.GetRandomRegionWithCaps([]string{"VPCs", "Linodes"}, "core")
	if err != nil {
		log.Fatal(err)
	}

	testRegion = region
}

func TestAccDataSourceInstanceNetworking_basic(t *testing.T) {
	t.Parallel()

	var instance linodego.Instance

	name := acctest.RandomWithPrefix("tf_test")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t, name, testRegion),
			},
			{
				Config: tmpl.DataBasic(t, name, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists("linode_instance.foobar", &instance),
					statecheck.ExpectKnownValue(testInstanceNetworkResName, tfjsonpath.New("ipv4").AtSliceIndex(0).AtMapKey("private"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceNetworkResName, tfjsonpath.New("ipv4").AtSliceIndex(0).AtMapKey("public"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceNetworkResName, tfjsonpath.New("ipv4").AtSliceIndex(0).AtMapKey("reserved"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceNetworkResName, tfjsonpath.New("ipv4").AtSliceIndex(0).AtMapKey("shared"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceNetworkResName, tfjsonpath.New("ipv6").AtSliceIndex(0).AtMapKey("global"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceNetworkResName, tfjsonpath.New("ipv6").AtSliceIndex(0).AtMapKey("link_local"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceNetworkResName, tfjsonpath.New("ipv6").AtSliceIndex(0).AtMapKey("slaac"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func TestAccDataSourceInstanceNetworking_vpc(t *testing.T) {
	t.Parallel()

	instanceVPCIP := "10.0.0.3"
	name := acctest.RandomWithPrefix("tf-test")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataVPC(t, name, testRegion, "10.0.0.0/24", instanceVPCIP),
			},
			{
				Config: tmpl.DataVPC(t, name, testRegion, "10.0.0.0/24", instanceVPCIP),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(testInstanceNetworkResName, tfjsonpath.New("ipv4").AtSliceIndex(0).AtMapKey("vpc"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceNetworkResName, tfjsonpath.New("ipv4").AtSliceIndex(0).AtMapKey("vpc").AtSliceIndex(0).AtMapKey("address"), knownvalue.StringExact(instanceVPCIP)),
				},
			},
		},
	})
}

func TestAccDataSourceInstanceNetworking_basicwithReseved(t *testing.T) {
	t.Parallel()

	var instance linodego.Instance

	name := acctest.RandomWithPrefix("tf_test")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic_withReservedField(t, name, testRegion),
			},
			{
				Config: tmpl.DataBasic_withReservedField(t, name, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists("linode_instance.foobar", &instance),
					statecheck.ExpectKnownValue(testInstanceNetworkResName, tfjsonpath.New("ipv4").AtSliceIndex(0).AtMapKey("private"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceNetworkResName, tfjsonpath.New("ipv4").AtSliceIndex(0).AtMapKey("public"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceNetworkResName, tfjsonpath.New("ipv4").AtSliceIndex(0).AtMapKey("reserved"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceNetworkResName, tfjsonpath.New("ipv4").AtSliceIndex(0).AtMapKey("shared"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceNetworkResName, tfjsonpath.New("ipv6").AtSliceIndex(0).AtMapKey("global"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceNetworkResName, tfjsonpath.New("ipv6").AtSliceIndex(0).AtMapKey("link_local"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceNetworkResName, tfjsonpath.New("ipv6").AtSliceIndex(0).AtMapKey("slaac"), knownvalue.NotNull()),
				},
			},
		},
	})
}

// TODO (Linode Interfaces): Add test for new interface_id field.
