//go:build integration || instanceip

package instanceip_test

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
	"github.com/linode/terraform-provider-linode/v3/linode/instanceip/tmpl"
)

const testInstanceIPResName = "linode_instance_ip.test"

var testRegion string

func init() {
	region, err := acceptance.GetRandomRegionWithCaps(nil, "core")
	if err != nil {
		log.Fatal(err)
	}

	testRegion = region
}

func TestAccInstanceIP_basic(t *testing.T) {
	t.Parallel()

	var instance linodego.Instance

	name := acctest.RandomWithPrefix("tf_test")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, name, testRegion, true),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists("linode_instance.foobar", &instance),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("address"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("gateway"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("prefix"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("rdns"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("subnet_mask"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("type"), knownvalue.StringExact("ipv4")),
				},
			},
			{
				PreConfig: func() {
					acceptance.AssertInstanceReboot(t, true, &instance)
				},
				Config: tmpl.Basic(t, name, testRegion, true),
			},
		},
	})
}

func TestAccInstanceIP_noboot(t *testing.T) {
	t.Parallel()

	var instance linodego.Instance

	name := acctest.RandomWithPrefix("tf_test")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.NoBoot(t, name, testRegion, true),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists("linode_instance.foobar", &instance),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("address"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("gateway"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("prefix"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("rdns"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("subnet_mask"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("type"), knownvalue.StringExact("ipv4")),
				},
			},
			{
				Config: tmpl.NoBoot(t, name, testRegion, true),
				PreConfig: func() {
					acceptance.AssertInstanceReboot(t, false, &instance)
				},
			},
		},
	})
}

func TestAccInstanceIP_noApply(t *testing.T) {
	t.Parallel()

	var instance linodego.Instance

	name := acctest.RandomWithPrefix("tf_test")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             acceptance.CheckInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, name, testRegion, false),
				ConfigStateChecks: []statecheck.StateCheck{
					acceptance.StateCheckInstanceExists("linode_instance.foobar", &instance),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("address"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("gateway"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("prefix"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("rdns"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("subnet_mask"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("region"), knownvalue.StringExact(testRegion)),
					statecheck.ExpectKnownValue(testInstanceIPResName, tfjsonpath.New("type"), knownvalue.StringExact("ipv4")),
				},
			},
			{
				PreConfig: func() {
					acceptance.AssertInstanceReboot(t, false, &instance)
				},
				Config: tmpl.Basic(t, name, testRegion, false),
			},
		},
	})
}
