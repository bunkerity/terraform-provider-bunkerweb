# Terraform

## Introduction

The Terraform provider for BunkerWeb allows you to manage your BunkerWeb instances, services, and configurations through Infrastructure as Code (IaC). This provider interacts with the BunkerWeb API to automate the deployment and management of your security configurations.

## Prerequisites

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.12
- A BunkerWeb instance with API enabled
- An API token or basic authentication credentials

## Installation

The provider is available on the [Terraform Registry](https://registry.terraform.io/providers/bunkerity/bunkerweb/latest). Add it to your Terraform configuration:

```terraform
terraform {
  required_providers {
    bunkerweb = {
      source  = "bunkerity/bunkerweb"
      version = "~> 0.0.2"
    }
  }
}
```

## Configuration

### Bearer Token Authentication (recommended)

```terraform
provider "bunkerweb" {
  api_endpoint = "https://bunkerweb.example.com:8888"
  api_token    = var.bunkerweb_token
}
```

### Basic Authentication

```terraform
provider "bunkerweb" {
  api_endpoint = "https://bunkerweb.example.com:8888"
  api_username = var.bunkerweb_username
  api_password = var.bunkerweb_password
}
```

## Usage Examples

### Create a Web Service

```terraform
resource "bunkerweb_service" "app" {
  server_name = "app.example.com"

  variables = {
    upstream = "10.0.0.12:8080"
    mode     = "production"
  }
}
```

### Register an Instance

```terraform
resource "bunkerweb_instance" "worker1" {
  hostname     = "worker-1.internal"
  name         = "Worker 1"
  port         = 8080
  listen_https = true
  https_port   = 8443
  server_name  = "worker-1.internal"
  method       = "api"
}
```

### Configure a Global Setting

```terraform
resource "bunkerweb_global_config_setting" "retry" {
  key   = "retry_limit"
  value = "10"
}
```

### Ban an IP Address

```terraform
resource "bunkerweb_ban" "suspicious_ip" {
  ip       = "192.0.2.100"
  reason   = "Multiple failed login attempts"
  duration = 3600  # 1 hour in seconds
}
```

### Custom Configuration

```terraform
resource "bunkerweb_config" "custom_rules" {
  service_id = "app.example.com"
  type       = "http"
  name       = "custom-rules.conf"
  content    = file("${path.module}/configs/custom-rules.conf")
}
```

## Available Resources

The provider exposes the following resources:

- **bunkerweb_service**: Web service management
- **bunkerweb_instance**: Instance registration and management
- **bunkerweb_global_config_setting**: Global configuration
- **bunkerweb_config**: Custom configurations
- **bunkerweb_ban**: IP banning management
- **bunkerweb_plugin**: Plugin installation and management

## Data Sources

Data sources allow reading existing information:

- **bunkerweb_service**: Read an existing service
- **bunkerweb_global_config**: Read global configuration
- **bunkerweb_plugins**: List available plugins
- **bunkerweb_cache**: Cache information
- **bunkerweb_jobs**: Scheduled jobs status

## Ephemeral Resources

For one-time operations:

- **bunkerweb_run_jobs**: Trigger jobs on demand
- **bunkerweb_instance_action**: Execute actions on instances (reload, stop, etc.)
- **bunkerweb_service_snapshot**: Capture service state
- **bunkerweb_config_upload**: Bulk configuration upload

## Complete Example

Here's an example of a complete infrastructure with BunkerWeb:

```terraform
terraform {
  required_providers {
    bunkerweb = {
      source  = "bunkerity/bunkerweb"
      version = "~> 0.0.1"
    }
  }
}

provider "bunkerweb" {
  api_endpoint = "https://bunkerweb.example.com:8888"
  api_token    = var.bunkerweb_token
}

# Global configuration
resource "bunkerweb_global_config_setting" "rate_limit" {
  key   = "rate_limit"
  value = "10r/s"
}

# Main service
resource "bunkerweb_service" "webapp" {
  server_name = "webapp.example.com"
  
  variables = {
    upstream          = "10.0.1.10:8080"
    mode              = "production"
    auto_lets_encrypt = "yes"
    use_modsecurity   = "yes"
    use_antibot       = "cookie"
  }
}

# API service with different configuration
resource "bunkerweb_service" "api" {
  server_name = "api.example.com"
  
  variables = {
    upstream        = "10.0.1.20:3000"
    mode            = "production"
    use_cors        = "yes"
    cors_allow_origin = "*"
  }
}

# Worker instance
resource "bunkerweb_instance" "worker1" {
  hostname     = "bw-worker-1.internal"
  name         = "Production Worker 1"
  port         = 8080
  listen_https = true
  https_port   = 8443
  server_name  = "bw-worker-1.internal"
  method       = "api"
}

# Custom configuration for webapp service
resource "bunkerweb_config" "custom_security" {
  service_id = bunkerweb_service.webapp.id
  type       = "http"
  name       = "custom-security.conf"
  content    = <<-EOT
    # Custom security headers
    add_header X-Frame-Options "DENY" always;
    add_header X-Content-Type-Options "nosniff" always;
  EOT
}

# Ban a suspicious IP
resource "bunkerweb_ban" "blocked_ip" {
  ip       = "203.0.113.50"
  reason   = "Detected malicious activity"
  duration = 86400  # 24 hours
}

output "webapp_service_id" {
  value = bunkerweb_service.webapp.id
}

output "api_service_id" {
  value = bunkerweb_service.api.id
}
```

## Additional Resources

- [Complete provider documentation](https://registry.terraform.io/providers/bunkerity/bunkerweb/latest/docs)
- [GitHub Repository](https://github.com/bunkerity/terraform-provider-bunkerweb)
- [Usage Examples](https://github.com/bunkerity/terraform-provider-bunkerweb/tree/main/examples)
- [BunkerWeb API Documentation](https://docs.bunkerweb.io/latest/api/)

## Support and Contribution

To report bugs or suggest improvements, visit the [provider's GitHub repository](https://github.com/bunkerity/terraform-provider-bunkerweb/issues).
