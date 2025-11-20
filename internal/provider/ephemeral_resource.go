// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ ephemeral.EphemeralResource = &BunkerWebEphemeralResource{}

func NewBunkerWebEphemeralResource() ephemeral.EphemeralResource {
	return &BunkerWebEphemeralResource{}
}

// BunkerWebEphemeralResource defines the ephemeral resource implementation.
type BunkerWebEphemeralResource struct {
	client *bunkerWebClient
}

// BunkerWebEphemeralResourceModel describes the ephemeral resource data model.
type BunkerWebEphemeralResourceModel struct {
	ServiceID  types.String `tfsdk:"service_id"`
	ServerName types.String `tfsdk:"server_name"`
	IsDraft    types.Bool   `tfsdk:"is_draft"`
	Variables  types.Map    `tfsdk:"variables"`
}

func (r *BunkerWebEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_snapshot"
}

func (r *BunkerWebEphemeralResource) Schema(ctx context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Captures a point-in-time snapshot of a BunkerWeb service by reading it from the API.",

		Attributes: map[string]schema.Attribute{
			"service_id": schema.StringAttribute{
				MarkdownDescription: "Identifier of the service to read.",
				Required:            true,
			},
			"server_name": schema.StringAttribute{
				MarkdownDescription: "Server name reported by the BunkerWeb API.",
				Computed:            true,
			},
			"is_draft": schema.BoolAttribute{
				MarkdownDescription: "Whether the service is still marked as a draft.",
				Computed:            true,
			},
			"variables": schema.MapAttribute{
				MarkdownDescription: "Service variables returned by the API.",
				ElementType:         types.StringType,
				Computed:            true,
			},
		},
	}
}

func (r *BunkerWebEphemeralResource) Configure(ctx context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*bunkerWebClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Ephemeral Resource Configure Type",
			fmt.Sprintf("Expected *bunkerWebClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *BunkerWebEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var data BunkerWebEphemeralResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ServiceID.IsUnknown() || data.ServiceID.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("service_id"),
			"Missing Service ID",
			"Set the `service_id` attribute to an existing BunkerWeb service identifier before opening the ephemeral resource.",
		)
		return
	}

	service, err := r.client.GetService(ctx, data.ServiceID.ValueString())
	if err != nil {
		var apiErr *bunkerWebAPIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			resp.Diagnostics.AddAttributeError(path.Root("service_id"), "Service Not Found", fmt.Sprintf("No service found with id %q", data.ServiceID.ValueString()))
			return
		}

		resp.Diagnostics.AddError("Unable to Read Service", err.Error())
		return
	}

	resp.Diagnostics.Append(populateEphemeralFromService(ctx, &data, service)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}

func populateEphemeralFromService(ctx context.Context, model *BunkerWebEphemeralResourceModel, svc *bunkerWebService) diag.Diagnostics {
	var diags diag.Diagnostics

	if svc == nil {
		diags.AddError("Missing Service Data", "Service payload returned by BunkerWeb API was empty")
		return diags
	}

	model.ServiceID = types.StringValue(svc.ID)
	model.ServerName = types.StringValue(svc.ServerName)
	model.IsDraft = types.BoolValue(svc.IsDraft)

	variables, mapDiags := mapToTerraform(ctx, svc.Variables)
	diags.Append(mapDiags...)
	if diags.HasError() {
		return diags
	}

	model.Variables = variables

	return diags
}
