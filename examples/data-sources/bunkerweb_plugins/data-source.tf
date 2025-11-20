provider "bunkerweb" {
  api_endpoint = "https://127.0.0.1:5000/api"
  api_token    = "changeme"
}

data "bunkerweb_plugins" "ui" {
  type = "ui"
}

output "plugin_ids" {
  value = [for plugin in data.bunkerweb_plugins.ui.plugins : plugin.id]
}
