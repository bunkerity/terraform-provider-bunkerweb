// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccBunkerWebEphemeralResource(t *testing.T) {
	fakeAPI := newFakeBunkerWebAPI(t)

	resource.Test(t, resource.TestCase{
		// Ephemeral resources are only available in 1.10 and later
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesWithEcho,
		Steps: []resource.TestStep{
			{
				Config: testAccBunkerWebEphemeralResourceConfig(fakeAPI.URL(), "initial"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"echo.snapshot",
						tfjsonpath.New("data").AtMapKey("server_name"),
						knownvalue.StringExact("test.example.com"),
					),
					statecheck.ExpectKnownValue(
						"echo.snapshot",
						tfjsonpath.New("data").AtMapKey("variables").AtMapKey("test"),
						knownvalue.StringExact("initial"),
					),
				},
			},
		},
	})
}

func testAccBunkerWebEphemeralResourceConfig(endpoint, value string) string {
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

ephemeral "bunkerweb_service_snapshot" "test" {
  service_id = bunkerweb_service.test.id
}

provider "echo" {
  data = ephemeral.bunkerweb_service_snapshot.test
}

resource "echo" "snapshot" {}
`, endpoint, value)
}
