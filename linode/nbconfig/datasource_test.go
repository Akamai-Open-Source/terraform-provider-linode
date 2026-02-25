//go:build integration || nbconfig

package nbconfig_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/linodego"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/nbconfig/tmpl"
)

func TestAccDataSourceNodeBalancerConfig_basic(t *testing.T) {
	t.Parallel()

	resName := "data.linode_nodebalancer_config.foofig"
	nodebalancerName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkNodeBalancerConfigDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t, nodebalancerName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodeBalancerConfigExists,
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("port"), knownvalue.Int64Exact(8080)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("protocol"), knownvalue.StringExact(string(linodego.ProtocolHTTP))),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("check"), knownvalue.StringExact(string(linodego.CheckHTTP))),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("check_path"), knownvalue.StringExact("/")),

					statecheck.ExpectKnownValue(resName, tfjsonpath.New("algorithm"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("stickiness"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("check_attempts"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("check_timeout"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("check_interval"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("check_passive"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("udp_check_port"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("udp_session_timeout"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("cipher_suite"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("ssl_commonname"), knownvalue.Null()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("ssl_fingerprint"), knownvalue.Null()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("node_status").AtSliceIndex(0).AtMapKey("up"), knownvalue.Int64Exact(0)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("node_status").AtSliceIndex(0).AtMapKey("down"), knownvalue.Int64Exact(0)),
				},
			},
		},
	})
}
