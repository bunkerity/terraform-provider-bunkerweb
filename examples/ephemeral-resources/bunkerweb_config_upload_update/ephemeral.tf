provider "bunkerweb" {
  api_endpoint = "https://127.0.0.1:5000/api"
  api_token    = "changeme"
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
