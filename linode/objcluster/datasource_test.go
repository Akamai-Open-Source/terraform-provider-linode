//go:build integration || objcluster

package objcluster_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/objcluster/tmpl"
)

func TestAccDataSourceObjectCluster_basic(t *testing.T) {
	t.Parallel()

	objectStorageClusterID := "us-east-1"
	region := "us-east"
	resourceName := "data.linode_object_storage_cluster.foobar"
	staticSiteDomain := "website-us-east-1.linodeobjects.com"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t, objectStorageClusterID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("region"), knownvalue.StringExact(region)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("id"), knownvalue.StringExact(objectStorageClusterID)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("static_site_domain"), knownvalue.StringExact(staticSiteDomain)),
				},
			},
		},
	})
}
