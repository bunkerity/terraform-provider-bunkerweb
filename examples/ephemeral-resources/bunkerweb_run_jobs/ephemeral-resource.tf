provider "bunkerweb" {
  api_endpoint = "https://127.0.0.1:5000/api"
  api_token    = "changeme"
}

ephemeral "bunkerweb_run_jobs" "trigger" {
  jobs = [{
    plugin = "reporter"
    name   = "daily"
  }]
}
