// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBunkerWebResource(t *testing.T) {
	fakeAPI := newFakeBunkerWebAPI(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBunkerWebResourceConfig(fakeAPI.URL(), "one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bunkerweb_service.test", "server_name", "test.example.com"),
					resource.TestCheckResourceAttr("bunkerweb_service.test", "is_draft", "false"),
					resource.TestCheckResourceAttr("bunkerweb_service.test", "variables.test", "one"),
				),
			},
			{
				ResourceName:            "bunkerweb_service.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"variables"},
			},
			{
				Config: testAccBunkerWebResourceConfig(fakeAPI.URL(), "two"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bunkerweb_service.test", "variables.test", "two"),
				),
			},
		},
	})
}

func testAccBunkerWebResourceConfig(endpoint, value string) string {
	return fmt.Sprintf(`
provider "bunkerweb" {
  api_endpoint = "%s"
  api_token    = "test-token"
}

resource "bunkerweb_service" "test" {
  server_name = "test.example.com"
  variables = {
    test = "%s"
  }
}
`, endpoint, value)
}
