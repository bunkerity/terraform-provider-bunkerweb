// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBunkerWebBanResource(t *testing.T) {
	fakeAPI := newFakeBunkerWebAPI(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBunkerWebBanResourceConfig(fakeAPI.URL(), "192.0.2.10", "maintenance", 3600),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bunkerweb_ban.block", "ip", "192.0.2.10"),
					resource.TestCheckResourceAttr("bunkerweb_ban.block", "service", "maintenance"),
					resource.TestCheckResourceAttr("bunkerweb_ban.block", "reason", "manual"),
					resource.TestCheckResourceAttr("bunkerweb_ban.block", "expiration_seconds", "3600"),
				),
			},
			{
				ResourceName:      "bunkerweb_ban.block",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccBunkerWebBanResourceConfig(endpoint, ip, service string, exp int) string {
	return fmt.Sprintf(`
provider "bunkerweb" {
  api_endpoint = "%s"
  api_token    = "test-token"
}

resource "bunkerweb_ban" "block" {
  ip                 = "%s"
  service            = "%s"
  reason             = "manual"
  expiration_seconds = %d
}
`, endpoint, ip, service, exp)
}
