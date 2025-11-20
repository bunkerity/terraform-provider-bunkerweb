provider "bunkerweb" {
  api_endpoint = "https://127.0.0.1:5000/api"
  api_token    = "changeme"
}

data "bunkerweb_service" "example" {
  id = "app.example.com"
}
