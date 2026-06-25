# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Terraform provider for [BunkerWeb](https://www.bunkerweb.io/) (open-source WAF). Manages BunkerWeb
through its HTTP API. Built on the **Terraform Plugin Framework** (`terraform-plugin-framework`),
NOT the legacy SDKv2 — use framework idioms (`types.String`, `resource.Resource`, `Schema`,
plan modifiers), never `schema.Resource`/`d.Get`.

- Go >= 1.24, Terraform >= 1.8 (>= 1.14 to exercise ephemeral resources).
- Provider address: `registry.terraform.io/bunkerity/bunkerweb`.
- All implementation lives in one package: `internal/provider/`. Entrypoint: `main.go`.

## Commands

Targets live in `GNUmakefile`:

```shell
make              # fmt + lint + install + generate (the full local pass)
make build        # go build -v ./...
make install      # go install -v ./... -> $GOBIN
make fmt          # gofmt -s -w -e .
make lint         # golangci-lint run
make test         # unit tests: go test -v -cover -timeout=120s -parallel=10 ./...
make testacc      # TF_ACC=1 go test -v -cover -timeout 120m ./...
make generate     # regenerate docs/ (see Docs section)
```

Run a single test:

```shell
go test -v -run TestAccBunkerWebResource ./internal/provider
```

Acceptance tests (`*_test.go`, gated by `TF_ACC=1`) run against an **in-memory fake API** in
`internal/provider/test_server_test.go` — no live BunkerWeb needed. When you add behavior, extend
that fake server to match the real API's envelope and routes.

## Local manual testing

Build, drop the binary into the local plugin dir, then run Terraform in `test-local/`:

```shell
go build -o terraform-provider-bunkerweb
cp terraform-provider-bunkerweb ~/.terraform.d/plugins/local/bunkerity/bunkerweb/0.0.1/linux_amd64/terraform-provider-bunkerweb_v0.0.1
cd test-local && terraform init && terraform plan
```

`test-local/test-provider.sh` automates this against a live BunkerWeb API.

## Architecture

Flow: provider config → one shared `*bunkerWebClient` → injected into every resource/data
source/ephemeral → typed CRUD calls against the BunkerWeb API.

- **`provider.go`** — `BunkerWebProvider` defines provider schema (`api_endpoint`, `api_token`,
  `api_username`/`api_password`, `skip_tls_verify`; each falls back to `BUNKERWEB_API_*` env vars).
  `Configure()` builds the client and hands it out via `resp.ResourceData`, `resp.DataSourceData`,
  and `resp.EphemeralResourceData`. `Resources()`/`DataSources()`/`EphemeralResources()`/`Functions()`
  register every type — a new type MUST be added to the matching list here or it won't load.

- **`client.go`** — `bunkerWebClient` wraps all HTTP. Auth: Bearer `api_token`, or Basic auth
  exchanged for a token via `Login()`. Every response is a `bunkerWebAPIEnvelope`
  (`{status, message, data}`); non-2xx or non-ok status becomes a typed `*bunkerWebAPIError`
  (carries `StatusCode`). Each endpoint has request/response structs (e.g. `ServiceCreateRequest`,
  `bunkerWebService`). Add new API calls as methods here, returning typed structs.

- **Resources** (`resource.go` = `bunkerweb_service`, plus `instance_resource.go`,
  `config_resource.go`, `ban_resource.go`, `plugin_resource.go`, `global_config_resource.go`)
  share one shape:
  - a `...Model` struct with `tfsdk:"..."` tags,
  - `Metadata`/`Schema`/`Configure`/`Create`/`Read`/`Update`/`Delete`/`ImportState`,
  - a `populateFrom<Type>()` method mapping an API struct onto the model,
  - `Read`/etc. treat a 404 (`errors.As(err, &apiErr)` + `StatusCode == http.StatusNotFound`)
    by calling `resp.State.RemoveResource(ctx)` rather than erroring.
  - immutable fields use `RequiresReplace()`; computed-stable fields use `UseStateForUnknown()`.

- **Data sources** (`data_source.go`, `*_data_source.go`) — same shape minus mutation; `Read` only.

- **Ephemeral resources** (`ephemeral_resource.go`, `*_ephemeral_resource.go`) — implement
  `ephemeral.EphemeralResource`; used for actions/snapshots/bulk ops that must NOT persist to state.

- **`terraform_conversion.go`** — shared converters: `mapFromTerraform`/`mapToTerraform` for
  `types.Map` ↔ `map[string]string`. Nullable model fields ↔ Go pointers go through
  `optionalString`/`optionalInt`/`optionalBool` (in `instance_resource.go`). Reuse these instead
  of re-deriving null/unknown handling.

To add a resource: implement the file in `internal/provider/`, register it in `provider.go`,
add an example under `examples/resources/<name>/resource.tf`, extend `test_server_test.go` and a
`*_test.go`, then `make generate`.

## Docs are generated — do not hand-edit `docs/`

`docs/` is produced by `terraform-plugin-docs` (see `tools/tools.go`) from the code schemas,
`examples/`, and `templates/*.tmpl`. After any schema/example change run `make generate`
(`gofmt`, `terraform fmt` on examples, `tfplugindocs`, and copywrite headers all run). Edit
`templates/` and `examples/`, never `docs/` directly. CI (`.github/workflows/test.yml`) runs
lint + `make generate` + acceptance tests across a Terraform version matrix, so a stale `docs/`
or unformatted code fails CI.
