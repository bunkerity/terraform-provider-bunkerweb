// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccBunkerWebServiceConvertEphemeralResource(t *testing.T) {
	fakeAPI := newFakeBunkerWebAPI(t)

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBunkerWebServiceConvertEphemeralResourceConfig(fakeAPI.URL()),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("ephemeral.bunkerweb_service_convert.convert", "is_draft", "true"),
				),
			},
		},
	})

	calls := fakeAPI.ConvertCalls()
	if len(calls) == 0 {
		t.Fatalf("expected convert endpoint to be invoked")
	}

	last := calls[len(calls)-1]
	if last.target != "draft" {
		t.Fatalf("expected last convert target to be draft, got %s", last.target)
	}
}

func testAccBunkerWebServiceConvertEphemeralResourceConfig(endpoint string) string {
	return fmt.Sprintf(`
provider "bunkerweb" {
  api_endpoint = "%s"
  api_token    = "test-token"
}

resource "bunkerweb_service" "app" {
  server_name = "app.example.com"
}

ephemeral "bunkerweb_service_convert" "convert" {
  service_id = bunkerweb_service.app.id
  convert_to = "draft"
  depends_on = [bunkerweb_service.app]
}
`, endpoint)
}
