# Publish guide

This document walks through the full process required to publish the Terraform provider `bunkerity/bunkerweb` to the public Terraform Registry and explains how the existing GitHub workflows automate most steps.

> Replace every `X.Y.Z` example with the target semantic version (for example `0.2.0`).

## 1. Prerequisites

- **Registry namespace**: the GitHub organisation (or user) `bunkerity` must be approved by HashiCorp. Follow the official guide: https://developer.hashicorp.com/terraform/registry/providers/publishing#publish-a-provider. Keep handy the SVG logo, repository URL, and contact email that the registry UI asks for.
- **GPG keys**: generate a signing key and store it as GitHub secrets:
  - `GPG_PRIVATE_KEY` containing the ASCII-armored private key (base64 if needed)
  - `PASSPHRASE` holding the key passphrase
- **Repository permissions**: the `Release` workflow triggers whenever a tag matching `v*` is pushed. Make sure you can push to the repository and create tags.
- **Local tooling (optional but recommended for validation)**:
  - Go >= 1.23
  - Terraform CLI >= 1.13 (1.14 once ephemeral resources are GA)
  - `goreleaser` if you want to dry-run builds locally

## 2. Pre-release checklist

1. **Sync the main branch**
   ```bash
   git checkout main
   git pull --ff-only origin main
   ```
2. **Regenerate docs and examples**
   ```bash
   make generate
   git status  # should be clean
   ```
3. **Run tests**
   ```bash
   go test ./...
   TF_ACC=1 go test ./internal/provider -count=1
   ```
   - Acceptance tests require a reachable BunkerWeb API (for example http://127.0.0.1:8888) plus `BUNKERWEB_API_ENDPOINT` and `BUNKERWEB_API_TOKEN` if you use environment variables.
   - When ephemeral resources depend on a specific Terraform CLI version, run the suite with that version explicitly.
4. **Update release notes**: refresh `CHANGELOG.md` (and README badges if they reference the version).
5. **Optional local release rehearsal**:
   ```bash
   goreleaser release --snapshot --skip-publish --clean
   ```

## 3. Tag and push

1. Pick the new semantic version.
2. Adjust version strings in documentation if required.
3. Commit the release prep changes:
   ```bash
   git commit -am "Prepare release vX.Y.Z"
   ```
4. Create and push the annotated tag:
   ```bash
   git tag -a vX.Y.Z -m "Release vX.Y.Z"
   git push origin main
   git push origin vX.Y.Z
   ```

## 4. GitHub Release workflow

The workflow `.github/workflows/release.yml` executes automatically after the tag push:

1. Check out the repository (full history for tag awareness).
2. Install Go using the version in `go.mod`.
3. Import the GPG key via `crazy-max/ghaction-import-gpg`.
4. Run `goreleaser release --clean`, which builds and signs the provider according to `.goreleaser.yml`:
   - ZIP archives for each GOOS/GOARCH pair
   - SHA256 checksum file plus detached signature
   - `terraform-registry-manifest.json`
5. Upload the artifacts to the GitHub release associated with the tag.

Confirm in the **Actions** tab that the `goreleaser` job succeeded. The release page should contain:
- `terraform-provider-bunkerweb_vX.Y.Z_<os>_<arch>.zip`
- `terraform-provider-bunkerweb_vX.Y.Z_SHA256SUMS`
- `terraform-provider-bunkerweb_vX.Y.Z_SHA256SUMS.sig`
- `terraform-provider-bunkerweb_vX.Y.Z_manifest.json`

## 5. Terraform Registry publication

1. After the GitHub release completes, the Terraform Registry crawler will fetch the assets (allow 5–15 minutes).
2. Visit https://registry.terraform.io/providers/bunkerity/bunkerweb/latest to confirm the new version appears.
3. Smoke test with Terraform CLI:
   ```bash
   mkdir -p /tmp/tf-provider-test && cd /tmp/tf-provider-test
   cat <<'EOF' > main.tf
   terraform {
     required_providers {
       bunkerweb = {
         source  = "bunkerity/bunkerweb"
         version = "= X.Y.Z"
       }
     }
   }

   provider "bunkerweb" {}
   EOF

   terraform init
   ```
   Terraform should download the new provider version from the registry.

## 6. Automation plumbing

### Existing workflows

- `test.yml`: lint, doc generation, and acceptance tests across several Terraform versions. Update the matrix (`matrix.terraform`) to include the versions you officially support (for example 1.5, 1.6, 1.14 once ephemeral resources graduate).
- `release.yml`: builds signed release artifacts on every `v*` tag.
- `issue-comment-triage.yml` and `lock.yml`: automatic community maintenance.

### Suggested improvements

- Add a pre-release test job inside `release.yml` that re-runs `go test ./...` and `TF_ACC=1 ...` using either the fake API or a containerized instance.
- Keep the Terraform matrix in `test.yml` current with the versions required to exercise ephemeral resources.
- If you want automatic release notes, set `changelog.disable: false` in `.goreleaser.yml` or supply a changelog template.

## 7. Hotfixes and rollbacks

- **Hotfix**: repeat the entire workflow using a new tag `vX.Y.Z+1`, ensuring the changelog highlights the fix.
- **Rollback**: delete the GitHub release and coordinate with HashiCorp support if you must yank a published version from the registry.

## 8. Quick recap

1. Refresh docs and run all tests (`make generate`, `go test`, `TF_ACC=1`).
2. Update changelog and commit the release prep.
3. Tag `vX.Y.Z` and push branch + tag.
4. Let GitHub Actions run GoReleaser and upload signed artifacts.
5. Verify the GitHub release and the Terraform Registry listing.
6. Announce the release (changelog, README badges, etc.).

Following this procedure keeps every provider release reproducible, tested, and automatically distributed through the official Terraform Registry.
