package acceptance

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
	"text/template"
	"time"

	tfjson "github.com/hashicorp/terraform-json"
	fwDiag "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/linode/linodego"
	"github.com/linode/terraform-provider-linode/v3/linode"
	"github.com/linode/terraform-provider-linode/v3/linode/helper"
	"github.com/linode/terraform-provider-linode/v3/version"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

const (
	optInTestsEnvVar         = "ACC_OPT_IN_TESTS"
	SkipInstanceReadyPollKey = "skip_instance_ready_poll"

	runLongTestsEnvVar  = "RUN_LONG_TESTS"
	skipLongTestMessage = "This test has been marked as a long-running test and is skipped by default. " +
		"If you would like to run this test, please set the RUN_LONG_TEST environment variable to true."
)

type (
	AttrValidateFunc     func(val string) error
	ListAttrValidateFunc func(resourceName, path string, state *terraform.State) error
	RegionFilterFunc     func(v linodego.Region) bool

	// StateListAttrValidateFunc is the StateCheck equivalent of ListAttrValidateFunc.
	// It accepts a resource name, path, and the list of tfjson.StateResource entries
	// from the root module for validation against the tfjson state format.
	StateListAttrValidateFunc func(resourceName, path string, resources []*tfjson.StateResource) error
)

var (
	optInTests               map[string]struct{}
	privateKeyMaterial       string
	PublicKeyMaterial        string
	TestAccSDKv2Providers    map[string]*schema.Provider
	TestAccSDKv2Provider     *schema.Provider
	TestAccFrameworkProvider *linode.FrameworkProvider
	ConfigTemplates          *template.Template
	TestImageLatest          string
	TestImagePrevious        string
)

func initOptInTests() {
	optInTests = make(map[string]struct{})

	optInTestsValue, ok := os.LookupEnv(optInTestsEnvVar)
	if !ok {
		return
	}

	for testName := range strings.SplitSeq(optInTestsValue, ",") {
		optInTests[testName] = struct{}{}
	}
}

// initTestImages grabs the latest Linode Alpine images for acceptance test configurations
func initTestImages() {
	client, err := GetTestClient()
	if err != nil {
		log.Fatalf("failed to get client: %s", err)
	}

	imageFilter := &linodego.Filter{}

	imageFilter.AddField(linodego.Eq, "vendor", "Alpine")

	filterJSON, err := imageFilter.MarshalJSON()
	if err != nil {
		log.Fatalf("failed to create image filter json: %s", err)
	}

	images, err := client.ListImages(context.Background(), &linodego.ListOptions{Filter: string(filterJSON)})
	if err != nil {
		log.Fatal(err)
	}

	sort.SliceStable(images, func(i, j int) bool {
		return images[i].Created.After(*images[j].Created)
	})

	TestImageLatest = images[0].ID
	TestImagePrevious = images[1].ID
}

func init() {
	var err error
	PublicKeyMaterial, privateKeyMaterial, err = acctest.RandSSHKeyPair("linode@ssh-acceptance-test")
	if err != nil {
		log.Fatalf("Failed to generate random SSH key pair for testing: %s", err)
	}

	initOptInTests()

	TestAccSDKv2Provider = linode.Provider()
	TestAccFrameworkProvider = linode.CreateFrameworkProvider(version.ProviderVersion).(*linode.FrameworkProvider)
	TestAccSDKv2Providers = map[string]*schema.Provider{
		"linode": TestAccSDKv2Provider,
	}

	var templateFiles []string

	err = filepath.Walk(
		"../",
		func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}

			if filepath.Ext(path) == ".gotf" {
				templateFiles = append(templateFiles, path)
			}

			return nil
		},
	)
	if err != nil {
		log.Fatalf("failed to load template files: %v", err)
	}

	ConfigTemplates = template.New("tf-test")
	if _, err := ConfigTemplates.ParseFiles(templateFiles...); err != nil {
		log.Fatalf("failed to parse templates: %v", err)
	}

	initTestImages()
}

func TestProvider(t *testing.T) {
	t.Parallel()

	if err := linode.Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func PreCheck(t testing.TB) {
	t.Helper()

	if v := os.Getenv("LINODE_TOKEN"); v == "" {
		t.Fatal("LINODE_TOKEN must be set for acceptance tests")
	}
}

func OptInTest(t testing.TB) {
	t.Helper()

	if _, ok := optInTests[t.Name()]; !ok {
		t.Skipf("skipping opt-in test; specify test in environment variable %q to run", optInTestsEnvVar)
	}
}

func LongRunningTest(t testing.TB) {
	t.Helper()

	shouldRunStr := os.Getenv(runLongTestsEnvVar)
	if len(shouldRunStr) == 0 {
		t.Skip(skipLongTestMessage)
	}

	shouldRun, err := strconv.ParseBool(shouldRunStr)
	if err != nil {
		t.Fatalf("failed to parse %s as bool: %s", runLongTestsEnvVar, err)
	}

	if !shouldRun {
		t.Skip(skipLongTestMessage)
	}
}

func GetSSHClient(t testing.TB, user, addr string) (client *ssh.Client) {
	t.Helper()

	signer, err := ssh.ParsePrivateKey([]byte(privateKeyMaterial))
	if err != nil {
		t.Fatalf("failed to parse private key: %s", err)
	}

	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
		// #nosec G106 -- Test data, not used in production
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Minute,
	}

	attempts := 3

	for attempts != 0 {
		client, err = ssh.Dial("tcp", addr+":22", config)
		if err == nil {
			break
		}

		t.Logf("ssh dial failed: %s", err)
		attempts--

		time.Sleep(5 * time.Second)
	}

	if client == nil {
		t.Fatal("failed to get ssh client")
	}
	return client
}

func CheckResourceAttrContains(resName string, path, desiredValue string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resName]
		if !ok {
			return fmt.Errorf("Not found: %s", resName)
		}

		value, ok := rs.Primary.Attributes[path]
		if !ok {
			return fmt.Errorf("attribute %s does not exist", path)
		}

		if !strings.Contains(value, desiredValue) {
			return fmt.Errorf("value '%s' was not found", desiredValue)
		}

		return nil
	}
}

// StateCheckResourceAttrContains is the StateCheck equivalent of CheckResourceAttrContains.
// It checks that a resource attribute contains the specified substring.
func StateCheckResourceAttrContains(resName string, path, desiredValue string) statecheck.StateCheck {
	return CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Address != resName {
				continue
			}

			value, ok := rc.AttributeValues[path]
			if !ok {
				resp.Error = fmt.Errorf("attribute %s does not exist", path)
				return
			}

			strValue, ok := value.(string)
			if !ok {
				resp.Error = fmt.Errorf("attribute %s is not a string", path)
				return
			}

			if !strings.Contains(strValue, desiredValue) {
				resp.Error = fmt.Errorf("value '%s' was not found in '%s'", desiredValue, strValue)
			}
			return
		}
		resp.Error = fmt.Errorf("Not found: %s", resName)
	})
}

func ValidateResourceAttr(resName, path string, comparisonFunc AttrValidateFunc) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resName]
		if !ok {
			return fmt.Errorf("Not found: %s", resName)
		}

		value, ok := rs.Primary.Attributes[path]
		if !ok {
			return fmt.Errorf("attribute %s does not exist", path)
		}

		err := comparisonFunc(value)
		if err != nil {
			return fmt.Errorf("comparison failed: %s", err)
		}

		return nil
	}
}

func CheckResourceAttrGreaterThan(resName, path string, target int) resource.TestCheckFunc {
	return ValidateResourceAttr(resName, path, func(val string) error {
		valInt, err := strconv.Atoi(val)
		if err != nil {
			return err
		}

		if valInt <= target {
			return fmt.Errorf("%d <= %d", valInt, target)
		}

		return nil
	})
}

// StateCheckResourceAttrGreaterThan is the StateCheck equivalent of CheckResourceAttrGreaterThan.
// It validates that a resource attribute's integer value is strictly greater than the target.
// Handles string, float64, and json.Number representations from the tfjson state.
// For flatmap-style ".#" suffix paths (e.g., "items.#"), it automatically resolves
// the base attribute as a list/set and compares its length against the target.
func StateCheckResourceAttrGreaterThan(resName, path string, target int) statecheck.StateCheck {
	return CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Address != resName {
				continue
			}

			// Handle flatmap-style ".#" suffix for list/set length checks.
			// In tfjson state, list/set attributes are stored as []interface{}
			// under their base attribute name, not with ".#" count suffixes.
			if strings.HasSuffix(path, ".#") {
				basePath := strings.TrimSuffix(path, ".#")
				listVal, ok := rc.AttributeValues[basePath]
				if !ok {
					resp.Error = fmt.Errorf("attribute %s does not exist", basePath)
					return
				}

				list, ok := listVal.([]interface{})
				if !ok {
					resp.Error = fmt.Errorf("attribute %s is not a list/set (got %T)", basePath, listVal)
					return
				}

				if len(list) <= target {
					resp.Error = fmt.Errorf("%d <= %d", len(list), target)
				}
				return
			}

			value, ok := rc.AttributeValues[path]
			if !ok {
				resp.Error = fmt.Errorf("attribute %s does not exist", path)
				return
			}

			// tfjson state may represent numbers as json.Number or float64
			var valInt int
			switch v := value.(type) {
			case string:
				var err error
				valInt, err = strconv.Atoi(v)
				if err != nil {
					resp.Error = fmt.Errorf("error parsing %v to int: %s", v, err)
					return
				}
			case float64:
				valInt = int(v)
			case json.Number:
				i, err := v.Int64()
				if err != nil {
					resp.Error = fmt.Errorf("error parsing %v to int: %s", v, err)
					return
				}
				valInt = int(i)
			default:
				resp.Error = fmt.Errorf("attribute %s has unexpected type %T", path, value)
				return
			}

			if valInt <= target {
				resp.Error = fmt.Errorf("%d <= %d", valInt, target)
			}
			return
		}
		resp.Error = fmt.Errorf("Not found: %s", resName)
	})
}

func CheckResourceAttrNotEqual(resName string, path, notValue string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resName]
		if !ok {
			return fmt.Errorf("Not found: %s", resName)
		}

		if value, ok := rs.Primary.Attributes[path]; !ok {
			return fmt.Errorf("attribute %s does not exist", path)
		} else if value == notValue {
			return fmt.Errorf("attribute was equal")
		}

		return nil
	}
}

// StateCheckResourceAttrNotEqual is the StateCheck equivalent of CheckResourceAttrNotEqual.
// It validates that a resource attribute does not equal the specified value.
func StateCheckResourceAttrNotEqual(resName string, path, notValue string) statecheck.StateCheck {
	return CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Address != resName {
				continue
			}

			value, ok := rc.AttributeValues[path]
			if !ok {
				resp.Error = fmt.Errorf("attribute %s does not exist", path)
				return
			}

			strValue := fmt.Sprintf("%v", value)
			if strValue == notValue {
				resp.Error = fmt.Errorf("attribute was equal")
			}
			return
		}
		resp.Error = fmt.Errorf("Not found: %s", resName)
	})
}

func CheckResourceAttrListContains(resName, path, desiredValue string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resName]
		if !ok {
			return fmt.Errorf("Not found: %s", resName)
		}

		length, err := strconv.Atoi(rs.Primary.Attributes[path+".#"])
		if err != nil {
			return fmt.Errorf("attribute %s does not exist", path)
		}

		for i := range length {
			if rs.Primary.Attributes[path+"."+strconv.Itoa(i)] == desiredValue {
				return nil
			}
		}

		return fmt.Errorf("Desired value not found in resource attribute")
	}
}

// StateCheckResourceAttrListContains is the StateCheck equivalent of CheckResourceAttrListContains.
// It validates that a resource attribute list contains the specified value.
func StateCheckResourceAttrListContains(resName, path, desiredValue string) statecheck.StateCheck {
	return CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Address != resName {
				continue
			}

			listVal, ok := rc.AttributeValues[path]
			if !ok {
				resp.Error = fmt.Errorf("attribute %s does not exist", path)
				return
			}

			list, ok := listVal.([]interface{})
			if !ok {
				resp.Error = fmt.Errorf("attribute %s is not a list", path)
				return
			}

			for _, item := range list {
				if fmt.Sprintf("%v", item) == desiredValue {
					return
				}
			}

			resp.Error = fmt.Errorf("Desired value %s not found in resource attribute %s", desiredValue, path)
			return
		}
		resp.Error = fmt.Errorf("Not found: %s", resName)
	})
}

func LoopThroughStringList(resName, path string, listValidateFunc ListAttrValidateFunc) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resName]
		if !ok {
			return fmt.Errorf("Not found: %s", resName)
		}

		length, err := strconv.Atoi(rs.Primary.Attributes[path+".#"])
		if err != nil {
			return fmt.Errorf("attribute %s does not exist", path)
		}

		for i := range length {
			err := listValidateFunc(resName, path+"."+strconv.Itoa(i), s)
			if err != nil {
				return fmt.Errorf("Value not found:%s", err)
			}
		}

		return nil
	}
}

// StateCheckLoopThroughStringList is the StateCheck equivalent of LoopThroughStringList.
// It iterates through a list attribute and runs a validation function on each element.
// The callback receives the resource name, element path, and the tfjson state resources.
func StateCheckLoopThroughStringList(resName, path string, listValidateFunc StateListAttrValidateFunc) statecheck.StateCheck {
	return CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Address != resName {
				continue
			}

			listVal, ok := rc.AttributeValues[path]
			if !ok {
				resp.Error = fmt.Errorf("attribute %s does not exist", path)
				return
			}

			list, ok := listVal.([]interface{})
			if !ok {
				resp.Error = fmt.Errorf("attribute %s is not a list", path)
				return
			}

			for i := range list {
				elemPath := path + "." + strconv.Itoa(i)
				err := listValidateFunc(resName, elemPath, req.State.Values.RootModule.Resources)
				if err != nil {
					resp.Error = fmt.Errorf("Value not found: %s", err)
					return
				}
			}
			return
		}
		resp.Error = fmt.Errorf("Not found: %s", resName)
	})
}

// CheckListContains checks whether a state list or set contains a given value
func CheckListContains(resName, path, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resName]
		if !ok {
			return fmt.Errorf("Not found: %s", resName)
		}

		length, err := strconv.Atoi(rs.Primary.Attributes[path+".#"])
		if err != nil {
			return fmt.Errorf("attribute %s does not exist", path)
		}

		for i := range length {
			foundValue, ok := rs.Primary.Attributes[path+"."+strconv.Itoa(i)]
			if !ok {
				return fmt.Errorf("index %d does not exist in attributes", i)
			}

			if foundValue == value {
				return nil
			}
		}

		return fmt.Errorf("failed to find value %s in %s", value, path)
	}
}

// StateCheckListContains is the StateCheck equivalent of CheckListContains.
// It checks whether a state list or set attribute contains a given value.
func StateCheckListContains(resName, path, value string) statecheck.StateCheck {
	return CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Address != resName {
				continue
			}

			listVal, ok := rc.AttributeValues[path]
			if !ok {
				resp.Error = fmt.Errorf("attribute %s does not exist", path)
				return
			}

			list, ok := listVal.([]interface{})
			if !ok {
				resp.Error = fmt.Errorf("attribute %s is not a list", path)
				return
			}

			for _, item := range list {
				if fmt.Sprintf("%v", item) == value {
					return
				}
			}

			resp.Error = fmt.Errorf("failed to find value %s in %s", value, path)
			return
		}
		resp.Error = fmt.Errorf("Not found: %s", resName)
	})
}

func CheckLKEClusterDestroy(s *terraform.State) error {
	client, err := GetTestClient()
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_lke_cluster" {
			continue
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("failed to parse LKE Cluster ID: %s", err)
		}

		if id == 0 {
			return fmt.Errorf("should not have LKE Cluster ID of 0")
		}

		if _, err = client.GetLKECluster(context.Background(), id); err == nil {
			return fmt.Errorf("should not find Linode ID %d existing after delete", id)
		} else if apiErr, ok := err.(*linodego.Error); !ok {
			return fmt.Errorf("expected API Error but got %#v", err)
		} else if apiErr.Code != 404 {
			return fmt.Errorf("expected an error 404 but got %#v", apiErr)
		}
	}

	return nil
}

func CheckVolumeDestroy(s *terraform.State) error {
	client, err := GetTestClient()
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_volume" {
			continue
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error parsing %v to int", rs.Primary.ID)
		}
		if id == 0 {
			return fmt.Errorf("Would have considered %v as %d", rs.Primary.ID, id)
		}

		_, err = client.GetVolume(context.Background(), id)

		if err == nil {
			return fmt.Errorf("Linode Volume with id %d still exists", id)
		}

		if apiErr, ok := err.(*linodego.Error); ok && apiErr.Code != 404 {
			return fmt.Errorf("Error requesting Linode Volume with id %d", id)
		}
	}

	return nil
}

func CheckVolumeExists(name string, volume *linodego.Volume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client, err := GetTestClient()
		if err != nil {
			return err
		}

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error parsing %v to int", rs.Primary.ID)
		}

		found, err := client.GetVolume(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Error retrieving state of Volume %s: %s", rs.Primary.Attributes["label"], err)
		}

		*volume = *found

		return nil
	}
}

// StateCheckVolumeExists is the StateCheck equivalent of CheckVolumeExists.
// It verifies that a Linode Volume resource exists by querying the Linode API
// using the resource ID from the tfjson state and populates the provided volume pointer.
func StateCheckVolumeExists(name string, volume *linodego.Volume) statecheck.StateCheck {
	return CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		client, err := GetTestClient()
		if err != nil {
			resp.Error = err
			return
		}

		var resourceID string
		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Address == name {
				idVal, ok := rc.AttributeValues["id"]
				if !ok {
					resp.Error = fmt.Errorf("No ID is set for %s", name)
					return
				}
				resourceID = fmt.Sprintf("%v", idVal)
				break
			}
		}

		if resourceID == "" {
			resp.Error = fmt.Errorf("Not found: %s", name)
			return
		}

		id, err := strconv.Atoi(resourceID)
		if err != nil {
			resp.Error = fmt.Errorf("Error parsing %v to int", resourceID)
			return
		}

		found, err := client.GetVolume(context.Background(), id)
		if err != nil {
			resp.Error = fmt.Errorf("Error retrieving state of Volume %s: %s", name, err)
			return
		}

		*volume = *found
	})
}

func CheckFirewallExists(name string, firewall *linodego.Firewall) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client, err := GetTestClient()
		if err != nil {
			return err
		}

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error parsing %v to int", rs.Primary.ID)
		}

		found, err := client.GetFirewall(context.Background(), id)
		if err != nil {
			return fmt.Errorf("Error retrieving state of Firewall %s: %s", rs.Primary.Attributes["label"], err)
		}

		*firewall = *found

		return nil
	}
}

// StateCheckFirewallExists is the StateCheck equivalent of CheckFirewallExists.
// It verifies that a Linode Firewall resource exists by querying the Linode API
// using the resource ID from the tfjson state and populates the provided firewall pointer.
func StateCheckFirewallExists(name string, firewall *linodego.Firewall) statecheck.StateCheck {
	return CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		client, err := GetTestClient()
		if err != nil {
			resp.Error = err
			return
		}

		var resourceID string
		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Address == name {
				idVal, ok := rc.AttributeValues["id"]
				if !ok {
					resp.Error = fmt.Errorf("No ID is set for %s", name)
					return
				}
				resourceID = fmt.Sprintf("%v", idVal)
				break
			}
		}

		if resourceID == "" {
			resp.Error = fmt.Errorf("Not found: %s", name)
			return
		}

		id, err := strconv.Atoi(resourceID)
		if err != nil {
			resp.Error = fmt.Errorf("Error parsing %v to int", resourceID)
			return
		}

		found, err := client.GetFirewall(context.Background(), id)
		if err != nil {
			resp.Error = fmt.Errorf("Error retrieving state of Firewall %s: %s", name, err)
			return
		}

		*firewall = *found
	})
}

func CheckEventAbsent(name string, entityType linodego.EntityType, action linodego.EventAction) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client, err := GetTestClient()
		if err != nil {
			return err
		}

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error parsing %v to int", rs.Primary.ID)
		}

		event, err := helper.GetLatestEvent(context.Background(), client, id, entityType, action)
		if err != nil {
			return err
		}

		if event != nil {
			return fmt.Errorf("event exists: %d", event.ID)
		}

		return nil
	}
}

// StateCheckEventAbsent is the StateCheck equivalent of CheckEventAbsent.
// It verifies that no event of the specified entity type and action exists
// for the resource by querying the Linode API.
func StateCheckEventAbsent(name string, entityType linodego.EntityType, action linodego.EventAction) statecheck.StateCheck {
	return CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		client, err := GetTestClient()
		if err != nil {
			resp.Error = err
			return
		}

		var resourceID string
		for _, rc := range req.State.Values.RootModule.Resources {
			if rc.Address == name {
				idVal, ok := rc.AttributeValues["id"]
				if !ok {
					resp.Error = fmt.Errorf("no ID is set for %s", name)
					return
				}
				resourceID = fmt.Sprintf("%v", idVal)
				break
			}
		}

		if resourceID == "" {
			resp.Error = fmt.Errorf("not found: %s", name)
			return
		}

		id, err := strconv.Atoi(resourceID)
		if err != nil {
			resp.Error = fmt.Errorf("error parsing %v to int", resourceID)
			return
		}

		event, err := helper.GetLatestEvent(context.Background(), client, id, entityType, action)
		if err != nil {
			resp.Error = err
			return
		}

		if event != nil {
			resp.Error = fmt.Errorf("event exists: %d", event.ID)
		}
	})
}

func AnyOfTestCheckFunc(funcs ...resource.TestCheckFunc) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		manyErrors := []error{}
		for _, f := range funcs {
			if err := f(s); err == nil {
				return err
			} else {
				manyErrors = append(manyErrors, err)
			}
		}
		return errors.Join(manyErrors...)
	}
}

func ExecuteTemplate(t testing.TB, templateName string, data any) string {
	t.Helper()

	var b bytes.Buffer

	err := ConfigTemplates.ExecuteTemplate(&b, templateName, data)
	if err != nil {
		t.Fatalf("failed to execute template %s: %v", templateName, err)
	}

	return b.String()
}

func CreateTempFile(t testing.TB, name, content string) *os.File {
	file, err := os.CreateTemp(os.TempDir(), name)
	if err != nil {
		t.Fatalf("failed to create temp file: %s", err)
	}

	t.Cleanup(func() {
		if err := os.Remove(file.Name()); err != nil { //#nosec G703
			t.Fatalf("failed to remove test file: %s", err)
		}
	})

	if _, err := file.WriteString(content); err != nil {
		t.Fatalf("failed to write to temp file: %s", err)
	}

	return file
}

func CreateTestProvider() (*schema.Provider, map[string]*schema.Provider) {
	provider := linode.Provider()
	providerMap := map[string]*schema.Provider{
		"linode": provider,
	}
	return provider, providerMap
}

type ProviderMetaModifier func(ctx context.Context, data *schema.ResourceData, config *helper.ProviderMeta) error

func ModifyProviderMeta(provider *schema.Provider, modifier ProviderMetaModifier) *schema.Provider {
	oldConfigure := provider.ConfigureContextFunc

	provider.ConfigureContextFunc = func(ctx context.Context, data *schema.ResourceData) (any, diag.Diagnostics) {
		config, err := oldConfigure(ctx, data)
		if err != nil {
			return nil, err
		}

		if err := modifier(ctx, data, config.(*helper.ProviderMeta)); err != nil {
			return nil, diag.FromErr(err)
		}

		return config, nil
	}

	return provider
}

func GetEndpointType(e linodego.ObjectStorageEndpoint) string {
	return string(e.EndpointType)
}

func GetEndpointRegion(e linodego.ObjectStorageEndpoint) string {
	return e.Region
}

func GetEndpointCluster(e linodego.ObjectStorageEndpoint) (string, error) {
	if e.S3Endpoint == nil {
		return "", fmt.Errorf(
			"the %q type endpoint is nil for region %q for the user",
			e.EndpointType, e.Region,
		)
	}

	endpointURL := *e.S3Endpoint
	splittedURL := strings.Split(endpointURL, ".")
	if len(splittedURL) == 0 {
		return "", fmt.Errorf("invalid s3 endpoint received: %v", splittedURL)
	}

	return strings.Split(endpointURL, ".")[0], nil
}

// Get an Object Storage services endpoint with non-nil S3Endpoint
func GetRandomObjectStorageEndpoint() (*linodego.ObjectStorageEndpoint, error) {
	client, err := GetTestClient()
	if err != nil {
		return nil, err
	}

	endpoints, err := client.ListObjectStorageEndpoints(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	rand.Shuffle(len(endpoints), func(i, j int) {
		endpoints[i], endpoints[j] = endpoints[j], endpoints[i]
	})

	for i, e := range endpoints {
		// Linode Object Storage clusters with E2 and E3 (Object Storage gen2) endpoints
		// doesn't support API call with only `cluster` rather than `region`.
		// Only selecting E1 (Object Storage gen1) here to make sure tests always pass.
		//
		// TODO:
		// Remove this condition when E1 is deprecated in the future
		// or test cases with `cluster` are removed.
		if e.S3Endpoint != nil && e.EndpointType == linodego.ObjectStorageEndpointE1 {
			result := endpoints[i]
			return &result, nil
		}
	}

	return nil, errors.New("failed to get an object storage endpoint")
}

// GetRegionsWithCaps returns a list of region IDs that support the given capabilities
// Parameters:
// - capabilities: Required capabilities that the regions must support.
// - siteType: The site type to filter by ("core" or "distributed" or "any").
// - filters: Optional custom filters for additional criteria.
func GetRegionsWithCaps(capabilities []string, regionType string, filters ...RegionFilterFunc) ([]string, error) {
	client, err := GetTestClient()
	if err != nil {
		return nil, err
	}

	regions, err := client.ListRegions(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	// Filter on capabilities and site type
	regionsWithCaps := slices.DeleteFunc(regions, func(region linodego.Region) bool {
		// Check if the site type matches
		// Skip site type check if "any" is passed
		if !strings.EqualFold(regionType, "any") && !strings.EqualFold(region.SiteType, regionType) {
			return true
		}

		capsMap := make(map[string]bool)

		for _, c := range region.Capabilities {
			capsMap[strings.ToUpper(c)] = true
		}

		for _, c := range capabilities {
			if _, ok := capsMap[strings.ToUpper(c)]; !ok {
				return true
			}
		}

		return false
	})

	// Apply test-supplied filters
	filteredRegions := slices.DeleteFunc(regionsWithCaps, func(region linodego.Region) bool {
		for _, filter := range filters {
			if !filter(region) {
				return true
			}
		}

		return false
	})

	return helper.MapSlice(filteredRegions, func(r linodego.Region) string {
		return r.ID
	}), nil
}

// GetRandomRegionWithCaps gets a random region given a list of region capabilities.
func GetRandomRegionWithCaps(capabilities []string, regionType string, filters ...RegionFilterFunc) (string, error) {
	regions, err := GetRegionsWithCaps(capabilities, regionType, filters...)
	if err != nil {
		return "", err
	}

	if len(regions) < 1 {
		return "", fmt.Errorf("no region found with the provided caps")
	}

	// #nosec G404 -- Test data, doesn't need to be cryptography
	return regions[rand.Intn(len(regions))], nil
}

// Deprecated: Cluster is now deprecated in favor of Region.
// GetRandomOBJCluster gets a random Object Storage cluster.
func GetRandomOBJCluster() (string, error) {
	client, err := GetTestClient()
	if err != nil {
		return "", err
	}

	clusters, err := client.ListObjectStorageClusters(context.Background(), nil)
	if err != nil {
		return "", err
	}

	if len(clusters) < 1 {
		return "", fmt.Errorf("no clusters found")
	}

	// #nosec G404 -- Test data, doesn't need to be cryptography
	return clusters[rand.Intn(len(clusters))].ID, nil
}

func GetTestClient() (*linodego.Client, error) {
	token := os.Getenv("LINODE_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("LINODE_TOKEN must be set for acceptance tests")
	}

	apiVersion := cmp.Or(os.Getenv("LINODE_API_VERSION"), "v4beta")

	config := &helper.Config{
		AccessToken: token,
		APIVersion:  apiVersion,
		APIURL:      os.Getenv("LINODE_URL"),
	}

	client, err := config.Client(context.Background())
	if err != nil {
		return nil, err
	}

	return client, nil
}

func GetTestClientAlternateToken(tokenName string) (*linodego.Client, error) {
	token := os.Getenv(tokenName)
	if token == "" {
		return nil, fmt.Errorf("%s must be set for acceptance tests", tokenName)
	}

	apiVersion := cmp.Or(os.Getenv("LINODE_API_VERSION"), "v4beta")

	config := &helper.Config{
		AccessToken: token,
		APIVersion:  apiVersion,
		APIURL:      os.Getenv("LINODE_URL"),
	}

	client, err := config.Client(context.Background())
	if err != nil {
		return nil, err
	}

	return client, nil
}

func GetFrameworkTestClient(
	t *testing.T,
	httpClientModifiers []helper.HTTPClientModifier,
) *linodego.Client {
	token := os.Getenv("LINODE_TOKEN")
	require.NotEmptyf(t, token, "LINODE_TOKEN must be set for acceptance tests")

	apiVersion := cmp.Or(os.Getenv("LINODE_API_VERSION"), "v4beta")

	var diags fwDiag.Diagnostics

	apiURL := types.StringNull()

	if linodeURL, ok := os.LookupEnv("LINODE_URL"); ok {
		apiURL = types.StringValue(linodeURL)
	}

	client := TestAccFrameworkProvider.InitLinodeClient(
		t.Context(),
		&helper.FrameworkProviderModel{
			AccessToken: types.StringValue(token),
			APIURL:      apiURL,
			APIVersion:  types.StringValue(apiVersion),
		},
		helper.GetFrameworkVersion(),
		httpClientModifiers,
		&diags,
	)
	if diags.HasError() {
		t.Fatal(diags.Errors())
	}

	return client
}
