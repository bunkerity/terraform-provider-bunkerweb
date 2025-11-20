provider "bunkerweb" {
  api_endpoint = "https://127.0.0.1:5000/api"
  api_token    = "changeme"
}

ephemeral "bunkerweb_config_upload" "batch" {
  service = "web"
  type    = "http"

  files = [
    {
      name    = "http.conf"
      content = "server { listen 80; }"
    },
    {
      name    = "https.conf"
      content = "server { listen 443 ssl; }"
    }
  ]
}
