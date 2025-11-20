// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccBunkerWebConfigBulkDeleteEphemeralResource(t *testing.T) {
	fakeAPI := newFakeBunkerWebAPI(t)

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBunkerWebConfigBulkDeleteEphemeralResource(fakeAPI.URL()),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("ephemeral.bunkerweb_config_bulk_delete.cleanup", "result"),
				),
			},
		},
	})

	if _, ok := fakeAPI.Config("global", "http", "foo"); ok {
		t.Fatalf("expected foo config to be deleted")
	}
	if _, ok := fakeAPI.Config("api", "http", "bar"); ok {
		t.Fatalf("expected bar config to be deleted")
	}

	batches := fakeAPI.DeletedConfigBatches()
	if len(batches) == 0 {
		t.Fatalf("expected delete batches to be recorded")
	}
	if len(batches[0]) != 2 {
		t.Fatalf("expected first batch to delete two configs, got %d", len(batches[0]))
	}
}

func testAccBunkerWebConfigBulkDeleteEphemeralResource(endpoint string) string {
	return fmt.Sprintf(`
provider "bunkerweb" {
  api_endpoint = "%s"
  api_token    = "test-token"
}

resource "bunkerweb_config" "foo" {
  type = "http"
  name = "foo"
  data = "server { listen 80; }"
}

resource "bunkerweb_config" "bar" {
  service = "api"
  type    = "http"
  name    = "bar"
  data    = "server { listen 81; }"
}

ephemeral "bunkerweb_config_bulk_delete" "cleanup" {
  configs = [
    {
      type = bunkerweb_config.foo.type
      name = bunkerweb_config.foo.name
    },
    {
      service = bunkerweb_config.bar.service
      type    = bunkerweb_config.bar.type
      name    = bunkerweb_config.bar.name
    }
  ]

  depends_on = [bunkerweb_config.foo, bunkerweb_config.bar]
}
`, endpoint)
}
