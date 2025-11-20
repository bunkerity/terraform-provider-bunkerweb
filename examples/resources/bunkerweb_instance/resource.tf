provider "bunkerweb" {
  api_endpoint = "https://127.0.0.1:5000/api"
  api_token    = "changeme"
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
