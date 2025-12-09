provider "bunkerweb" {
  api_endpoint = "https://127.0.0.1:8888"
  # Bearer token Auth
  api_token = var.api_token # If you choose to use Bearer Token configured in your API deployment
  # OR Basic Auth
  api_username = var.api_username # Basic Auth configured in your API deployment.
  api_password = var.api_password # required with api_username to work.
}

resource "bunkerweb_config" "primary" {
  type = "http"
  name = "primary"
  data = "server { listen 8080; }"
}

ephemeral "bunkerweb_config_upload_update" "promote" {
  type    = bunkerweb_config.primary.type
  name    = bunkerweb_config.primary.name
  content = "stream { listen 9000; }"

  new_service = "backend"
  new_type    = "stream"
  new_name    = "promoted"

  depends_on = [bunkerweb_config.primary]
}
