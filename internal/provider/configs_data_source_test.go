// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBunkerWebConfigsDataSource(t *testing.T) {
	fakeAPI := newFakeBunkerWebAPI(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBunkerWebConfigsDataSourceConfig(fakeAPI.URL()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.bunkerweb_configs.all", "configs.#", "2"),
					resource.TestCheckResourceAttr("data.bunkerweb_configs.global", "configs.#", "1"),
				),
			},
		},
	})
}

func testAccBunkerWebConfigsDataSourceConfig(endpoint string) string {
	return fmt.Sprintf(`
provider "bunkerweb" {
  api_endpoint = "%s"
  api_token    = "test-token"
}

resource "bunkerweb_config" "app" {
  service = "app"
  type    = "http"
  name    = "app.conf"
  data    = "content"
}

resource "bunkerweb_config" "global_conf" {
  type = "http"
  name = "global.conf"
  data = "global content"
}

data "bunkerweb_configs" "all" {
  depends_on = [bunkerweb_config.app, bunkerweb_config.global_conf]
}

data "bunkerweb_configs" "global" {
  service    = "global"
  with_data  = true
  depends_on = [bunkerweb_config.global_conf]
}

`, endpoint)
}
