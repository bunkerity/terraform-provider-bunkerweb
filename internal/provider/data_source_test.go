// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBunkerWebDataSource(t *testing.T) {
	fakeAPI := newFakeBunkerWebAPI(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccBunkerWebDataSourceConfig(fakeAPI.URL()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.bunkerweb_service.test", "server_name", "test.example.com"),
					resource.TestCheckResourceAttr("data.bunkerweb_service.test", "variables.test", "one"),
				),
			},
		},
	})
}

func testAccBunkerWebDataSourceConfig(endpoint string) string {
	return fmt.Sprintf(`
provider "bunkerweb" {
  api_endpoint = "%s"
  api_token    = "test-token"
}

resource "bunkerweb_service" "test" {
  server_name = "test.example.com"
  variables = {
    test = "one"
  }
}

data "bunkerweb_service" "test" {
  id = bunkerweb_service.test.id
}
`, endpoint)
}
