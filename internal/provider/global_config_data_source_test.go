// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBunkerWebGlobalConfigDataSource(t *testing.T) {
	fakeAPI := newFakeBunkerWebAPI(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBunkerWebGlobalConfigDataSourceConfig(fakeAPI.URL()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.bunkerweb_global_config.current", "settings.%", "3"),
					resource.TestCheckResourceAttr("data.bunkerweb_global_config.current", "settings.some_setting", "value"),
					resource.TestCheckResourceAttr("data.bunkerweb_global_config.current", "settings.feature_enabled", "true"),
					resource.TestCheckResourceAttr("data.bunkerweb_global_config.current", "settings.retry_limit", "5"),
				),
			},
		},
	})
}

func testAccBunkerWebGlobalConfigDataSourceConfig(endpoint string) string {
	return fmt.Sprintf(`
provider "bunkerweb" {
  api_endpoint = "%s"
  api_token    = "test-token"
}

data "bunkerweb_global_config" "current" {
  full = false
}
`, endpoint)
}
