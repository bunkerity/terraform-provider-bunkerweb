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

var _ ephemeral.EphemeralResource = &BunkerWebConfigUploadEphemeralResource{}

// BunkerWebConfigUploadEphemeralResource uploads multiple custom config files.
type BunkerWebConfigUploadEphemeralResource struct {
	client *bunkerWebClient
}

// BunkerWebConfigUploadEphemeralResourceModel captures Terraform input/result fields.
type BunkerWebConfigUploadEphemeralResourceModel struct {
	Service types.String                     `tfsdk:"service"`
	Type    types.String                     `tfsdk:"type"`
	Files   []BunkerWebConfigUploadFileModel `tfsdk:"files"`
	Result  types.String                     `tfsdk:"result"`
}

// BunkerWebConfigUploadFileModel represents a single upload file entry.
type BunkerWebConfigUploadFileModel struct {
	Name    types.String `tfsdk:"name"`
	Content types.String `tfsdk:"content"`
}

func NewBunkerWebConfigUploadEphemeralResource() ephemeral.EphemeralResource {
	return &BunkerWebConfigUploadEphemeralResource{}
}

func (r *BunkerWebConfigUploadEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config_upload"
}

func (r *BunkerWebConfigUploadEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Uploads one or more custom configuration files via the BunkerWeb API during plan/apply.",
		Attributes: map[string]schema.Attribute{
			"service": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Target service identifier; defaults to `global` when omitted.",
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Configuration type (e.g. `http`, `stream`).",
			},
			"files": schema.ListNestedAttribute{
				Required:            true,
				MarkdownDescription: "Files to upload. Names are sanitized by the API to match BunkerWeb naming rules.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "File name associated with the upload part.",
						},
						"content": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "File content to send. Use Terraform functions like `file()` or `filebase64decode()` as needed.",
							Sensitive:           true,
						},
					},
				},
			},
			"result": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "JSON-encoded response payload describing the uploaded configs.",
				Sensitive:           true,
			},
		},
	}
}

func (r *BunkerWebConfigUploadEphemeralResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
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

func (r *BunkerWebConfigUploadEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var data BunkerWebConfigUploadEphemeralResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uploadReq, diags := data.toUploadRequest()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	configs, err := r.client.UploadConfigs(ctx, uploadReq)
	if err != nil {
		resp.Diagnostics.AddError("Upload Configs", err.Error())
		return
	}

	encoded, err := encodeResult(configs)
	if err != nil {
		resp.Diagnostics.AddError("Encode Result", err.Error())
		return
	}

	data.Result = types.StringValue(encoded)
	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}

func (r *BunkerWebConfigUploadEphemeralResource) Close(context.Context, ephemeral.CloseRequest, *ephemeral.CloseResponse) {
	// No follow-up required.
}

func (m *BunkerWebConfigUploadEphemeralResourceModel) toUploadRequest() (ConfigUploadRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	if m.Type.IsNull() || m.Type.IsUnknown() {
		diags.AddAttributeError(path.Root("type"), "Missing Type", "Provide the configuration type to upload files against.")
	}

	if len(m.Files) == 0 {
		diags.AddAttributeError(path.Root("files"), "Missing Files", "Provide at least one file entry to upload.")
	}

	if diags.HasError() {
		return ConfigUploadRequest{}, diags
	}

	service := normalizeTFService(m.Service)
	if strings.EqualFold(service, "global") {
		service = ""
	}

	files := make([]ConfigUploadFile, 0, len(m.Files))
	for idx, file := range m.Files {
		if file.Name.IsNull() || file.Name.IsUnknown() {
			diags.AddAttributeError(path.Root("files").AtListIndex(idx).AtName("name"), "Missing Name", "Each file requires a name value.")
			continue
		}
		name := strings.TrimSpace(file.Name.ValueString())
		if name == "" {
			diags.AddAttributeError(path.Root("files").AtListIndex(idx).AtName("name"), "Invalid Name", "File name cannot be empty or whitespace.")
			continue
		}

		if file.Content.IsNull() || file.Content.IsUnknown() {
			diags.AddAttributeError(path.Root("files").AtListIndex(idx).AtName("content"), "Missing Content", "Each file requires content to upload.")
			continue
		}

		content := file.Content.ValueString()
		files = append(files, ConfigUploadFile{FileName: name, Content: []byte(content)})
	}

	if diags.HasError() {
		return ConfigUploadRequest{}, diags
	}

	return ConfigUploadRequest{
		Service: service,
		Type:    strings.TrimSpace(m.Type.ValueString()),
		Files:   files,
	}, diags
}
