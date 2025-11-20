terraform {
  required_providers {
    bunkerweb = {
      source  = "local/bunkerity/bunkerweb"
      version = "0.1.0"
    }
  }
}

provider "bunkerweb" {
  api_endpoint = var.api_endpoint
  api_token    = var.api_token
}

variable "api_endpoint" {
  type        = string
  description = "BunkerWeb API base URL, e.g. https://stack.example.com/api"
  default     = "http://127.0.0.1:8888"
}

variable "api_token" {
  type        = string
  description = "API token or Biscuit for BunkerWeb"
  sensitive   = true
}

variable "server_name" {
  type        = string
  description = "Server name for the BunkerWeb service"
}

variable "service_is_draft" {
  type        = bool
  description = "Whether to mark the service as draft"
  default     = false
}

variable "service_variables" {
  type        = map(string)
  description = "Variables to set on the service"
  default     = {}
}

resource "bunkerweb_service" "example" {
  server_name = var.server_name
  is_draft    = var.service_is_draft
  variables   = var.service_variables
}

output "bunkerweb_api_endpoint" {
  value = var.api_endpoint
}

output "bunkerweb_service_id" {
  value = bunkerweb_service.example.id
}
