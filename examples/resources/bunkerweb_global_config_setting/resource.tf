provider "bunkerweb" {
  api_endpoint = "http://127.0.0.1:5000/api"
  api_token    = "example-token"
}

resource "bunkerweb_global_config_setting" "retry" {
  key   = "retry_limit"
  value = "10"
}
