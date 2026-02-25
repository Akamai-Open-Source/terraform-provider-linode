//go:build integration || token

package token_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/linode/linodego"
	"github.com/linode/terraform-provider-linode/v3/linode/acceptance"
	"github.com/linode/terraform-provider-linode/v3/linode/token/tmpl"
)

func init() {
	resource.AddTestSweepers("linode_token", &resource.Sweeper{
		Name: "linode_token",
		F:    sweep,
	})
}

func sweep(prefix string) error {
	client, err := acceptance.GetTestClient()
	if err != nil {
		return fmt.Errorf("Error getting client: %s", err)
	}

	listOpts := acceptance.SweeperListOptions(prefix, "label")
	tokens, err := client.ListTokens(context.Background(), listOpts)
	if err != nil {
		return fmt.Errorf("Error getting tokens: %s", err)
	}
	for _, token := range tokens {
		if !acceptance.ShouldSweep(prefix, token.Label) {
			continue
		}
		err := client.DeleteToken(context.Background(), token.ID)
		if err != nil {
			return fmt.Errorf("Error destroying %s during sweep: %s", token.Label, err)
		}
	}

	return nil
}

func TestAccResourceToken_basic(t *testing.T) {
	t.Parallel()

	resName := "linode_token.foobar"
	tokenName := acctest.RandomWithPrefix("tf_test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		CheckDestroy:             checkTokenDestroy,
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, tokenName),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckTokenExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(tokenName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("expiry"), knownvalue.StringExact("2100-01-02T03:04:05Z")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("scopes"), knownvalue.StringExact("linodes:read_only")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("token"), knownvalue.NotNull()),
				},
			},
			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
			{
				Config: tmpl.Updates(t, tokenName),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckTokenExists(),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(fmt.Sprintf("%s_renamed", tokenName))),
				},
			},
		},
	})
}

func TestAccResourceToken_recreative_update(t *testing.T) {
	t.Parallel()

	resName := "linode_token.foobar"
	tokenName := acctest.RandomWithPrefix("tf_test")

	var currentToken string
	tokenRecreatedCheck := acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		for _, rs := range req.State.Values.RootModule.Resources {
			if rs.Type != "linode_token" {
				continue
			}

			newToken, ok := rs.AttributeValues["token"].(string)
			if !ok {
				resp.Error = fmt.Errorf("Can't find the token in the state.")
				return
			}
			if newToken == currentToken {
				resp.Error = fmt.Errorf("The token suppose to be but was not recreated.")
				return
			}
		}
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.PreCheck(t) },
		CheckDestroy:             checkTokenDestroy,
		ProtoV6ProviderFactories: acceptance.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tmpl.Basic(t, tokenName),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckTokenExists(),
					tokenRecreatedCheck,
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("label"), knownvalue.StringExact(tokenName)),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("expiry"), knownvalue.StringExact("2100-01-02T03:04:05Z")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("scopes"), knownvalue.StringExact("linodes:read_only")),
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("token"), knownvalue.NotNull()),
				},
			},
			{
				ResourceName:            resName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
			{
				Config: tmpl.RecreateNewExpiryDate(t, tokenName, "2099-05-04T03:02:01+00:00"),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckTokenExists(),
					tokenRecreatedCheck,
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("expiry"), knownvalue.StringExact("2099-05-04T03:02:01+00:00")),
				},
			},
			{
				Config: tmpl.RecreateNewScopes(t, tokenName, "linodes:read_only lke:read_only"),
				ConfigStateChecks: []statecheck.StateCheck{
					stateCheckTokenExists(),
					tokenRecreatedCheck,
					statecheck.ExpectKnownValue(resName, tfjsonpath.New("scopes"), knownvalue.StringExact("linodes:read_only lke:read_only")),
				},
			},
		},
	})
}

func stateCheckTokenExists() statecheck.StateCheck {
	return acceptance.CustomStateCheck(func(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		client := acceptance.TestAccFrameworkProvider.Meta.Client

		for _, rs := range req.State.Values.RootModule.Resources {
			if rs.Type != "linode_token" {
				continue
			}

			idStr, ok := rs.AttributeValues["id"].(string)
			if !ok {
				resp.Error = fmt.Errorf("Error parsing ID for token")
				return
			}

			id, err := strconv.Atoi(idStr)
			if err != nil {
				resp.Error = fmt.Errorf("Error parsing %v to int", idStr)
				return
			}

			_, err = client.GetToken(ctx, id)
			if err != nil {
				label, _ := rs.AttributeValues["label"].(string)
				resp.Error = fmt.Errorf("Error retrieving state of Token %s: %s", label, err)
				return
			}
		}
	})
}

func checkTokenDestroy(s *terraform.State) error {
	client := acceptance.TestAccFrameworkProvider.Meta.Client
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "linode_token" {
			continue
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error parsing %v to int", rs.Primary.ID)
		}
		if id == 0 {
			return fmt.Errorf("Would have considered %v as %d", rs.Primary.ID, id)
		}

		_, err = client.GetToken(context.Background(), id)

		if err == nil {
			return fmt.Errorf("Linode Token with id %d still exists", id)
		}

		if apiErr, ok := err.(*linodego.Error); ok && apiErr.Code != 404 {
			return fmt.Errorf("Error requesting Linode Token with id %d", id)
		}
	}

	return nil
}
