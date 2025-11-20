provider "bunkerweb" {
  api_endpoint = "https://127.0.0.1:5000/api"
  api_token    = "changeme"
}

resource "bunkerweb_ban" "blocked_host" {
  ip                 = "198.51.100.10"
  reason             = "manual"
  expiration_seconds = 86400
}
