// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBunkerWebInstanceResource(t *testing.T) {
	fakeAPI := newFakeBunkerWebAPI(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBunkerWebInstanceResourceConfigCreate(fakeAPI.URL()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bunkerweb_instance.worker", "hostname", "worker-1.example.internal"),
					resource.TestCheckResourceAttr("bunkerweb_instance.worker", "name", "Worker 1"),
					resource.TestCheckResourceAttr("bunkerweb_instance.worker", "port", "8080"),
					resource.TestCheckResourceAttr("bunkerweb_instance.worker", "listen_https", "true"),
					resource.TestCheckResourceAttr("bunkerweb_instance.worker", "https_port", "8443"),
					resource.TestCheckResourceAttr("bunkerweb_instance.worker", "server_name", "worker-1.example.internal"),
					resource.TestCheckResourceAttr("bunkerweb_instance.worker", "method", "api"),
				),
			},
			{
				ResourceName:            "bunkerweb_instance.worker",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"method"},
			},
			{
				Config: testAccBunkerWebInstanceResourceConfigUpdate(fakeAPI.URL()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bunkerweb_instance.worker", "name", "Worker node"),
					resource.TestCheckResourceAttr("bunkerweb_instance.worker", "port", "8081"),
					resource.TestCheckResourceAttr("bunkerweb_instance.worker", "listen_https", "false"),
					resource.TestCheckResourceAttr("bunkerweb_instance.worker", "https_port", "7443"),
					resource.TestCheckResourceAttr("bunkerweb_instance.worker", "server_name", "worker.internal"),
				),
			},
		},
	})
}

func testAccBunkerWebInstanceResourceConfigCreate(endpoint string) string {
	return fmt.Sprintf(`
provider "bunkerweb" {
  api_endpoint = "%s"
  api_token    = "test-token"
}

resource "bunkerweb_instance" "worker" {
  hostname     = "worker-1.example.internal"
  name         = "Worker 1"
  port         = 8080
  listen_https = true
  https_port   = 8443
  server_name  = "worker-1.example.internal"
  method       = "api"
}
`, endpoint)
}

func testAccBunkerWebInstanceResourceConfigUpdate(endpoint string) string {
	return fmt.Sprintf(`
provider "bunkerweb" {
  api_endpoint = "%s"
  api_token    = "test-token"
}

resource "bunkerweb_instance" "worker" {
  hostname     = "worker-1.example.internal"
  name         = "Worker node"
  port         = 8081
  listen_https = false
  https_port   = 7443
  server_name  = "worker.internal"
  method       = "api"
}
`, endpoint)
}
