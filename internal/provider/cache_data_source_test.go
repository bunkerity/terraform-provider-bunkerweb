// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBunkerWebCacheDataSource(t *testing.T) {
	fakeAPI := newFakeBunkerWebAPI(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBunkerWebCacheDataSourceConfig(fakeAPI.URL()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.bunkerweb_cache.entries", "entries.#", "1"),
					resource.TestCheckResourceAttr("data.bunkerweb_cache.entries", "entries.0.file_name", "summary.txt"),
				),
			},
		},
	})
}

func testAccBunkerWebCacheDataSourceConfig(endpoint string) string {
	return fmt.Sprintf(`
provider "bunkerweb" {
  api_endpoint = "%s"
  api_token    = "test-token"
}

data "bunkerweb_cache" "entries" {
  with_data = false
}
`, endpoint)
}
