provider "bunkerweb" {
  api_endpoint = "https://127.0.0.1:5000/api"
  api_token    = "changeme"
}

data "bunkerweb_global_config" "current" {
  full = true
}

output "global_settings" {
  value = data.bunkerweb_global_config.current.settings
}
