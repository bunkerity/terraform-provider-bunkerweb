// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccBunkerWebConfigUploadEphemeralResource(t *testing.T) {
	fakeAPI := newFakeBunkerWebAPI(t)

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBunkerWebConfigUploadEphemeralResource(fakeAPI.URL()),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("ephemeral.bunkerweb_config_upload.batch", "result"),
				),
			},
		},
	})

	cfg, ok := fakeAPI.Config("web", "http", "alpha")
	if !ok {
		t.Fatalf("expected alpha config to exist after upload")
	}
	if cfg.Data != "server { listen 80; }" {
		t.Fatalf("unexpected data for alpha config: %q", cfg.Data)
	}

	cfg, ok = fakeAPI.Config("web", "http", "beta")
	if !ok {
		t.Fatalf("expected beta config to exist after upload")
	}
	if cfg.Data != "server { listen 443 ssl; }" {
		t.Fatalf("unexpected data for beta config: %q", cfg.Data)
	}
}

func testAccBunkerWebConfigUploadEphemeralResource(endpoint string) string {
	return fmt.Sprintf(`
provider "bunkerweb" {
  api_endpoint = "%s"
  api_token    = "test-token"
}

ephemeral "bunkerweb_config_upload" "batch" {
  service = "web"
  type    = "http"

  files = [
    {
      name    = "alpha.conf"
      content = "server { listen 80; }"
    },
    {
      name    = "beta.cfg"
      content = "server { listen 443 ssl; }"
    }
  ]
}
`, endpoint)
}
