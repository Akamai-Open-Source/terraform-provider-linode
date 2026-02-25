//go:build integration || nbconfigs

package nbconfigs_test

import (
	"log"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/nbconfigs/tmpl"
)

func TestAccDataSourceNodeBalancerConfigs_basic(t *testing.T) {
	t.Parallel()

	resourceName := "data.linode_nodebalancer_configs.foo"

	nbLabel := acctest.RandomWithPrefix("tf_test")
	nbRegion, err := acceptance.GetRandomRegionWithCaps([]string{"NodeBalancers"}, "core")
	if err != nil {
		log.Fatal(err)
	}

	port := 80

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t, nbLabel, nbRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancer_configs"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancer_configs").AtSliceIndex(0).AtMapKey("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancer_configs").AtSliceIndex(0).AtMapKey("nodebalancer_id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancer_configs").AtSliceIndex(0).AtMapKey("protocol"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancer_configs").AtSliceIndex(0).AtMapKey("proxy_protocol"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancer_configs").AtSliceIndex(0).AtMapKey("port"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancer_configs").AtSliceIndex(0).AtMapKey("check_interval"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancer_configs").AtSliceIndex(0).AtMapKey("check_passive"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancer_configs").AtSliceIndex(0).AtMapKey("udp_check_port"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancer_configs").AtSliceIndex(0).AtMapKey("udp_session_timeout"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancer_configs").AtSliceIndex(0).AtMapKey("cipher_suite"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancer_configs").AtSliceIndex(0).AtMapKey("ssl_common"), knownvalue.Null()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancer_configs").AtSliceIndex(0).AtMapKey("ssl_ciphersuite"), knownvalue.Null()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancer_configs").AtSliceIndex(0).AtMapKey("node_status").AtSliceIndex(0).AtMapKey("up"), knownvalue.Int64Exact(0)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancer_configs").AtSliceIndex(0).AtMapKey("node_status").AtSliceIndex(0).AtMapKey("down"), knownvalue.Int64Exact(0)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancer_configs").AtSliceIndex(0).AtMapKey("ssl_cert"), knownvalue.Null()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancer_configs").AtSliceIndex(0).AtMapKey("ssl_key"), knownvalue.Null()),
				},
			},
			{
				Config: tmpl.DataFilter(t, nbLabel, nbRegion, port),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancer_configs"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("nodebalancer_configs").AtSliceIndex(0).AtMapKey("port"), knownvalue.Int64Exact(int64(port))),
				},
			},
		},
	})
}
