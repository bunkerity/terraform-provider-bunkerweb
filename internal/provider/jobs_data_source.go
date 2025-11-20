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

var _ datasource.DataSource = &BunkerWebJobsDataSource{}

// BunkerWebJobsDataSource provides job metadata.
type BunkerWebJobsDataSource struct {
	client *bunkerWebClient
}

// BunkerWebJobsDataSourceModel holds state.
type BunkerWebJobsDataSourceModel struct {
	Jobs types.List `tfsdk:"jobs"`
}

func NewBunkerWebJobsDataSource() datasource.DataSource {
	return &BunkerWebJobsDataSource{}
}

func (d *BunkerWebJobsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_jobs"
}

func (d *BunkerWebJobsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists scheduler jobs known to the BunkerWeb control plane.",
		Attributes: map[string]schema.Attribute{
			"jobs": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Job descriptors reported by the API.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"plugin": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Plugin identifier.",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Job name (when set).",
						},
						"status": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Latest known status from the scheduler.",
						},
						"last_run": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Timestamp of the most recent run if reported.",
						},
					},
				},
			},
		},
	}
}

func (d *BunkerWebJobsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *BunkerWebJobsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	jobs, err := d.client.ListJobs(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to List Jobs", err.Error())
		return
	}

	attrTypes := map[string]attr.Type{
		"plugin":   types.StringType,
		"name":     types.StringType,
		"status":   types.StringType,
		"last_run": types.StringType,
	}

	objs := make([]attr.Value, 0, len(jobs))
	for _, job := range jobs {
		objs = append(objs, types.ObjectValueMust(attrTypes, map[string]attr.Value{
			"plugin":   types.StringValue(job.Plugin),
			"name":     types.StringValue(job.Name),
			"status":   types.StringValue(job.Status),
			"last_run": types.StringValue(job.LastRun),
		}))
	}

	data := BunkerWebJobsDataSourceModel{
		Jobs: types.ListValueMust(types.ObjectType{AttrTypes: attrTypes}, objs),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
