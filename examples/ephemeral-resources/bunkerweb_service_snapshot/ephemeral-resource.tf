provider "bunkerweb" {
  api_endpoint = "https://127.0.0.1:5000/api"
  api_token    = "changeme"
}

resource "bunkerweb_service" "example" {
  server_name = "app.example.com"
}

ephemeral "bunkerweb_service_snapshot" "current" {
  service_id = bunkerweb_service.example.id
}
