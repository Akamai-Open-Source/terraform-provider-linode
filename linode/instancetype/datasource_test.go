//go:build integration || instancetype

package instancetype_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/linodego"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/instancetype/tmpl"
)

func TestAccDataSourceLinodeInstanceType_basic(t *testing.T) {
	t.Parallel()

	resourceName := "data.linode_instance_type.foobar"

	client, err := acceptance.GetTestClient()
	if err != nil {
		t.Fatal(err)
	}

	// Resolve a type with region-specific pricing
	allTypes, err := client.ListTypes(context.Background(), nil)
	if err != nil {
		t.Fatalf("failed to list regions: %s", err)
	}

	var targetType linodego.LinodeType
	for _, v := range allTypes {
		if len(v.RegionPrices) > 0 && v.RegionPrices[0].Hourly > 0 {
			targetType = v
			break
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t, targetType.ID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("id"), knownvalue.StringExact(targetType.ID)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("label"), knownvalue.StringExact(targetType.Label)),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("disk"),
						knownvalue.StringExact(strconv.FormatInt(int64(targetType.Disk), 10)),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("class"),
						knownvalue.StringExact(string(targetType.Class)),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("memory"),
						knownvalue.StringExact(strconv.FormatInt(int64(targetType.Memory), 10)),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("vcpus"),
						knownvalue.StringExact(strconv.FormatInt(int64(targetType.VCPUs), 10)),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("accelerated_devices"),
						knownvalue.StringExact(strconv.FormatInt(int64(targetType.AcceleratedDevices), 10)),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("network_out"),
						knownvalue.StringExact(strconv.FormatInt(int64(targetType.NetworkOut), 10)),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("price").AtSliceIndex(0).AtMapKey("hourly"),
						knownvalue.StringExact(strconv.FormatFloat(float64(targetType.Price.Hourly), 'f', -1, 64)),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("price").AtSliceIndex(0).AtMapKey("monthly"),
						knownvalue.StringExact(strconv.FormatFloat(float64(targetType.Price.Monthly), 'f', -1, 64)),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("addons").AtSliceIndex(0).AtMapKey("backups").AtSliceIndex(0).AtMapKey("price").AtSliceIndex(0).AtMapKey("hourly"),
						knownvalue.StringExact(strconv.FormatFloat(float64(targetType.Addons.Backups.Price.Hourly), 'f', -1, 64)),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("addons").AtSliceIndex(0).AtMapKey("backups").AtSliceIndex(0).AtMapKey("price").AtSliceIndex(0).AtMapKey("monthly"),
						knownvalue.StringExact(strconv.FormatFloat(float64(targetType.Addons.Backups.Price.Monthly), 'f', -1, 64)),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("region_prices").AtSliceIndex(0).AtMapKey("monthly"),
						knownvalue.StringExact(strconv.FormatFloat(float64(targetType.RegionPrices[0].Monthly), 'f', -1, 64)),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("region_prices").AtSliceIndex(0).AtMapKey("hourly"),
						knownvalue.StringExact(strconv.FormatFloat(float64(targetType.RegionPrices[0].Hourly), 'f', -1, 64)),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("addons").AtSliceIndex(0).AtMapKey("backups").AtSliceIndex(0).AtMapKey("region_prices").AtSliceIndex(0).AtMapKey("monthly"),
						knownvalue.StringExact(strconv.FormatFloat(float64(targetType.Addons.Backups.RegionPrices[0].Monthly), 'f', -1, 64)),
					),
					statecheck.ExpectKnownValue(
						resourceName,
						tfjsonpath.New("addons").AtSliceIndex(0).AtMapKey("backups").AtSliceIndex(0).AtMapKey("region_prices").AtSliceIndex(0).AtMapKey("hourly"),
						knownvalue.StringExact(strconv.FormatFloat(float64(targetType.Addons.Backups.RegionPrices[0].Hourly), 'f', -1, 64)),
					),
				},
			},
		},
	})
}
