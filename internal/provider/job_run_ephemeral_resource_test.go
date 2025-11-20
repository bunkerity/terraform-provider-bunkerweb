// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccBunkerWebRunJobsEphemeralResource(t *testing.T) {
	fakeAPI := newFakeBunkerWebAPI(t)

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBunkerWebRunJobsEphemeralResourceConfig(fakeAPI.URL()),
			},
		},
	})

	if len(fakeAPI.runJobs) == 0 {
		t.Fatalf("expected run jobs request to be captured")
	}
}

func testAccBunkerWebRunJobsEphemeralResourceConfig(endpoint string) string {
	return fmt.Sprintf(`
provider "bunkerweb" {
  api_endpoint = "%s"
  api_token    = "test-token"
}

ephemeral "bunkerweb_run_jobs" "trigger" {
  jobs = [{
    plugin = "reporter"
    name   = "daily"
  }]
}
`, endpoint)
}
