// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

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
