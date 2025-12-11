# Terraform

## Einführung

Der Terraform-Provider für BunkerWeb ermöglicht es Ihnen, Ihre BunkerWeb-Instanzen, -Dienste und -Konfigurationen über Infrastructure as Code (IaC) zu verwalten. Dieser Provider interagiert mit der BunkerWeb-API, um die Bereitstellung und Verwaltung Ihrer Sicherheitskonfigurationen zu automatisieren.

## Voraussetzungen

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.12
- Eine BunkerWeb-Instanz mit aktivierter API
- Ein API-Token oder Basic-Authentication-Anmeldedaten

## Installation

Der Provider ist im [Terraform Registry](https://registry.terraform.io/providers/bunkerity/bunkerweb/latest) verfügbar. Fügen Sie ihn zu Ihrer Terraform-Konfiguration hinzu:

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

## Konfiguration

### Bearer-Token-Authentifizierung (empfohlen)

```terraform
provider "bunkerweb" {
  api_endpoint = "https://bunkerweb.example.com:8888"
  api_token    = var.bunkerweb_token
}
```

### Basis-Authentifizierung

```terraform
provider "bunkerweb" {
  api_endpoint = "https://bunkerweb.example.com:8888"
  api_username = var.bunkerweb_username
  api_password = var.bunkerweb_password
}
```

## Verwendungsbeispiele

### Einen Webdienst erstellen

```terraform
resource "bunkerweb_service" "app" {
  server_name = "app.example.com"

  variables = {
    upstream = "10.0.0.12:8080"
    mode     = "production"
  }
}
```

### Eine Instanz registrieren

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

### Eine globale Einstellung konfigurieren

```terraform
resource "bunkerweb_global_config_setting" "retry" {
  key   = "retry_limit"
  value = "10"
}
```

### Eine IP-Adresse sperren

```terraform
resource "bunkerweb_ban" "suspicious_ip" {
  ip       = "192.0.2.100"
  reason   = "Multiple failed login attempts"
  duration = 3600  # 1 Stunde in Sekunden
}
```

### Benutzerdefinierte Konfiguration

```terraform
resource "bunkerweb_config" "custom_rules" {
  service_id = "app.example.com"
  type       = "http"
  name       = "custom-rules.conf"
  content    = file("${path.module}/configs/custom-rules.conf")
}
```

## Verfügbare Ressourcen

Der Provider stellt die folgenden Ressourcen bereit:

- **bunkerweb_service**: Webdienst-Verwaltung
- **bunkerweb_instance**: Instanz-Registrierung und -Verwaltung
- **bunkerweb_global_config_setting**: Globale Konfiguration
- **bunkerweb_config**: Benutzerdefinierte Konfigurationen
- **bunkerweb_ban**: IP-Sperrverwaltung
- **bunkerweb_plugin**: Plugin-Installation und -Verwaltung

## Datenquellen

Datenquellen ermöglichen das Lesen vorhandener Informationen:

- **bunkerweb_service**: Einen vorhandenen Dienst lesen
- **bunkerweb_global_config**: Globale Konfiguration lesen
- **bunkerweb_plugins**: Verfügbare Plugins auflisten
- **bunkerweb_cache**: Cache-Informationen
- **bunkerweb_jobs**: Status geplanter Jobs

## Ephemere Ressourcen

Für einmalige Operationen:

- **bunkerweb_run_jobs**: Jobs bei Bedarf auslösen
- **bunkerweb_instance_action**: Aktionen auf Instanzen ausführen (reload, stop, etc.)
- **bunkerweb_service_snapshot**: Dienstzustand erfassen
- **bunkerweb_config_upload**: Massen-Konfiguration hochladen

## Vollständiges Beispiel

Hier ist ein Beispiel einer vollständigen Infrastruktur mit BunkerWeb:

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

# Globale Konfiguration
resource "bunkerweb_global_config_setting" "rate_limit" {
  key   = "rate_limit"
  value = "10r/s"
}

# Hauptdienst
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

# API-Dienst mit unterschiedlicher Konfiguration
resource "bunkerweb_service" "api" {
  server_name = "api.example.com"
  
  variables = {
    upstream        = "10.0.1.20:3000"
    mode            = "production"
    use_cors        = "yes"
    cors_allow_origin = "*"
  }
}

# Worker-Instanz
resource "bunkerweb_instance" "worker1" {
  hostname     = "bw-worker-1.internal"
  name         = "Production Worker 1"
  port         = 8080
  listen_https = true
  https_port   = 8443
  server_name  = "bw-worker-1.internal"
  method       = "api"
}

# Benutzerdefinierte Konfiguration für webapp-Dienst
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

# Eine verdächtige IP sperren
resource "bunkerweb_ban" "blocked_ip" {
  ip       = "203.0.113.50"
  reason   = "Detected malicious activity"
  duration = 86400  # 24 Stunden
}

output "webapp_service_id" {
  value = bunkerweb_service.webapp.id
}

output "api_service_id" {
  value = bunkerweb_service.api.id
}
```

## Zusätzliche Ressourcen

- [Vollständige Provider-Dokumentation](https://registry.terraform.io/providers/bunkerity/bunkerweb/latest/docs)
- [GitHub-Repository](https://github.com/bunkerity/terraform-provider-bunkerweb)
- [Verwendungsbeispiele](https://github.com/bunkerity/terraform-provider-bunkerweb/tree/main/examples)
- [BunkerWeb-API-Dokumentation](https://docs.bunkerweb.io/latest/api/)

## Support und Beiträge

Um Fehler zu melden oder Verbesserungen vorzuschlagen, besuchen Sie das [GitHub-Repository des Providers](https://github.com/bunkerity/terraform-provider-bunkerweb/issues).
