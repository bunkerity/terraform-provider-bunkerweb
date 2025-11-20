// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBunkerWebPluginResource(t *testing.T) {
	fakeAPI := newFakeBunkerWebAPI(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBunkerWebPluginResourceConfig(fakeAPI.URL(), "custom.lua", "return 42"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bunkerweb_plugin.custom", "name", "custom.lua"),
					resource.TestCheckResourceAttrSet("bunkerweb_plugin.custom", "id"),
				),
			},
			{
				ResourceName:      "bunkerweb_plugin.custom",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	if _, ok := fakeAPI.Plugin("custom"); !ok {
		t.Fatalf("expected plugin to remain uploaded after acceptance test")
	}
}

func testAccBunkerWebPluginResourceConfig(endpoint, name, content string) string {
	return fmt.Sprintf(`
provider "bunkerweb" {
  api_endpoint = "%s"
  api_token    = "test-token"
}

resource "bunkerweb_plugin" "custom" {
  name    = "%s"
  content = "%s"
  method  = "custom"
}
`, endpoint, name, content)
}
