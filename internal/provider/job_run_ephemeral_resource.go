// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ ephemeral.EphemeralResource = &BunkerWebRunJobsEphemeralResource{}

// BunkerWebRunJobsEphemeralResource triggers scheduler jobs during plan/apply.
type BunkerWebRunJobsEphemeralResource struct {
	client *bunkerWebClient
}

// BunkerWebRunJobsEphemeralResourceModel captures Terraform shape.
type BunkerWebRunJobsEphemeralResourceModel struct {
	Jobs []BunkerWebRunJobItem `tfsdk:"jobs"`
}

// BunkerWebRunJobItem describes a single job request.
type BunkerWebRunJobItem struct {
	Plugin types.String `tfsdk:"plugin"`
	Name   types.String `tfsdk:"name"`
}

func NewBunkerWebRunJobsEphemeralResource() ephemeral.EphemeralResource {
	return &BunkerWebRunJobsEphemeralResource{}
}

func (r *BunkerWebRunJobsEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_run_jobs"
}

func (r *BunkerWebRunJobsEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Triggers one or more scheduler jobs via the BunkerWeb API during planning or apply.",
		Attributes: map[string]schema.Attribute{
			"jobs": schema.ListNestedAttribute{
				Required:            true,
				MarkdownDescription: "Jobs to trigger, defined by plugin and optional job name.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"plugin": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Plugin identifier owning the job.",
						},
						"name": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Optional job name; omit to target all jobs exposed by the plugin.",
						},
					},
				},
			},
		},
	}
}

func (r *BunkerWebRunJobsEphemeralResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
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

func (r *BunkerWebRunJobsEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var data BunkerWebRunJobsEphemeralResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(data.Jobs) == 0 {
		resp.Diagnostics.AddAttributeError(path.Root("jobs"), "Missing Jobs", "Provide at least one job to trigger.")
		return
	}

	jobItems, diags := data.toJobItems()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.RunJobs(ctx, jobItems); err != nil {
		resp.Diagnostics.AddError("Run Jobs", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}

func (r *BunkerWebRunJobsEphemeralResource) Close(context.Context, ephemeral.CloseRequest, *ephemeral.CloseResponse) {
	// No follow-up action required.
}

func (m *BunkerWebRunJobsEphemeralResourceModel) toJobItems() ([]JobItem, diag.Diagnostics) {
	var diags diag.Diagnostics

	jobs := make([]JobItem, 0, len(m.Jobs))
	for idx, job := range m.Jobs {
		if job.Plugin.IsNull() || job.Plugin.IsUnknown() || job.Plugin.ValueString() == "" {
			diags.AddAttributeError(path.Root("jobs").AtListIndex(idx).AtName("plugin"), "Missing Plugin", "Each job must include a plugin identifier.")
			continue
		}

		item := JobItem{Plugin: job.Plugin.ValueString()}
		if !job.Name.IsNull() && !job.Name.IsUnknown() {
			name := job.Name.ValueString()
			if name != "" {
				item.Name = &name
			}
		}
		jobs = append(jobs, item)
	}

	if diags.HasError() {
		return nil, diags
	}

	return jobs, diags
}
