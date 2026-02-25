//go:build integration || kernels

package kernels_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/kernels/tmpl"
)

func TestAccDataSourceKernels_basic(t *testing.T) {
	t.Parallel()

	resourceName := "data.linode_kernels.kernels"

	kernelID := "linode/latest-64bit"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.DataBasic(t, kernelID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("kernels").AtSliceIndex(0).AtMapKey("id"), knownvalue.StringExact(kernelID)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("kernels").AtSliceIndex(0).AtMapKey("architecture"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("kernels").AtSliceIndex(0).AtMapKey("deprecated"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("kernels").AtSliceIndex(0).AtMapKey("kvm"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("kernels").AtSliceIndex(0).AtMapKey("label"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("kernels").AtSliceIndex(0).AtMapKey("pvops"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("kernels").AtSliceIndex(0).AtMapKey("version"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("kernels").AtSliceIndex(0).AtMapKey("xen"), knownvalue.NotNull()),
				},
			},
			{
				Config: tmpl.DataFilter(t, kernelID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("kernels"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("kernels").AtSliceIndex(0).AtMapKey("id"), knownvalue.StringExact(kernelID)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("kernels").AtSliceIndex(0).AtMapKey("architecture"), knownvalue.StringExact("x86_64")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("kernels").AtSliceIndex(0).AtMapKey("deprecated"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("kernels").AtSliceIndex(0).AtMapKey("kvm"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("kernels").AtSliceIndex(0).AtMapKey("pvops"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("kernels").AtSliceIndex(0).AtMapKey("label"), knownvalue.StringRegexp(regexp.MustCompile("Latest 64 bit"))),
				},
			},
			{
				Config: tmpl.DataFilterEmpty(t, kernelID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("kernels"), knownvalue.ListSizeExact(0)),
				},
			},
		},
	})
}
