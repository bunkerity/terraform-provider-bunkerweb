// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBunkerWebJobsDataSource(t *testing.T) {
	fakeAPI := newFakeBunkerWebAPI(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBunkerWebJobsDataSourceConfig(fakeAPI.URL()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.bunkerweb_jobs.all", "jobs.#", "1"),
					resource.TestCheckResourceAttr("data.bunkerweb_jobs.all", "jobs.0.plugin", "reporter"),
				),
			},
		},
	})
}

func testAccBunkerWebJobsDataSourceConfig(endpoint string) string {
	return fmt.Sprintf(`
provider "bunkerweb" {
  api_endpoint = "%s"
  api_token    = "test-token"
}

data "bunkerweb_jobs" "all" {}
`, endpoint)
}
