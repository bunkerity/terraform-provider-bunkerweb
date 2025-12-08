// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccBunkerWebInstanceActionEphemeralResource(t *testing.T) {
	fakeAPI := newFakeBunkerWebAPI(t)

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// First create the instance
				Config: testAccBunkerWebInstanceActionInstanceOnlyConfig(fakeAPI.URL()),
			},
			{
				// Then use ephemeral resources that reference it
				Config: testAccBunkerWebInstanceActionEphemeralResourceConfig(fakeAPI.URL()),
			},
		},
	})

	hosts := fakeAPI.PingHosts()
	if len(hosts) == 0 || hosts[len(hosts)-1] != "edge-1" {
		t.Fatalf("expected ping host history to include edge-1, got %v", hosts)
	}

	reloadCalls := fakeAPI.ReloadHostCalls()
	if len(reloadCalls) == 0 || reloadCalls[len(reloadCalls)-1].host != "edge-1" || reloadCalls[len(reloadCalls)-1].test {
		t.Fatalf("expected reload host call for edge-1 with test=false, got %v", reloadCalls)
	}

	if fakeAPI.StopAllCount() == 0 {
		t.Fatalf("expected stop all to be invoked")
	}
}

func testAccBunkerWebInstanceActionInstanceOnlyConfig(endpoint string) string {
	return fmt.Sprintf(`
provider "bunkerweb" {
  api_endpoint = "%s"
  api_token    = "test-token"
}

resource "bunkerweb_instance" "edge" {
  hostname = "edge-1"
}
`, endpoint)
}

func testAccBunkerWebInstanceActionEphemeralResourceConfig(endpoint string) string {
	return fmt.Sprintf(`
provider "bunkerweb" {
  api_endpoint = "%s"
  api_token    = "test-token"
}

resource "bunkerweb_instance" "edge" {
  hostname = "edge-1"
}

ephemeral "bunkerweb_instance_action" "ping_host" {
  operation = "ping"
  hostnames = ["edge-1"]
  depends_on = [bunkerweb_instance.edge]
}

ephemeral "bunkerweb_instance_action" "reload_host" {
  operation = "reload"
  hostnames = ["edge-1"]
  test      = false
  depends_on = [bunkerweb_instance.edge]
}

ephemeral "bunkerweb_instance_action" "stop_all" {
  operation = "stop"
  depends_on = [bunkerweb_instance.edge]
}
`, endpoint)
}
