// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBunkerWebGlobalConfigResource(t *testing.T) {
	fakeAPI := newFakeBunkerWebAPI(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBunkerWebGlobalConfigResourceConfigValue(fakeAPI.URL()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bunkerweb_global_config_setting.retry", "id", "retry_limit"),
					resource.TestCheckResourceAttr("bunkerweb_global_config_setting.retry", "value", "10"),
					resource.TestCheckNoResourceAttr("bunkerweb_global_config_setting.retry", "value_json"),
				),
			},
			{
				Config: testAccBunkerWebGlobalConfigResourceConfigJSON(fakeAPI.URL()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bunkerweb_global_config_setting.retry", "value_json", "true"),
					resource.TestCheckNoResourceAttr("bunkerweb_global_config_setting.retry", "value"),
				),
			},
			{
				ResourceName:      "bunkerweb_global_config_setting.retry",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"value",      // Import always returns value (not value_json)
					"value_json", // The format (value vs value_json) is not preserved during import
				},
			},
		},
	})
}

func testAccBunkerWebGlobalConfigResourceConfigValue(endpoint string) string {
	return fmt.Sprintf(`
provider "bunkerweb" {
  api_endpoint = "%s"
  api_token    = "test-token"
}

resource "bunkerweb_global_config_setting" "retry" {
  key   = "retry_limit"
  value = "10"
}
`, endpoint)
}

func testAccBunkerWebGlobalConfigResourceConfigJSON(endpoint string) string {
	return fmt.Sprintf(`
provider "bunkerweb" {
  api_endpoint = "%s"
  api_token    = "test-token"
}

resource "bunkerweb_global_config_setting" "retry" {
  key        = "retry_limit"
  value_json = jsonencode(true)
}
`, endpoint)
}
