// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ ephemeral.EphemeralResource = &BunkerWebServiceConvertEphemeralResource{}

// BunkerWebServiceConvertEphemeralResource switches a service between draft and online states.
type BunkerWebServiceConvertEphemeralResource struct {
	client *bunkerWebClient
}

// BunkerWebServiceConvertModel captures Terraform-side shape.
type BunkerWebServiceConvertModel struct {
	ServiceID types.String `tfsdk:"service_id"`
	ConvertTo types.String `tfsdk:"convert_to"`
	IsDraft   types.Bool   `tfsdk:"is_draft"`
}

func NewBunkerWebServiceConvertEphemeralResource() ephemeral.EphemeralResource {
	return &BunkerWebServiceConvertEphemeralResource{}
}

func (r *BunkerWebServiceConvertEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_convert"
}

func (r *BunkerWebServiceConvertEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Converts a BunkerWeb service between online and draft states using the `/services/{service}/convert` endpoint.",
		Attributes: map[string]schema.Attribute{
			"service_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Identifier of the service to convert.",
			},
			"convert_to": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Target state: `online` or `draft`.",
			},
			"is_draft": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Draft flag returned by the API after conversion.",
			},
		},
	}
}

func (r *BunkerWebServiceConvertEphemeralResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
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

func (r *BunkerWebServiceConvertEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var data BunkerWebServiceConvertModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ServiceID.IsNull() || data.ServiceID.IsUnknown() || strings.TrimSpace(data.ServiceID.ValueString()) == "" {
		resp.Diagnostics.AddAttributeError(path.Root("service_id"), "Missing Service ID", "Provide a valid service identifier to convert.")
		return
	}

	if data.ConvertTo.IsNull() || data.ConvertTo.IsUnknown() {
		resp.Diagnostics.AddAttributeError(path.Root("convert_to"), "Missing Target", "Set `convert_to` to either online or draft.")
		return
	}

	target := strings.ToLower(strings.TrimSpace(data.ConvertTo.ValueString()))
	if target != "online" && target != "draft" {
		resp.Diagnostics.AddAttributeError(path.Root("convert_to"), "Invalid Target", "`convert_to` must be either online or draft.")
		return
	}

	service, err := r.client.ConvertService(ctx, data.ServiceID.ValueString(), target)
	if err != nil {
		resp.Diagnostics.AddError("Convert Service", err.Error())
		return
	}

	data.IsDraft = types.BoolValue(service.IsDraft)
	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}

func (r *BunkerWebServiceConvertEphemeralResource) Close(context.Context, ephemeral.CloseRequest, *ephemeral.CloseResponse) {
	// No clean-up.
}
