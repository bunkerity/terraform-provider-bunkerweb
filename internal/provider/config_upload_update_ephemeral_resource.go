// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ ephemeral.EphemeralResource = &BunkerWebConfigUploadUpdateEphemeralResource{}

// BunkerWebConfigUploadUpdateEphemeralResource updates an existing config with multipart upload semantics.
type BunkerWebConfigUploadUpdateEphemeralResource struct {
	client *bunkerWebClient
}

// BunkerWebConfigUploadUpdateModel describes the Terraform schema.
type BunkerWebConfigUploadUpdateModel struct {
	Service    types.String `tfsdk:"service"`
	Type       types.String `tfsdk:"type"`
	Name       types.String `tfsdk:"name"`
	FileName   types.String `tfsdk:"file_name"`
	Content    types.String `tfsdk:"content"`
	NewService types.String `tfsdk:"new_service"`
	NewType    types.String `tfsdk:"new_type"`
	NewName    types.String `tfsdk:"new_name"`
	Result     types.String `tfsdk:"result"`
}

func NewBunkerWebConfigUploadUpdateEphemeralResource() ephemeral.EphemeralResource {
	return &BunkerWebConfigUploadUpdateEphemeralResource{}
}

func (r *BunkerWebConfigUploadUpdateEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config_upload_update"
}

func (r *BunkerWebConfigUploadUpdateEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Updates an existing custom configuration by uploading file content, optionally renaming or moving it.",
		Attributes: map[string]schema.Attribute{
			"service": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Current service identifier; defaults to `global` when omitted.",
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Current configuration type.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Current configuration name.",
			},
			"file_name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "File name used for the upload part. Defaults to the current configuration name.",
			},
			"content": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "File content to upload.",
				Sensitive:           true,
			},
			"new_service": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional service to move the configuration into.",
			},
			"new_type": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional new configuration type.",
			},
			"new_name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional new configuration name.",
			},
			"result": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "JSON-encoded response payload returned by the API.",
				Sensitive:           true,
			},
		},
	}
}

func (r *BunkerWebConfigUploadUpdateEphemeralResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
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

func (r *BunkerWebConfigUploadUpdateEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var data BunkerWebConfigUploadUpdateModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, updateReq, diags := data.toUploadUpdateRequest()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.client.UpdateConfigFromUpload(ctx, key, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Update Config From Upload", err.Error())
		return
	}

	encoded, err := encodeResult(config)
	if err != nil {
		resp.Diagnostics.AddError("Encode Result", err.Error())
		return
	}

	data.Result = types.StringValue(encoded)
	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}

func (r *BunkerWebConfigUploadUpdateEphemeralResource) Close(context.Context, ephemeral.CloseRequest, *ephemeral.CloseResponse) {
	// No-op.
}

func (m *BunkerWebConfigUploadUpdateModel) toUploadUpdateRequest() (ConfigKey, ConfigUploadUpdateRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	if m.Type.IsNull() || m.Type.IsUnknown() {
		diags.AddAttributeError(path.Root("type"), "Missing Type", "Provide the current configuration type.")
	}
	if m.Name.IsNull() || m.Name.IsUnknown() {
		diags.AddAttributeError(path.Root("name"), "Missing Name", "Provide the current configuration name.")
	}
	if m.Content.IsNull() || m.Content.IsUnknown() {
		diags.AddAttributeError(path.Root("content"), "Missing Content", "Provide file content to upload.")
	}

	if diags.HasError() {
		return ConfigKey{}, ConfigUploadUpdateRequest{}, diags
	}

	service := normalizeTFService(m.Service)
	key := ConfigKey{
		Service: stringPointer(service, true),
		Type:    strings.TrimSpace(m.Type.ValueString()),
		Name:    strings.TrimSpace(m.Name.ValueString()),
	}

	filename := strings.TrimSpace(m.Name.ValueString())
	if !m.FileName.IsNull() && !m.FileName.IsUnknown() {
		value := strings.TrimSpace(m.FileName.ValueString())
		if value == "" {
			diags.AddAttributeError(path.Root("file_name"), "Invalid File Name", "When set, file_name must be non-empty.")
			return ConfigKey{}, ConfigUploadUpdateRequest{}, diags
		}
		filename = value
	}

	req := ConfigUploadUpdateRequest{
		FileName: filename,
		Content:  []byte(m.Content.ValueString()),
	}

	if !m.NewService.IsNull() && !m.NewService.IsUnknown() {
		value := strings.TrimSpace(m.NewService.ValueString())
		if value == "" {
			empty := ""
			req.NewService = &empty
		} else {
			req.NewService = &value
		}
	}

	if !m.NewType.IsNull() && !m.NewType.IsUnknown() {
		value := strings.TrimSpace(m.NewType.ValueString())
		if value != "" {
			req.NewType = &value
		}
	}

	if !m.NewName.IsNull() && !m.NewName.IsUnknown() {
		value := strings.TrimSpace(m.NewName.ValueString())
		if value != "" {
			req.NewName = &value
		}
	}

	return key, req, diags
}
