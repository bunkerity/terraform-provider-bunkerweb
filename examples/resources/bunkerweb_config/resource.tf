provider "bunkerweb" {
  api_endpoint = "https://127.0.0.1:5000/api"
  api_token    = "changeme"
}

resource "bunkerweb_config" "http_snippet" {
  type = "http"
  name = "log_settings"
  data = "log_format combined '$remote_addr - $remote_user [$time_local] \"$request\" $status $body_bytes_sent';"
}
