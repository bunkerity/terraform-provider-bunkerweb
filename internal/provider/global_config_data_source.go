// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &BunkerWebGlobalConfigDataSource{}

func NewBunkerWebGlobalConfigDataSource() datasource.DataSource {
	return &BunkerWebGlobalConfigDataSource{}
}

type BunkerWebGlobalConfigDataSource struct {
	client *bunkerWebClient
}

type BunkerWebGlobalConfigDataSourceModel struct {
	Full     types.Bool `tfsdk:"full"`
	Settings types.Map  `tfsdk:"settings"`
}

func (d *BunkerWebGlobalConfigDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_global_config"
}

func (d *BunkerWebGlobalConfigDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches the global configuration maintained by the BunkerWeb control plane.",
		Attributes: map[string]schema.Attribute{
			"full": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "When true, include settings that currently hold their default values.",
			},
			"settings": schema.MapAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				MarkdownDescription: "Key/value pairs representing the global configuration. Complex values are JSON encoded.",
			},
		},
	}
}

func (d *BunkerWebGlobalConfigDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *BunkerWebGlobalConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var data BunkerWebGlobalConfigDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	full := true
	if !data.Full.IsNull() && !data.Full.IsUnknown() {
		full = data.Full.ValueBool()
	}

	settings, err := d.client.GetGlobalConfig(ctx, full, false)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Global Config", err.Error())
		return
	}

	stringified := map[string]string{}
	for key, value := range settings {
		stringified[key] = stringifyValue(value)
	}

	value, diag := types.MapValueFrom(ctx, types.StringType, stringified)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Settings = value

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func stringifyValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	case fmt.Stringer:
		return v.String()
	case nil:
		return ""
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%v", v)
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(encoded)
	}
}
