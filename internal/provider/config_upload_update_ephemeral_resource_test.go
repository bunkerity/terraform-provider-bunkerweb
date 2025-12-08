// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccBunkerWebConfigUploadUpdateEphemeralResource(t *testing.T) {
	fakeAPI := newFakeBunkerWebAPI(t)

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             testAccBunkerWebConfigUploadUpdateEphemeralResource(fakeAPI.URL()),
				ExpectNonEmptyPlan: true, // Ephemeral resource modifies managed resources
			},
		},
	})

	if _, ok := fakeAPI.Config("global", "http", "primary"); ok {
		t.Fatalf("expected original config location to be removed")
	}

	cfg, ok := fakeAPI.Config("backend", "stream", "processed")
	if !ok {
		t.Fatalf("expected config to be moved to backend/stream")
	}
	if cfg.Data != "stream { listen 9000; }" {
		t.Fatalf("unexpected data for updated config: %q", cfg.Data)
	}
}

func testAccBunkerWebConfigUploadUpdateEphemeralResource(endpoint string) string {
	return fmt.Sprintf(`
provider "bunkerweb" {
  api_endpoint = "%s"
  api_token    = "test-token"
}

resource "bunkerweb_config" "primary" {
  type = "http"
  name = "primary"
  data = "server { listen 8080; }"
}

ephemeral "bunkerweb_config_upload_update" "promote" {
  type     = bunkerweb_config.primary.type
  name     = bunkerweb_config.primary.name
  content  = "stream { listen 9000; }"
  file_name = "primary.conf"
  new_service = "backend"
  new_type    = "stream"
  new_name    = "processed"

  depends_on = [bunkerweb_config.primary]
}
`, endpoint)
}
