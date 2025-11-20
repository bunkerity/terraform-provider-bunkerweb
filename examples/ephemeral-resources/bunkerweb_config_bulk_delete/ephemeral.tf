provider "bunkerweb" {
  api_endpoint = "https://127.0.0.1:5000/api"
  api_token    = "changeme"
}

resource "bunkerweb_config" "foo" {
  type = "http"
  name = "foo"
  data = "server { listen 80; }"
}

resource "bunkerweb_config" "bar" {
  service = "api"
  type    = "http"
  name    = "bar"
  data    = "server { listen 81; }"
}

ephemeral "bunkerweb_config_bulk_delete" "cleanup" {
  configs = [
    {
      type = bunkerweb_config.foo.type
      name = bunkerweb_config.foo.name
    },
    {
      service = bunkerweb_config.bar.service
      type    = bunkerweb_config.bar.type
      name    = bunkerweb_config.bar.name
    }
  ]

  depends_on = [bunkerweb_config.foo, bunkerweb_config.bar]
}
