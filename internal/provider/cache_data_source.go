// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &BunkerWebCacheDataSource{}

// BunkerWebCacheDataSource lists cached job artefacts.
type BunkerWebCacheDataSource struct {
	client *bunkerWebClient
}

// BunkerWebCacheDataSourceModel holds state.
type BunkerWebCacheDataSourceModel struct {
	Service  types.String `tfsdk:"service"`
	Plugin   types.String `tfsdk:"plugin"`
	JobName  types.String `tfsdk:"job_name"`
	WithData types.Bool   `tfsdk:"with_data"`
	Entries  types.List   `tfsdk:"entries"`
}

func NewBunkerWebCacheDataSource() datasource.DataSource {
	return &BunkerWebCacheDataSource{}
}

func (d *BunkerWebCacheDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cache"
}

func (d *BunkerWebCacheDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves cached artefacts produced by BunkerWeb jobs.",
		Attributes: map[string]schema.Attribute{
			"service": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Filter by service identifier (use \"global\" for global cache).",
			},
			"plugin": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Filter by plugin identifier.",
			},
			"job_name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Filter by job name.",
			},
			"with_data": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Include inline file content when true.",
			},
			"entries": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Cache entries that match the filters.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"service": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Service context for the cache file.",
						},
						"plugin": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Owning plugin identifier.",
						},
						"job_name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Job name that produced the cache file.",
						},
						"file_name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Cache file name.",
						},
						"data": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Inline cache contents when requested.",
						},
					},
				},
			},
		},
	}
}

func (d *BunkerWebCacheDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *BunkerWebCacheDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var data BunkerWebCacheDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	filters := url.Values{}
	if !data.Service.IsNull() && !data.Service.IsUnknown() {
		svc := strings.TrimSpace(data.Service.ValueString())
		if svc != "" {
			filters.Set("service", svc)
		}
	}
	if !data.Plugin.IsNull() && !data.Plugin.IsUnknown() {
		plugin := strings.TrimSpace(data.Plugin.ValueString())
		if plugin != "" {
			filters.Set("plugin", plugin)
		}
	}
	if !data.JobName.IsNull() && !data.JobName.IsUnknown() {
		name := strings.TrimSpace(data.JobName.ValueString())
		if name != "" {
			filters.Set("job_name", name)
		}
	}
	withData := false
	if !data.WithData.IsNull() && !data.WithData.IsUnknown() {
		withData = data.WithData.ValueBool()
	}
	if withData {
		filters.Set("with_data", "true")
	}

	entries, err := d.client.ListCacheEntries(ctx, filters)
	if err != nil {
		resp.Diagnostics.AddError("Unable to List Cache Entries", err.Error())
		return
	}

	attrTypes := map[string]attr.Type{
		"service":   types.StringType,
		"plugin":    types.StringType,
		"job_name":  types.StringType,
		"file_name": types.StringType,
		"data":      types.StringType,
	}
	objs := make([]attr.Value, 0, len(entries))
	for _, entry := range entries {
		dataVal := types.StringNull()
		if entry.Data != nil {
			dataVal = types.StringValue(*entry.Data)
		}
		objs = append(objs, types.ObjectValueMust(attrTypes, map[string]attr.Value{
			"service":   types.StringValue(entry.Service),
			"plugin":    types.StringValue(entry.Plugin),
			"job_name":  types.StringValue(entry.JobName),
			"file_name": types.StringValue(entry.FileName),
			"data":      dataVal,
		}))
	}

	data.Entries = types.ListValueMust(types.ObjectType{AttrTypes: attrTypes}, objs)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
