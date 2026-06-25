schema_version = 1

project {
  license          = "MPL-2.0"
  copyright_holder = "Bunkerity"
  copyright_year   = 2025

  header_ignore = [
    # internal catalog metadata (prose)
    "META.d/**/*.yaml",

    # examples used within documentation (prose)
    "examples/**",

    # GitHub issue template configuration
    ".github/ISSUE_TEMPLATE/*.yml",

    # golangci-lint tooling configuration
    ".golangci.yml",

    # GoReleaser tooling configuration
    ".goreleaser.yml",

    # Files derived from HashiCorp's terraform-provider-scaffolding-framework
    # (MPL-2.0). They carry a dual copyright notice (HashiCorp + Bunkerity) that
    # is maintained by hand, so copywrite must not overwrite their headers.
    "main.go",
    "tools/tools.go",
    "internal/provider/provider.go",
    "internal/provider/provider_test.go",
    "internal/provider/resource.go",
    "internal/provider/resource_test.go",
    "internal/provider/data_source.go",
    "internal/provider/data_source_test.go",
    "internal/provider/function.go",
    "internal/provider/function_test.go",
    "internal/provider/ephemeral_resource.go",
    "internal/provider/ephemeral_resource_test.go",
  ]
}
