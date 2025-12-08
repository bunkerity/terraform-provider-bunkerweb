# Terraform Provider for BunkerWeb

This repository contains the Terraform provider that manages [BunkerWeb](https://www.bunkerweb.io/) services through the BunkerWeb HTTP API. The provider is implemented with the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework) and exposes the core building blocks needed to model BunkerWeb workloads in code:

- `bunkerweb_service` resource for creating, updating, and deleting services.
- `bunkerweb_instance` resource for registering and managing control-plane instances.
- `bunkerweb_global_config_setting` resource for enforcing individual control-plane settings.
- `bunkerweb_config` resource for authoring API-managed configuration snippets.
- `bunkerweb_ban` resource for orchestrating bans across instances.
- `bunkerweb_service` data source for reading existing services.
- `bunkerweb_global_config` data source for inspecting control-plane defaults.
- `bunkerweb_plugins`, `bunkerweb_cache`, and `bunkerweb_jobs` data sources for observing UI plugins, cached artefacts, and scheduled jobs.
- `bunkerweb_service_snapshot` ephemeral resource for capturing service state during a plan.
- `bunkerweb_run_jobs` ephemeral resource for triggering scheduler jobs on demand.
- `bunkerweb_instance_action` ephemeral resource for pinging, reloading, stopping, or deleting instances.
- `bunkerweb_service_convert` ephemeral resource for toggling services between draft and online states.
- `bunkerweb_config_upload`, `bunkerweb_config_upload_update`, and `bunkerweb_config_bulk_delete` ephemerals for batch config uploads, file-based edits, and clean-up operations.
- `provider::bunkerweb::service_identifier` function that normalizes server names into API identifiers.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.8
- [Go](https://go.dev/dl/) >= 1.24 (for building from source)

## Quick Start

```terraform
provider "bunkerweb" {
	api_endpoint = "https://127.0.0.1:5000/api"
	api_token    = var.bunkerweb_token
}

resource "bunkerweb_service" "app" {
	server_name = "app.example.com"

	variables = {
		upstream = "10.0.0.12"
		mode     = "production"
	}
}
```

Refer to the `examples/` directory and the generated docs in `docs/` for additional usage patterns.

## Building the Provider

```shell
go install ./...
```

The compiled provider binary will be placed in `$GOBIN` (`$GOPATH/bin` by default).

### Building and Testing Locally

For local development and testing against a live BunkerWeb instance:

```shell
# Quick start - automated workflow
cd test-local
./test-provider.sh

# Or manually
go build -o terraform-provider-bunkerweb
cp terraform-provider-bunkerweb ~/.terraform.d/plugins/local/bunkerity/bunkerweb/0.0.1/linux_amd64/terraform-provider-bunkerweb_v0.0.1
cd test-local
terraform init
terraform plan
```

See the [`test-local/`](test-local/) directory for comprehensive integration tests that validate all provider functionality against a live BunkerWeb API.

## Testing

Run the full unit test suite with:

```shell
go test ./...
```

Acceptance-style tests exercise the provider against a local in-memory API defined in `internal/provider/test_server_test.go`, so they are safe to run without contacting a live BunkerWeb instance.

### Integration Testing

The `test-local/` directory contains a comprehensive test suite with 22 tests covering:
- All data sources (global_config, plugins, jobs, cache, service, configs)
- All resources (instance, service, global_config_setting, config, ban, plugin)
- All ephemeral resources (snapshots, actions, conversions, uploads, jobs)
- All functions (service_identifier)

Quick reference: `cd test-local && ./quick-ref.sh`

## Generating Documentation

The repository uses [terraform-plugin-docs](https://github.com/hashicorp/terraform-plugin-docs). Regenerate documentation after schema changes:

```shell
make generate
```

## Contributing

Issues and pull requests are welcome. Please ensure new changes include appropriate unit tests and updates to the docs/examples when applicable.
