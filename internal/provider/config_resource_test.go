// Copyright Bunkerity 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestBunkerWebConfigPopulateFromConfigPreservesType locks the Read behaviour for
// non-canonical config types: "server-http" normalises to the API's "server_http",
// so it must be preserved (type is RequiresReplace) instead of triggering a replace.
func TestBunkerWebConfigPopulateFromConfigPreservesType(t *testing.T) {
	m := &BunkerWebConfigResourceModel{
		Type: types.StringValue("server-http"),
		Name: types.StringValue("snippet"),
	}
	cfg := &bunkerWebConfig{Service: "global", Type: "server_http", Name: "snippet", Data: "x", Method: "api"}

	if diags := m.populateFromConfig(cfg); diags.HasError() {
		t.Fatalf("populateFromConfig: %v", diags)
	}
	if got := m.Type.ValueString(); got != "server-http" {
		t.Fatalf("expected configured type preserved, got %q", got)
	}
	if got := m.ID.ValueString(); got != "global/server-http/snippet" {
		t.Fatalf("expected id built from configured type, got %q", got)
	}

	// A genuinely different type must be adopted from the API.
	m2 := &BunkerWebConfigResourceModel{Type: types.StringValue("http"), Name: types.StringValue("snippet")}
	if diags := m2.populateFromConfig(cfg); diags.HasError() {
		t.Fatalf("populateFromConfig: %v", diags)
	}
	if got := m2.Type.ValueString(); got != "server_http" {
		t.Fatalf("expected API type adopted, got %q", got)
	}
}

func TestAccBunkerWebConfigResource(t *testing.T) {
	fakeAPI := newFakeBunkerWebAPI(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBunkerWebConfigResourceConfig(fakeAPI.URL(), "server_http", "access_log", "log_format combined;"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bunkerweb_config.sample", "service", "global"),
					resource.TestCheckResourceAttr("bunkerweb_config.sample", "type", "server_http"),
					resource.TestCheckResourceAttr("bunkerweb_config.sample", "name", "access_log"),
					resource.TestCheckResourceAttr("bunkerweb_config.sample", "data", "log_format combined;"),
				),
			},
			{
				Config: testAccBunkerWebConfigResourceConfig(fakeAPI.URL(), "server_http", "access_log", "log_format custom;"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bunkerweb_config.sample", "data", "log_format custom;"),
				),
			},
		},
	})
}

func testAccBunkerWebConfigResourceConfig(endpoint, cfgType, name, data string) string {
	return fmt.Sprintf(`
provider "bunkerweb" {
  api_endpoint = "%s"
  api_token    = "test-token"
}

resource "bunkerweb_config" "sample" {
  type = "%s"
  name = "%s"
  data = "%s"
}
`, endpoint, cfgType, name, data)
}
