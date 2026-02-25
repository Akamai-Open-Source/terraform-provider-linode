//go:build integration || nbconfig

package nbconfig_test

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/linodego"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/helper"
	"github.com/linode/terraform-provider-linode/v3/linode/nbconfig"
	"github.com/linode/terraform-provider-linode/v3/linode/nbconfig/tmpl"
)

var testRegion string

func init() {
	region, err := acceptance.GetRandomRegionWithCaps([]string{"nodebalancers"}, "core")
	if err != nil {
		log.Fatal(err)
	}

	testRegion = region
}

var stateCheckNodeBalancerConfigExists statecheck.StateCheck = acceptance.CustomStateCheck(
	func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client

		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Type != "linode_nodebalancer_config" {
				continue
			}

			idVal, ok := rc.AttributeValues["id"]
			if !ok {
				resp.Error = fmt.Errorf("No ID is set")
				return
			}

			id, err := strconv.Atoi(idVal.(string))
			if err != nil {
				resp.Error = fmt.Errorf("Error parsing %v to int", idVal)
				return
			}

			nbIDVal, ok := rc.AttributeValues["nodebalancer_id"]
			if !ok {
				resp.Error = fmt.Errorf("nodebalancer_id not set")
				return
			}

			// nodebalancer_id is Int64Attribute — JSON state represents as float64
			nbIDFloat, ok := nbIDVal.(float64)
			if !ok {
				resp.Error = fmt.Errorf("expected nodebalancer_id to be float64, got %T", nbIDVal)
				return
			}
			nodebalancerID := int(nbIDFloat)

			_, err = client.GetNodeBalancerConfig(context.Background(), nodebalancerID, id)
			if err != nil {
				resp.Error = fmt.Errorf("Error retrieving state of NodeBalancer Config: %s", err)
				return
			}
		}
	},
)

func TestAccResourceNodeBalancerConfig_basic(t *testing.T) {
	t.Parallel()

	resName := "linode_nodebalancer_config.foofig"
	nodebalancerName := acctest.RandomWithPrefix("tf_test")
	resource.Test(t, resource.TestCase{
		PreventPostDestroyRefresh: true,
		PreCheck:                  func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories:  acceptance.ProtoV6ProviderFactories,
		CheckDestroy:              checkNodeBalancerConfigDestroy,
		ExternalProviders:         acceptance.HttpExternalProviders,
		Steps: []resource.TestStep{
			{
				Config:       tmpl.Basic(t, nodebalancerName, testRegion),
				ResourceName: resName,
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
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("ssl_cert"), knownvalue.Null()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("ssl_key"), knownvalue.Null()),
				},
			},
			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: resourceImportStateID,
			},
		},
	})
}

func TestAccResourceNodeBalancerConfig_ssl(t *testing.T) {
	t.Parallel()

	resName := "linode_nodebalancer_config.foofig"
	nodebalancerName := acctest.RandomWithPrefix("tf_test")
	resource.Test(t, resource.TestCase{
		PreventPostDestroyRefresh: true,
		PreCheck:                  func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories:  acceptance.ProtoV6ProviderFactories,
		CheckDestroy:              checkNodeBalancerConfigDestroy,
		ExternalProviders:         acceptance.HttpExternalProviders,
		Steps: []resource.TestStep{
			{
				Config:       tmpl.SSL(t, nodebalancerName, testRegion, tmpl.TestCertifcate, tmpl.TestPrivateKey),
				ResourceName: resName,
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodeBalancerConfigExists,
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("protocol"), knownvalue.StringExact(string(linodego.ProtocolHTTPS))),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("ssl_cert"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("ssl_key"), knownvalue.NotNull()),
				},
			},
			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"ssl_cert", "ssl_key"},
				ImportStateIdFunc:       resourceImportStateID,
			},
		},
	})
}

func TestAccResourceNodeBalancerConfig_update(t *testing.T) {
	t.Parallel()

	resName := "linode_nodebalancer_config.foofig"
	nodebalancerName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkNodeBalancerConfigDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, nodebalancerName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodeBalancerConfigExists,
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("port"), knownvalue.Int64Exact(8080)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("protocol"), knownvalue.StringExact(string(linodego.ProtocolHTTP))),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("check"), knownvalue.StringExact(string(linodego.CheckHTTP))),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("check_path"), knownvalue.StringExact("/")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("check_passive"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("stickiness"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("check_attempts"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("check_timeout"), knownvalue.NotNull()),
				},
			},
			{
				Config: tmpl.Updates(t, nodebalancerName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodeBalancerConfigExists,
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("port"), knownvalue.Int64Exact(8088)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("protocol"), knownvalue.StringExact(string(linodego.ProtocolHTTP))),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("check"), knownvalue.StringExact(string(linodego.CheckHTTP))),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("check_path"), knownvalue.StringExact("/foo")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("check_attempts"), knownvalue.Int64Exact(3)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("check_timeout"), knownvalue.Int64Exact(30)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("check_interval"), knownvalue.Int64Exact(31)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("udp_check_port"), knownvalue.Int64Exact(1234)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("check_passive"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("stickiness"), knownvalue.StringExact(string(linodego.StickinessHTTPCookie))),
				},
			},
		},
	})
}

func TestAccResourceNodeBalancerConfig_proxyProtocol(t *testing.T) {
	t.Parallel()

	resName := "linode_nodebalancer_config.foofig"
	nodebalancerName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkNodeBalancerConfigDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.ProxyProtocol(t, nodebalancerName, testRegion),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodeBalancerConfigExists,
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("port"), knownvalue.Int64Exact(80)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("protocol"), knownvalue.StringExact(string(linodego.ProtocolTCP))),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("proxy_protocol"), knownvalue.StringExact(string(linodego.ProxyProtocolV2))),
				},
			},
		},
	})
}

func TestLinodeNodeBalancerConfig_UpgradeV0(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	desiredDown := 13
	desiredUp := 37

	oldNodesStatus := map[string]attr.Value{
		"down": types.StringValue(strconv.Itoa(desiredDown)),
		"up":   types.StringValue(strconv.Itoa(desiredUp)),
	}

	oldNodesStatusMapValue, diags := types.MapValueFrom(
		ctx, nbconfig.NodeStatusTypeV0, oldNodesStatus,
	)
	if diags.HasError() {
		t.Fatal(diags.Errors())
	}

	oldData := nbconfig.ResourceModelV0{
		NodesStatus: oldNodesStatusMapValue,
	}
	var newData nbconfig.ResourceModelV1

	diags = newData.UpgradeFromV0(ctx, oldData)
	if diags.HasError() {
		t.Fatal(diags.Errors())
	}

	newDown := newData.NodesStatus.Elements()[0].(types.Object).Attributes()["down"].(types.Int64).ValueInt64()
	newUp := newData.NodesStatus.Elements()[0].(types.Object).Attributes()["up"].(types.Int64).ValueInt64()

	if !(newDown == int64(desiredDown)) {
		t.Fatalf("expected %v, got %v", desiredDown, desiredDown)
	}

	if !(newUp == int64(desiredUp)) {
		t.Fatalf("expected %v, got %v", desiredUp, desiredUp)
	}
}

func TestLinodeNodeBalancerConfig_UpgradeV0Empty(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	desiredDown := 0
	desiredUp := 0

	oldNodesStatus := map[string]attr.Value{
		"down": types.StringValue(""),
		"up":   types.StringValue(""),
	}

	oldNodesStatusMapValue, diags := types.MapValueFrom(
		ctx, nbconfig.NodeStatusTypeV0, oldNodesStatus,
	)
	if diags.HasError() {
		t.Fatal(diags.Errors())
	}

	oldData := nbconfig.ResourceModelV0{
		NodesStatus: oldNodesStatusMapValue,
	}
	var newData nbconfig.ResourceModelV1

	diags = newData.UpgradeFromV0(ctx, oldData)
	if diags.HasError() {
		t.Fatal(diags.Errors())
	}

	newDown := newData.NodesStatus.Elements()[0].(types.Object).Attributes()["down"].(types.Int64).ValueInt64()
	newUp := newData.NodesStatus.Elements()[0].(types.Object).Attributes()["up"].(types.Int64).ValueInt64()

	if !(newDown == int64(desiredDown)) {
		t.Fatalf("expected %v, got %v", desiredDown, desiredDown)
	}

	if newUp != int64(desiredUp) {
		t.Fatalf("expected %v, got %v", desiredUp, desiredUp)
	}
}
func checkNodeBalancerConfigDestroy(s *terraform.State) error {
	client := acceptance.TestAccSDKv2Provider.Meta().(*helper.ProviderMeta).Client
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_nodebalancer_config" {
			continue
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error parsing %v to int", rs.Primary.ID)
		}
		nodebalancerID, err := strconv.Atoi(rs.Primary.Attributes["nodebalancer_id"])
		if err != nil {
			return fmt.Errorf("Error parsing %v to int", rs.Primary.Attributes["nodebalancer_id"])
		}

		if id == 0 {
			return fmt.Errorf("Would have considered %v as %d", rs.Primary.ID, id)
		}

		_, err = client.GetNodeBalancerConfig(context.Background(), nodebalancerID, id)

		if err == nil {
			return fmt.Errorf("NodeBalancer Config with id %d still exists", id)
		}

		if apiErr, ok := err.(*linodego.Error); ok && apiErr.Code != 404 {
			return fmt.Errorf("Error requesting NodeBalancer Config with id %d", id)
		}
	}

	return nil
}

func resourceImportStateID(s *terraform.State) (string, error) {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_nodebalancer_config" {
			continue
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return "", fmt.Errorf("Error parsing ID %v to int", rs.Primary.ID)
		}
		nodebalancerID, err := strconv.Atoi(rs.Primary.Attributes["nodebalancer_id"])
		if err != nil {
			return "", fmt.Errorf("Error parsing nodebalancer_id %v to int", rs.Primary.Attributes["nodebalancer_id"])
		}
		return fmt.Sprintf("%d,%d", nodebalancerID, id), nil
	}

	return "", fmt.Errorf("Error finding linode_nodebalancer_config")
}
