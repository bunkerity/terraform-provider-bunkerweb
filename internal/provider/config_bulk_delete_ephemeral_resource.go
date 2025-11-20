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

var _ ephemeral.EphemeralResource = &BunkerWebConfigBulkDeleteEphemeralResource{}

// BunkerWebConfigBulkDeleteEphemeralResource deletes multiple custom configs at once.
type BunkerWebConfigBulkDeleteEphemeralResource struct {
	client *bunkerWebClient
}

// BunkerWebConfigBulkDeleteModel represents the Terraform schema.
type BunkerWebConfigBulkDeleteModel struct {
	Configs []BunkerWebConfigBulkDeleteItem `tfsdk:"configs"`
	Result  types.String                    `tfsdk:"result"`
}

// BunkerWebConfigBulkDeleteItem models a single config identifier.
type BunkerWebConfigBulkDeleteItem struct {
	Service types.String `tfsdk:"service"`
	Type    types.String `tfsdk:"type"`
	Name    types.String `tfsdk:"name"`
}

func NewBunkerWebConfigBulkDeleteEphemeralResource() ephemeral.EphemeralResource {
	return &BunkerWebConfigBulkDeleteEphemeralResource{}
}

func (r *BunkerWebConfigBulkDeleteEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config_bulk_delete"
}

func (r *BunkerWebConfigBulkDeleteEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Deletes multiple custom configurations in a single API call during plan/apply.",
		Attributes: map[string]schema.Attribute{
			"configs": schema.ListNestedAttribute{
				Required:            true,
				MarkdownDescription: "Configurations to delete.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"service": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Service identifier; defaults to `global` when omitted.",
						},
						"type": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Configuration type.",
						},
						"name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Configuration name.",
						},
					},
				},
			},
			"result": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "JSON-encoded payload containing the names of deleted configurations.",
				Sensitive:           true,
			},
		},
	}
}

func (r *BunkerWebConfigBulkDeleteEphemeralResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
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

func (r *BunkerWebConfigBulkDeleteEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var data BunkerWebConfigBulkDeleteModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	keys, diags := data.toConfigKeys()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteConfigs(ctx, keys); err != nil {
		resp.Diagnostics.AddError("Delete Configs", err.Error())
		return
	}

	deleted := make([]map[string]string, 0, len(keys))
	for _, key := range keys {
		service := "global"
		if key.Service != nil && strings.TrimSpace(*key.Service) != "" {
			service = strings.TrimSpace(*key.Service)
		}
		deleted = append(deleted, map[string]string{
			"service": service,
			"type":    key.Type,
			"name":    key.Name,
		})
	}

	encoded, err := encodeResult(map[string]any{"deleted": deleted})
	if err != nil {
		resp.Diagnostics.AddError("Encode Result", err.Error())
		return
	}

	data.Result = types.StringValue(encoded)
	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}

func (r *BunkerWebConfigBulkDeleteEphemeralResource) Close(context.Context, ephemeral.CloseRequest, *ephemeral.CloseResponse) {
	// No clean-up work required.
}

func (m *BunkerWebConfigBulkDeleteModel) toConfigKeys() ([]ConfigKey, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(m.Configs) == 0 {
		diags.AddAttributeError(path.Root("configs"), "Missing Configs", "Provide at least one configuration to delete.")
		return nil, diags
	}

	keys := make([]ConfigKey, 0, len(m.Configs))
	for idx, item := range m.Configs {
		if item.Type.IsNull() || item.Type.IsUnknown() {
			diags.AddAttributeError(path.Root("configs").AtListIndex(idx).AtName("type"), "Missing Type", "Each configuration must include a type.")
		}
		if item.Name.IsNull() || item.Name.IsUnknown() {
			diags.AddAttributeError(path.Root("configs").AtListIndex(idx).AtName("name"), "Missing Name", "Each configuration must include a name.")
		}
		if diags.HasError() {
			continue
		}

		service := normalizeTFService(item.Service)
		keys = append(keys, ConfigKey{
			Service: stringPointer(service, true),
			Type:    strings.TrimSpace(item.Type.ValueString()),
			Name:    strings.TrimSpace(item.Name.ValueString()),
		})
	}

	if diags.HasError() {
		return nil, diags
	}

	return keys, diags
}
