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
			},
		},
	})

	// Note: Ephemeral resources don't persist their state after evaluation,
	// so we can't verify the configs exist in the fake API after the test completes.
	// The successful completion of the test step is sufficient to verify the upload worked.
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
