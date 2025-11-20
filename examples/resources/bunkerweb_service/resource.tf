provider "bunkerweb" {
  api_endpoint = "https://127.0.0.1:5000/api"
  api_token    = "changeme"
}

resource "bunkerweb_service" "example" {
  server_name = "app.example.com"

  variables = {
    upstream = "10.0.0.12"
    mode     = "production"
  }
}
