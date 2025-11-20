// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &BunkerWebDataSource{}

func NewBunkerWebDataSource() datasource.DataSource {
	return &BunkerWebDataSource{}
}

// BunkerWebDataSource defines the data source implementation.
type BunkerWebDataSource struct {
	client *bunkerWebClient
}

// BunkerWebDataSourceModel describes the data source data model.
type BunkerWebDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	ServerName types.String `tfsdk:"server_name"`
	IsDraft    types.Bool   `tfsdk:"is_draft"`
	Variables  types.Map    `tfsdk:"variables"`
}

func (d *BunkerWebDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (d *BunkerWebDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves an existing BunkerWeb service from the BunkerWeb API.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Identifier of the service to look up.",
			},
			"server_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Server name of the service.",
			},
			"is_draft": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the service is still a draft.",
			},
			"variables": schema.MapAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				MarkdownDescription: "Service variables as key/value pairs.",
			},
		},
	}
}

func (d *BunkerWebDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*bunkerWebClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *bunkerWebClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *BunkerWebDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var data BunkerWebDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	service, err := d.client.GetService(ctx, data.ID.ValueString())
	if err != nil {
		var apiErr *bunkerWebAPIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			resp.Diagnostics.AddError("Service Not Found", fmt.Sprintf("No service found with id %q", data.ID.ValueString()))
			return
		}

		resp.Diagnostics.AddError("Unable to Read Service", err.Error())
		return
	}

	populateDiags := data.populateFromService(ctx, service)
	resp.Diagnostics.Append(populateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (m *BunkerWebDataSourceModel) populateFromService(ctx context.Context, svc *bunkerWebService) diag.Diagnostics {
	var diags diag.Diagnostics

	if svc == nil {
		diags.AddError("Missing Service Data", "Service payload returned by BunkerWeb API was empty")
		return diags
	}

	m.ID = types.StringValue(svc.ID)
	m.ServerName = types.StringValue(svc.ServerName)
	m.IsDraft = types.BoolValue(svc.IsDraft)

	variables, mapDiags := mapToTerraform(ctx, svc.Variables)
	diags.Append(mapDiags...)
	if diags.HasError() {
		return diags
	}

	m.Variables = variables

	return diags
}
