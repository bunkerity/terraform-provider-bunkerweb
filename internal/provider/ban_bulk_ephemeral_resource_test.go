// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccBunkerWebBanBulkEphemeralResource(t *testing.T) {
	fakeAPI := newFakeBunkerWebAPI(t)

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBunkerWebBanBulkEphemeralResourceConfig(fakeAPI.URL()),
			},
		},
	})

	created := fakeAPI.CreatedBanBatches()
	if len(created) == 0 {
		t.Fatalf("expected ban batch to be recorded")
	}
	if len(created[0]) != 2 {
		t.Fatalf("expected two bans in first batch, got %d", len(created[0]))
	}

	deleted := fakeAPI.DeletedBanBatches()
	if len(deleted) == 0 {
		t.Fatalf("expected unban batch to be recorded")
	}
	if len(deleted[0]) != 1 {
		t.Fatalf("expected one unban in first batch, got %d", len(deleted[0]))
	}
}

func testAccBunkerWebBanBulkEphemeralResourceConfig(endpoint string) string {
	return fmt.Sprintf(`
provider "bunkerweb" {
  api_endpoint = "%s"
  api_token    = "test-token"
}

ephemeral "bunkerweb_ban_bulk" "batch" {
  bans = [
    {
      ip        = "203.0.113.10"
      reason    = "automation"
      expires_in = 600
    },
    {
      ip      = "203.0.113.11"
      service = "frontend"
    }
  ]

  unbans = [
    {
      ip = "203.0.113.11"
    }
  ]
}
`, endpoint)
}
