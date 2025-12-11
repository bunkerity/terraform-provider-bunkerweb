# Terraform

## Introducción

El proveedor de Terraform para BunkerWeb le permite administrar sus instancias, servicios y configuraciones de BunkerWeb mediante Infraestructura como Código (IaC). Este proveedor interactúa con la API de BunkerWeb para automatizar el despliegue y la gestión de sus configuraciones de seguridad.

## Requisitos previos

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.12
- Una instancia de BunkerWeb con la API habilitada
- Un token de API o credenciales de autenticación básica

## Instalación

El proveedor está disponible en el [Terraform Registry](https://registry.terraform.io/providers/bunkerity/bunkerweb/latest). Añádalo a su configuración de Terraform:

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

## Configuración

### Autenticación por Bearer Token (recomendada)

```terraform
provider "bunkerweb" {
  api_endpoint = "https://bunkerweb.example.com:8888"
  api_token    = var.bunkerweb_token
}
```

### Autenticación básica

```terraform
provider "bunkerweb" {
  api_endpoint = "https://bunkerweb.example.com:8888"
  api_username = var.bunkerweb_username
  api_password = var.bunkerweb_password
}
```

## Ejemplos de uso

### Crear un servicio web

```terraform
resource "bunkerweb_service" "app" {
  server_name = "app.example.com"

  variables = {
    upstream = "10.0.0.12:8080"
    mode     = "production"
  }
}
```

### Registrar una instancia

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

### Configurar un parámetro global

```terraform
resource "bunkerweb_global_config_setting" "retry" {
  key   = "retry_limit"
  value = "10"
}
```

### Banear una dirección IP

```terraform
resource "bunkerweb_ban" "suspicious_ip" {
  ip       = "192.0.2.100"
  reason   = "Multiple failed login attempts"
  duration = 3600  # 1 hora en segundos
}
```

### Configuración personalizada

```terraform
resource "bunkerweb_config" "custom_rules" {
  service_id = "app.example.com"
  type       = "http"
  name       = "custom-rules.conf"
  content    = file("${path.module}/configs/custom-rules.conf")
}
```

## Recursos disponibles

El proveedor expone los siguientes recursos:

- **bunkerweb_service**: Gestión de servicios web
- **bunkerweb_instance**: Registro y gestión de instancias
- **bunkerweb_global_config_setting**: Configuración global
- **bunkerweb_config**: Configuraciones personalizadas
- **bunkerweb_ban**: Gestión de baneos de IP
- **bunkerweb_plugin**: Instalación y gestión de plugins

## Fuentes de datos

Las fuentes de datos permiten leer información existente:

- **bunkerweb_service**: Leer un servicio existente
- **bunkerweb_global_config**: Leer la configuración global
- **bunkerweb_plugins**: Listar plugins disponibles
- **bunkerweb_cache**: Información de la caché
- **bunkerweb_jobs**: Estado de trabajos programados

## Recursos efímeros

Para operaciones puntuales:

- **bunkerweb_run_jobs**: Desencadenar trabajos bajo demanda
- **bunkerweb_instance_action**: Ejecutar acciones en instancias (reload, stop, etc.)
- **bunkerweb_service_snapshot**: Capturar el estado de un servicio
- **bunkerweb_config_upload**: Carga masiva de configuraciones

## Ejemplo completo

Aquí hay un ejemplo de una infraestructura completa con BunkerWeb:

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

# Configuración global
resource "bunkerweb_global_config_setting" "rate_limit" {
  key   = "rate_limit"
  value = "10r/s"
}

# Servicio principal
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

# Servicio API con configuración diferente
resource "bunkerweb_service" "api" {
  server_name = "api.example.com"
  
  variables = {
    upstream        = "10.0.1.20:3000"
    mode            = "production"
    use_cors        = "yes"
    cors_allow_origin = "*"
  }
}

# Instancia worker
resource "bunkerweb_instance" "worker1" {
  hostname     = "bw-worker-1.internal"
  name         = "Production Worker 1"
  port         = 8080
  listen_https = true
  https_port   = 8443
  server_name  = "bw-worker-1.internal"
  method       = "api"
}

# Configuración personalizada para el servicio webapp
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

# Banear una IP sospechosa
resource "bunkerweb_ban" "blocked_ip" {
  ip       = "203.0.113.50"
  reason   = "Detected malicious activity"
  duration = 86400  # 24 horas
}

output "webapp_service_id" {
  value = bunkerweb_service.webapp.id
}

output "api_service_id" {
  value = bunkerweb_service.api.id
}
```

## Recursos adicionales

- [Documentación completa del proveedor](https://registry.terraform.io/providers/bunkerity/bunkerweb/latest/docs)
- [Repositorio GitHub](https://github.com/bunkerity/terraform-provider-bunkerweb)
- [Ejemplos de uso](https://github.com/bunkerity/terraform-provider-bunkerweb/tree/main/examples)
- [Documentación de la API de BunkerWeb](https://docs.bunkerweb.io/latest/api/)

## Soporte y contribución

Para reportar errores o sugerir mejoras, visite el [repositorio GitHub del proveedor](https://github.com/bunkerity/terraform-provider-bunkerweb/issues).
