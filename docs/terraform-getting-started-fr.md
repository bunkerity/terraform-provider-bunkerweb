# Terraform

## Introduction

Le provider Terraform pour BunkerWeb vous permet de gérer vos instances, services et configurations BunkerWeb via l'Infrastructure as Code (IaC). Ce provider interagit avec l'API BunkerWeb pour automatiser le déploiement et la gestion de vos configurations de sécurité.

## Prérequis

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.12
- Une instance BunkerWeb avec l'API activée
- Un token d'API ou des identifiants d'authentification basique

## Installation

Le provider est disponible sur le [Terraform Registry](https://registry.terraform.io/providers/bunkerity/bunkerweb/latest). Ajoutez-le à votre configuration Terraform :

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

### Authentification par Bearer token (recommandée)

```terraform
provider "bunkerweb" {
  api_endpoint = "https://bunkerweb.example.com:8888"
  api_token    = var.bunkerweb_token
}
```

### Authentification basique

```terraform
provider "bunkerweb" {
  api_endpoint = "https://bunkerweb.example.com:8888"
  api_username = var.bunkerweb_username
  api_password = var.bunkerweb_password
}
```

## Exemples d'utilisation

### Créer un service web

```terraform
resource "bunkerweb_service" "app" {
  server_name = "app.example.com"

  variables = {
    upstream = "10.0.0.12:8080"
    mode     = "production"
  }
}
```

### Enregistrer une instance

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

### Configurer un paramètre global

```terraform
resource "bunkerweb_global_config_setting" "retry" {
  key   = "retry_limit"
  value = "10"
}
```

### Bannir une adresse IP

```terraform
resource "bunkerweb_ban" "suspicious_ip" {
  ip       = "192.0.2.100"
  reason   = "Multiple failed login attempts"
  duration = 3600  # 1 heure en secondes
}
```

### Configuration personnalisée

```terraform
resource "bunkerweb_config" "custom_rules" {
  service_id = "app.example.com"
  type       = "http"
  name       = "custom-rules.conf"
  content    = file("${path.module}/configs/custom-rules.conf")
}
```

## Ressources disponibles

Le provider expose les ressources suivantes :

- **bunkerweb_service** : Gestion des services web
- **bunkerweb_instance** : Enregistrement et gestion des instances
- **bunkerweb_global_config_setting** : Configuration globale
- **bunkerweb_config** : Configurations personnalisées
- **bunkerweb_ban** : Gestion des bannissements IP
- **bunkerweb_plugin** : Installation et gestion des plugins

## Data sources

Les data sources permettent de lire des informations existantes :

- **bunkerweb_service** : Lecture d'un service existant
- **bunkerweb_global_config** : Lecture de la configuration globale
- **bunkerweb_plugins** : Liste des plugins disponibles
- **bunkerweb_cache** : Informations sur le cache
- **bunkerweb_jobs** : État des jobs planifiés

## Ressources éphémères

Pour des opérations ponctuelles :

- **bunkerweb_run_jobs** : Déclencher des jobs à la demande
- **bunkerweb_instance_action** : Exécuter des actions sur les instances (reload, stop, etc.)
- **bunkerweb_service_snapshot** : Capturer l'état d'un service
- **bunkerweb_config_upload** : Upload en masse de configurations

## Exemple complet

Voici un exemple d'infrastructure complète avec BunkerWeb :

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

# Configuration globale
resource "bunkerweb_global_config_setting" "rate_limit" {
  key   = "rate_limit"
  value = "10r/s"
}

# Service principal
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

# Service API avec configuration différente
resource "bunkerweb_service" "api" {
  server_name = "api.example.com"
  
  variables = {
    upstream        = "10.0.1.20:3000"
    mode            = "production"
    use_cors        = "yes"
    cors_allow_origin = "*"
  }
}

# Instance worker
resource "bunkerweb_instance" "worker1" {
  hostname     = "bw-worker-1.internal"
  name         = "Production Worker 1"
  port         = 8080
  listen_https = true
  https_port   = 8443
  server_name  = "bw-worker-1.internal"
  method       = "api"
}

# Configuration personnalisée pour le service webapp
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

# Bannir une IP suspecte
resource "bunkerweb_ban" "blocked_ip" {
  ip       = "203.0.113.50"
  reason   = "Detected malicious activity"
  duration = 86400  # 24 heures
}

output "webapp_service_id" {
  value = bunkerweb_service.webapp.id
}

output "api_service_id" {
  value = bunkerweb_service.api.id
}
```

## Ressources additionnelles

- [Documentation complète du provider](https://registry.terraform.io/providers/bunkerity/bunkerweb/latest/docs)
- [Dépôt GitHub](https://github.com/bunkerity/terraform-provider-bunkerweb)
- [Exemples d'utilisation](https://github.com/bunkerity/terraform-provider-bunkerweb/tree/main/examples)
- [Documentation de l'API BunkerWeb](https://docs.bunkerweb.io/latest/api/)

## Support et contribution

Pour signaler des bugs ou proposer des améliorations, rendez-vous sur le [dépôt GitHub du provider](https://github.com/bunkerity/terraform-provider-bunkerweb/issues).
