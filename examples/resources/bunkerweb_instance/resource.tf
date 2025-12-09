provider "bunkerweb" {
  api_endpoint = "https://127.0.0.1:8888"
  # Bearer token Auth
  api_token = var.api_token # If you choose to use Bearer Token configured in your API deployment
  # OR Basic Auth
  api_username = var.api_username # Basic Auth configured in your API deployment.
  api_password = var.api_password # required with api_username to work.
}

resource "bunkerweb_instance" "example" {
  hostname     = "worker-1.example.internal"
  name         = "Worker 1"
  port         = 8080
  listen_https = true
  https_port   = 8443
  server_name  = "worker-1.example.internal"
  method       = "api"
}
