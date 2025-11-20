// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &BunkerWebPluginsDataSource{}

// BunkerWebPluginsDataSource lists installed plugins.
type BunkerWebPluginsDataSource struct {
	client *bunkerWebClient
}

// BunkerWebPluginsDataSourceModel represents the data source state.
type BunkerWebPluginsDataSourceModel struct {
	Type     types.String `tfsdk:"type"`
	WithData types.Bool   `tfsdk:"with_data"`
	Plugins  types.List   `tfsdk:"plugins"`
}

func NewBunkerWebPluginsDataSource() datasource.DataSource {
	return &BunkerWebPluginsDataSource{}
}

func (d *BunkerWebPluginsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plugins"
}

func (d *BunkerWebPluginsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists BunkerWeb UI plugins managed by the control plane.",
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional plugin type filter (\"all\", \"ui\", \"external\", ...).",
			},
			"with_data": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "When true, requests plugin content payloads as well.",
			},
			"plugins": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Plugins returned by the API.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Unique plugin identifier.",
						},
						"type": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Plugin type classification.",
						},
						"version": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Reported plugin version.",
						},
						"description": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Short description if supplied by the API.",
						},
					},
				},
			},
		},
	}
}

func (d *BunkerWebPluginsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *BunkerWebPluginsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var data BunkerWebPluginsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pluginType := ""
	if !data.Type.IsNull() && !data.Type.IsUnknown() {
		pluginType = data.Type.ValueString()
	}

	withData := false
	if !data.WithData.IsNull() && !data.WithData.IsUnknown() {
		withData = data.WithData.ValueBool()
	}

	plugins, err := d.client.ListPlugins(ctx, pluginType, withData)
	if err != nil {
		resp.Diagnostics.AddError("Unable to List Plugins", err.Error())
		return
	}

	elems := make([]attr.Value, 0, len(plugins))
	elemType := map[string]attr.Type{
		"id":          types.StringType,
		"type":        types.StringType,
		"version":     types.StringType,
		"description": types.StringType,
	}

	for _, plugin := range plugins {
		values := map[string]attr.Value{
			"id":          types.StringValue(plugin.ID),
			"type":        types.StringValue(plugin.Type),
			"version":     types.StringValue(plugin.Version),
			"description": types.StringValue(plugin.Description),
		}
		elems = append(elems, types.ObjectValueMust(elemType, values))
	}

	data.Plugins = types.ListValueMust(types.ObjectType{AttrTypes: elemType}, elems)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
