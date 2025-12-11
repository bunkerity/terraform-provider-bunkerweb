# Terraform

## 简介

BunkerWeb 的 Terraform Provider 允许您通过基础设施即代码（IaC）管理 BunkerWeb 实例、服务和配置。该 Provider 与 BunkerWeb API 交互,自动化部署和管理您的安全配置。

## 先决条件

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.12
- 已启用 API 的 BunkerWeb 实例
- API 令牌或基本身份验证凭据

## 安装

该 Provider 可在 [Terraform Registry](https://registry.terraform.io/providers/bunkerity/bunkerweb/latest) 上获得。将其添加到您的 Terraform 配置中:

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

## 配置

### Bearer Token 身份验证(推荐)

```terraform
provider "bunkerweb" {
  api_endpoint = "https://bunkerweb.example.com:8888"
  api_token    = var.bunkerweb_token
}
```

### 基本身份验证

```terraform
provider "bunkerweb" {
  api_endpoint = "https://bunkerweb.example.com:8888"
  api_username = var.bunkerweb_username
  api_password = var.bunkerweb_password
}
```

## 使用示例

### 创建 Web 服务

```terraform
resource "bunkerweb_service" "app" {
  server_name = "app.example.com"

  variables = {
    upstream = "10.0.0.12:8080"
    mode     = "production"
  }
}
```

### 注册实例

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

### 配置全局设置

```terraform
resource "bunkerweb_global_config_setting" "retry" {
  key   = "retry_limit"
  value = "10"
}
```

### 封禁 IP 地址

```terraform
resource "bunkerweb_ban" "suspicious_ip" {
  ip       = "192.0.2.100"
  reason   = "Multiple failed login attempts"
  duration = 3600  # 1小时(秒)
}
```

### 自定义配置

```terraform
resource "bunkerweb_config" "custom_rules" {
  service_id = "app.example.com"
  type       = "http"
  name       = "custom-rules.conf"
  content    = file("${path.module}/configs/custom-rules.conf")
}
```

## 可用资源

该 Provider 提供以下资源:

- **bunkerweb_service**: Web 服务管理
- **bunkerweb_instance**: 实例注册和管理
- **bunkerweb_global_config_setting**: 全局配置
- **bunkerweb_config**: 自定义配置
- **bunkerweb_ban**: IP 封禁管理
- **bunkerweb_plugin**: 插件安装和管理

## 数据源

数据源允许读取现有信息:

- **bunkerweb_service**: 读取现有服务
- **bunkerweb_global_config**: 读取全局配置
- **bunkerweb_plugins**: 列出可用插件
- **bunkerweb_cache**: 缓存信息
- **bunkerweb_jobs**: 计划作业状态

## 临时资源

用于一次性操作:

- **bunkerweb_run_jobs**: 按需触发作业
- **bunkerweb_instance_action**: 在实例上执行操作(重新加载、停止等)
- **bunkerweb_service_snapshot**: 捕获服务状态
- **bunkerweb_config_upload**: 批量配置上传

## 完整示例

以下是使用 BunkerWeb 的完整基础设施示例:

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

# 全局配置
resource "bunkerweb_global_config_setting" "rate_limit" {
  key   = "rate_limit"
  value = "10r/s"
}

# 主服务
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

# 具有不同配置的 API 服务
resource "bunkerweb_service" "api" {
  server_name = "api.example.com"
  
  variables = {
    upstream        = "10.0.1.20:3000"
    mode            = "production"
    use_cors        = "yes"
    cors_allow_origin = "*"
  }
}

# Worker 实例
resource "bunkerweb_instance" "worker1" {
  hostname     = "bw-worker-1.internal"
  name         = "Production Worker 1"
  port         = 8080
  listen_https = true
  https_port   = 8443
  server_name  = "bw-worker-1.internal"
  method       = "api"
}

# webapp 服务的自定义配置
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

# 封禁可疑 IP
resource "bunkerweb_ban" "blocked_ip" {
  ip       = "203.0.113.50"
  reason   = "Detected malicious activity"
  duration = 86400  # 24小时
}

output "webapp_service_id" {
  value = bunkerweb_service.webapp.id
}

output "api_service_id" {
  value = bunkerweb_service.api.id
}
```

## 其他资源

- [完整 Provider 文档](https://registry.terraform.io/providers/bunkerity/bunkerweb/latest/docs)
- [GitHub 仓库](https://github.com/bunkerity/terraform-provider-bunkerweb)
- [使用示例](https://github.com/bunkerity/terraform-provider-bunkerweb/tree/main/examples)
- [BunkerWeb API 文档](https://docs.bunkerweb.io/latest/api/)

## 支持和贡献

要报告错误或提出改进建议,请访问 [Provider 的 GitHub 仓库](https://github.com/bunkerity/terraform-provider-bunkerweb/issues)。
