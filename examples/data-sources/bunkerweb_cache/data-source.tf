provider "bunkerweb" {
  api_endpoint = "https://127.0.0.1:5000/api"
  api_token    = "changeme"
}

data "bunkerweb_cache" "logs" {
  plugin    = "reporter"
  with_data = false
}

output "cache_entries" {
  value = data.bunkerweb_cache.logs.entries
}
