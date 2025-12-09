provider "bunkerweb" {
  api_endpoint = "https://127.0.0.1:8888"
  # Bearer token Auth
  api_token = var.api_token # If you choose to use Bearer Token configured in your API deployment
  # OR Basic Auth
  api_username = var.api_username # Basic Auth configured in your API deployment.
  api_password = var.api_password # required with api_username to work.
}

data "bunkerweb_jobs" "all" {}

output "job_plugins" {
  value = [for job in data.bunkerweb_jobs.all.jobs : job.plugin]
}
