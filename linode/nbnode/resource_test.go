//go:build integration || nbnode

package nbnode_test

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/linodego"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/nbnode/tmpl"
)

var testRegion string

func init() {
	region, err := acceptance.GetRandomRegionWithCaps([]string{"nodebalancers"}, "core")
	if err != nil {
		log.Fatal(err)
	}

	testRegion = region
}

func TestAccResourceNodeBalancerNode_basic(t *testing.T) {
	t.Parallel()

	resName := "linode_nodebalancer_node.foonode"
	nodeName := acctest.RandomWithPrefix("tf_test")
	config := tmpl.Basic(t, nodeName, testRegion, acctest.RandString(64))

	resource.Test(t, resource.TestCase{
		PreventPostDestroyRefresh: true,
		PreCheck:                  func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories:  acceptance.ProtoV6ProviderFactories,
		CheckDestroy:              checkNodeBalancerNodeDestroy,
		ExternalProviders:         acceptance.HttpExternalProviders,
		Steps: []resource.TestStep{
			{
				Config: config,
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodeBalancerNodeExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(nodeName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("status"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("mode"), knownvalue.StringExact("accept")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("weight"), knownvalue.Int64Exact(50)),
				},
			},
			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: importResourceStateID,
			},
		},
	})
}

func TestAccResourceNodeBalancerNode_update(t *testing.T) {
	t.Parallel()

	resName := "linode_nodebalancer_node.foonode"
	nodeName := acctest.RandomWithPrefix("tf_test")
	rootPass := acctest.RandString(64)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkNodeBalancerNodeDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, nodeName, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodeBalancerNodeExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(nodeName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("weight"), knownvalue.Int64Exact(50)),
				},
			},
			{
				Config: tmpl.Updates(t, nodeName, testRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckNodeBalancerNodeExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(fmt.Sprintf("%s_r", nodeName))),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("weight"), knownvalue.Int64Exact(200)),
				},
			},
			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: importResourceStateID,
			},
		},
	})
}

func TestAccResourceNodeBalancerNode_vpc(t *testing.T) {
	t.Parallel()

	resName := "linode_nodebalancer_node.test"
	label := acctest.RandomWithPrefix("tf-test")
	rootPass := acctest.RandString(64)

	targetRegion, err := acceptance.GetRandomRegionWithCaps([]string{"NodeBalancers", "VPCs"}, "core")
	if err != nil {
		log.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		CheckDestroy:             checkNodeBalancerNodeDestroy,

		Steps: []resource.TestStep{
			{
				Config: tmpl.VPC(t, label, targetRegion, rootPass),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("nodebalancer_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("config_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("subnet_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("address"),
						knownvalue.StringExact("10.0.0.5:80"),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("label"),
						knownvalue.StringExact(label),
					),
					statecheck.ExpectKnownValue(
						resName,
						tfjsonpath.New("weight"),
						knownvalue.Int64Exact(50),
					),
				},
			},
			{
				Config: tmpl.VPC(t, label, targetRegion, rootPass),
			},
			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateIdFunc:       importResourceStateID,
				ImportStateVerifyIgnore: []string{"status"},
			},
		},
	})
}

func stateCheckNodeBalancerNodeExists() statecheck.StateCheck {
	return acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		client, err := acceptance.GetTestClient()
		if err != nil {
			resp.Error = fmt.Errorf("failed to get client: %s", err)
			return
		}

		var linodeID, nodebalancerID, nodeID, configID int
		var expectedNodePort string

		// find Linode instance ID
		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Type != "linode_instance" {
				continue
			}

			idStr, ok := rc.AttributeValues["id"].(string)
			if !ok {
				resp.Error = fmt.Errorf("Error getting instance ID")
				return
			}
			linodeID, err = strconv.Atoi(idStr)
			if err != nil {
				resp.Error = fmt.Errorf("Error parsing %v to int", idStr)
				return
			}
		}

		// find NodeBalancer Node ID
		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Type != "linode_nodebalancer_node" {
				continue
			}

			idStr, ok := rc.AttributeValues["id"].(string)
			if !ok {
				resp.Error = fmt.Errorf("Error getting node ID")
				return
			}
			nodeID, err = strconv.Atoi(idStr)
			if err != nil {
				resp.Error = fmt.Errorf("Error parsing %v to int", idStr)
				return
			}

			// nodebalancer_id is Int64 in schema, comes as float64 in tfjson state
			nbIDVal, ok := rc.AttributeValues["nodebalancer_id"]
			if !ok {
				resp.Error = fmt.Errorf("nodebalancer_id not found")
				return
			}
			nbIDFloat, ok := nbIDVal.(float64)
			if !ok {
				resp.Error = fmt.Errorf("Error parsing nodebalancer_id to float64")
				return
			}
			nodebalancerID = int(nbIDFloat)

			// config_id is Int64 in schema, comes as float64 in tfjson state
			configIDVal, ok := rc.AttributeValues["config_id"]
			if !ok {
				resp.Error = fmt.Errorf("config_id not found")
				return
			}
			configIDFloat, ok := configIDVal.(float64)
			if !ok {
				resp.Error = fmt.Errorf("Error parsing config_id to float64")
				return
			}
			configID = int(configIDFloat)

			// address is a string attribute
			address, ok := rc.AttributeValues["address"].(string)
			if !ok {
				resp.Error = fmt.Errorf("Error getting address")
				return
			}
			expectedNodePort = strings.Split(address, ":")[1]
		}

		instanceNetwork, err := client.GetInstanceIPAddresses(context.Background(), linodeID)
		if err != nil {
			resp.Error = fmt.Errorf("failed to get IPs for instance %d: %s", linodeID, err)
			return
		}

		node, err := client.GetNodeBalancerNode(context.Background(), nodebalancerID, configID, nodeID)
		if err != nil {
			resp.Error = fmt.Errorf("Error retrieving state of NodeBalancer Node %d: %s", nodeID, err)
			return
		}

		privateIP := instanceNetwork.IPv4.Private[0].Address

		nodeAddrComps := strings.Split(node.Address, ":")
		nodeHost, nodePort := nodeAddrComps[0], nodeAddrComps[1]

		if nodeHost != privateIP {
			resp.Error = fmt.Errorf("expected node to have host '%s'; got '%s'", privateIP, node.Address)
			return
		}

		if nodePort != expectedNodePort {
			resp.Error = fmt.Errorf("expected node to have port '%s'; got '%s'", expectedNodePort, nodePort)
		}
	})
}

func checkNodeBalancerNodeExists(s *terraform.State) (err error) {
	client, err := acceptance.GetTestClient()
	if err != nil {
		log.Fatalf("failed to get client: %s", err)
	}

	var linodeID, nodebalancerID, nodeID, configID int
	var expectedNodePort string

	// find Linode instance ID
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_instance" {
			continue
		}

		linodeID, err = strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error parsing %v to int", rs.Primary.ID)
		}
	}

	// find NodeBalancer Node ID
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_nodebalancer_node" {
			continue
		}

		nodeID, err = strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error parsing %v to int", rs.Primary.ID)
		}

		nodebalancerID, err = strconv.Atoi(rs.Primary.Attributes["nodebalancer_id"])
		if err != nil {
			return fmt.Errorf("Error parsing %v to int", rs.Primary.Attributes["nodebalancer_id"])
		}

		configID, err = strconv.Atoi(rs.Primary.Attributes["config_id"])
		if err != nil {
			return fmt.Errorf("Error parsing %v to int", rs.Primary.Attributes["config_id"])
		}

		expectedNodePort = strings.Split(rs.Primary.Attributes["address"], ":")[1]
	}

	instanceNetwork, err := client.GetInstanceIPAddresses(context.Background(), linodeID)
	if err != nil {
		return fmt.Errorf("failed to get IPs for instance %d: %s", linodeID, err)
	}

	node, err := client.GetNodeBalancerNode(context.Background(), nodebalancerID, configID, nodeID)
	if err != nil {
		return fmt.Errorf("Error retrieving state of NodeBalancer Node %d: %s", nodeID, err)
	}

	privateIP := instanceNetwork.IPv4.Private[0].Address

	nodeAddrComps := strings.Split(node.Address, ":")
	nodeHost, nodePort := nodeAddrComps[0], nodeAddrComps[1]

	if nodeHost != privateIP {
		return fmt.Errorf("expected node to have host '%s'; got '%s'", privateIP, node.Address)
	}

	if nodePort != expectedNodePort {
		return fmt.Errorf("expected node to have port '%s'; got '%s'", expectedNodePort, nodePort)
	}
	return err
}

func checkNodeBalancerNodeDestroy(s *terraform.State) error {
	client, err := acceptance.GetTestClient()
	if err != nil {
		log.Fatalf("failed to get client: %s", err)
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_nodebalancer_node" {
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

		configID, err := strconv.Atoi(rs.Primary.Attributes["config_id"])
		if err != nil {
			return fmt.Errorf("Error parsing %v to int", rs.Primary.Attributes["config_id"])
		}

		if id == 0 {
			return fmt.Errorf("Would have considered %v as %d", rs.Primary.ID, id)
		}

		_, err = client.GetNodeBalancerNode(context.Background(), nodebalancerID, configID, id)

		if err == nil {
			return fmt.Errorf("NodeBalancer Node with id %d still exists", id)
		}

		if apiErr, ok := err.(*linodego.Error); ok && apiErr.Code != 404 {
			return fmt.Errorf("Error requesting NodeBalancer Node with id %d", id)
		}
	}

	return nil
}

func importResourceStateID(s *terraform.State) (string, error) {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_nodebalancer_node" {
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
		configID, err := strconv.Atoi(rs.Primary.Attributes["config_id"])
		if err != nil {
			return "", fmt.Errorf("Error parsing config_id %v to int", rs.Primary.Attributes["config_id"])
		}
		return fmt.Sprintf("%d,%d,%d", nodebalancerID, configID, id), nil
	}

	return "", fmt.Errorf("Error finding linode_nodebalancer_config")
}
