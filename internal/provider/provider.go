// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	defaultAPIEndpoint    = "https://127.0.0.1:5000/api"
	envAPIEndpoint        = "BUNKERWEB_API_ENDPOINT"
	envAPIToken           = "BUNKERWEB_API_TOKEN"
	defaultRequestTimeout = 30 * time.Second
)

// Ensure BunkerWebProvider satisfies various provider interfaces.
var _ provider.Provider = &BunkerWebProvider{}
var _ provider.ProviderWithFunctions = &BunkerWebProvider{}
var _ provider.ProviderWithEphemeralResources = &BunkerWebProvider{}

// BunkerWebProvider defines the provider implementation.
type BunkerWebProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// BunkerWebProviderModel describes the provider data model.
type BunkerWebProviderModel struct {
	APIEndpoint   types.String `tfsdk:"api_endpoint"`
	APIToken      types.String `tfsdk:"api_token"`
	SkipTLSVerify types.Bool   `tfsdk:"skip_tls_verify"`
}

func (p *BunkerWebProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "bunkerweb"
	resp.Version = p.version
}

func (p *BunkerWebProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_endpoint": schema.StringAttribute{
				MarkdownDescription: "Base URL for the BunkerWeb API. Defaults to `" + defaultAPIEndpoint + "` if neither the attribute nor `" + envAPIEndpoint + "` environment variable are set.",
				Optional:            true,
			},
			"api_token": schema.StringAttribute{
				MarkdownDescription: "API token used to authenticate with BunkerWeb. Can also be provided via the `" + envAPIToken + "` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"skip_tls_verify": schema.BoolAttribute{
				MarkdownDescription: "Disables TLS certificate validation when set to true. Useful for development environments only.",
				Optional:            true,
			},
		},
	}
}

func (p *BunkerWebProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data BunkerWebProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	apiEndpoint := defaultAPIEndpoint
	if !data.APIEndpoint.IsNull() && !data.APIEndpoint.IsUnknown() {
		apiEndpoint = data.APIEndpoint.ValueString()
	} else if envVal := os.Getenv(envAPIEndpoint); envVal != "" {
		apiEndpoint = envVal
	}

	if _, err := url.ParseRequestURI(apiEndpoint); err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_endpoint"),
			"Invalid API Endpoint",
			"Unable to parse the `api_endpoint` value. Ensure it is a valid URL. Error: "+err.Error(),
		)
	}

	skipTLSVerify := false
	if !data.SkipTLSVerify.IsNull() && !data.SkipTLSVerify.IsUnknown() {
		skipTLSVerify = data.SkipTLSVerify.ValueBool()
	}

	apiToken := ""
	if !data.APIToken.IsNull() && !data.APIToken.IsUnknown() {
		apiToken = data.APIToken.ValueString()
	} else if envVal := os.Getenv(envAPIToken); envVal != "" {
		apiToken = envVal
	}

	if apiToken == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token"),
			"Missing API Token",
			"Set the `api_token` attribute or provide the `"+envAPIToken+"` environment variable to authenticate against the BunkerWeb API.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected HTTP Transport Type",
			"http.DefaultTransport is not an *http.Transport; unable to configure custom transport",
		)
		return
	}

	transport := defaultTransport.Clone()
	if skipTLSVerify {
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{}
		}
		transport.TLSClientConfig.InsecureSkipVerify = true
	}

	httpClient := &http.Client{
		Timeout:   defaultRequestTimeout,
		Transport: transport,
	}

	client, err := newBunkerWebClient(apiEndpoint, httpClient, apiToken)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Configure BunkerWeb Client",
			err.Error(),
		)
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
	resp.EphemeralResourceData = client
}

func (p *BunkerWebProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewBunkerWebResource,
		NewBunkerWebInstanceResource,
		NewBunkerWebGlobalConfigResource,
		NewBunkerWebConfigResource,
		NewBunkerWebBanResource,
		NewBunkerWebPluginResource,
	}
}

func (p *BunkerWebProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{
		NewBunkerWebEphemeralResource,
		NewBunkerWebRunJobsEphemeralResource,
		NewBunkerWebInstanceActionEphemeralResource,
		NewBunkerWebServiceConvertEphemeralResource,
		NewBunkerWebConfigUploadEphemeralResource,
		NewBunkerWebConfigUploadUpdateEphemeralResource,
		NewBunkerWebConfigBulkDeleteEphemeralResource,
		NewBunkerWebBanBulkEphemeralResource,
	}
}

func (p *BunkerWebProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewBunkerWebDataSource,
		NewBunkerWebGlobalConfigDataSource,
		NewBunkerWebPluginsDataSource,
		NewBunkerWebCacheDataSource,
		NewBunkerWebJobsDataSource,
		NewBunkerWebConfigsDataSource,
	}
}

func (p *BunkerWebProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{
		NewBunkerWebFunction,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &BunkerWebProvider{
			version: version,
		}
	}
}
