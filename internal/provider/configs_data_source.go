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

var _ datasource.DataSource = &BunkerWebConfigsDataSource{}

// BunkerWebConfigsDataSource lists configuration files managed by BunkerWeb.
type BunkerWebConfigsDataSource struct {
	client *bunkerWebClient
}

// BunkerWebConfigsDataSourceModel represents the data source configuration/state.
type BunkerWebConfigsDataSourceModel struct {
	Service  types.String `tfsdk:"service"`
	Type     types.String `tfsdk:"type"`
	WithData types.Bool   `tfsdk:"with_data"`
	Configs  types.List   `tfsdk:"configs"`
}

func NewBunkerWebConfigsDataSource() datasource.DataSource {
	return &BunkerWebConfigsDataSource{}
}

func (d *BunkerWebConfigsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_configs"
}

func (d *BunkerWebConfigsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists configuration files stored in BunkerWeb for a given service/type pair.",
		Attributes: map[string]schema.Attribute{
			"service": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Target service identifier to filter on. Defaults to the global scope when omitted.",
			},
			"type": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Configuration type filter (for example `http`).",
			},
			"with_data": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "When true, includes the configuration file contents in the response.",
			},
			"configs": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Configurations returned by the API.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"service": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Service scope for the configuration entry (" + "global" + " when not bound to a specific service).",
						},
						"type": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Configuration type segment.",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Configuration file name.",
						},
						"data": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Configuration content when requested via `with_data`.",
							Sensitive:           true,
						},
						"method": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Creation method reported by the API (for example `api`).",
						},
					},
				},
			},
		},
	}
}

func (d *BunkerWebConfigsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *BunkerWebConfigsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var data BunkerWebConfigsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := ConfigListOptions{}
	if !data.Service.IsNull() && !data.Service.IsUnknown() {
		service := data.Service.ValueString()
		opts.Service = &service
	}
	if !data.Type.IsNull() && !data.Type.IsUnknown() {
		cfgType := data.Type.ValueString()
		opts.Type = &cfgType
	}
	if !data.WithData.IsNull() && !data.WithData.IsUnknown() {
		withData := data.WithData.ValueBool()
		opts.WithData = &withData
	}

	configs, err := d.client.ListConfigs(ctx, opts)
	if err != nil {
		resp.Diagnostics.AddError("Unable to List Configs", err.Error())
		return
	}

	elemType := map[string]attr.Type{
		"service": types.StringType,
		"type":    types.StringType,
		"name":    types.StringType,
		"data":    types.StringType,
		"method":  types.StringType,
	}
	elems := make([]attr.Value, 0, len(configs))

	for _, cfg := range configs {
		values := map[string]attr.Value{
			"service": types.StringValue(cfg.Service),
			"type":    types.StringValue(cfg.Type),
			"name":    types.StringValue(cfg.Name),
			"data":    types.StringValue(cfg.Data),
			"method":  types.StringValue(cfg.Method),
		}
		elems = append(elems, types.ObjectValueMust(elemType, values))
	}

	data.Configs = types.ListValueMust(types.ObjectType{AttrTypes: elemType}, elems)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
