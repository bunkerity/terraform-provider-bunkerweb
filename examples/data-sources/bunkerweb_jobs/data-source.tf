provider "bunkerweb" {
  api_endpoint = "https://127.0.0.1:5000/api"
  api_token    = "changeme"
}

data "bunkerweb_jobs" "all" {}

output "job_plugins" {
  value = [for job in data.bunkerweb_jobs.all.jobs : job.plugin]
}
