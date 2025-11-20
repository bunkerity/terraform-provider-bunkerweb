// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &BunkerWebResource{}
var _ resource.ResourceWithImportState = &BunkerWebResource{}

func NewBunkerWebResource() resource.Resource {
	return &BunkerWebResource{}
}

// BunkerWebResource represents the bunkerweb_service Terraform resource.
type BunkerWebResource struct {
	client *bunkerWebClient
}

// BunkerWebResourceModel mirrors the Terraform state for bunkerweb_service.
type BunkerWebResourceModel struct {
	ID         types.String `tfsdk:"id"`
	ServerName types.String `tfsdk:"server_name"`
	IsDraft    types.Bool   `tfsdk:"is_draft"`
	Variables  types.Map    `tfsdk:"variables"`
}

func (r *BunkerWebResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *BunkerWebResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a BunkerWeb service via the BunkerWeb API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identifier of the service inside BunkerWeb.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"server_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Server name of the service (first label used as identifier).",
			},
			"is_draft": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "When true, the service stays in draft mode.",
				Default:             booldefault.StaticBool(false),
			},
			"variables": schema.MapAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Additional service variables as key/value pairs.",
			},
		},
	}
}

func (r *BunkerWebResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*bunkerWebClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *bunkerWebClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *BunkerWebResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var plan BunkerWebResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	variables, diags := mapFromTerraform(ctx, plan.Variables)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	service, err := r.client.CreateService(ctx, ServiceCreateRequest{
		ServerName: plan.ServerName.ValueString(),
		IsDraft:    plan.IsDraft.ValueBool(),
		Variables:  variables,
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Service", err.Error())
		return
	}

	populateDiags := plan.populateFromService(ctx, service)
	resp.Diagnostics.Append(populateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "created bunkerweb service", map[string]any{"id": service.ID})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BunkerWebResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var state BunkerWebResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	service, err := r.client.GetService(ctx, state.ID.ValueString())
	if err != nil {
		var apiErr *bunkerWebAPIError
		if errors.As(err, &apiErr) {
			if apiErr.StatusCode == http.StatusNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}

		resp.Diagnostics.AddError("Unable to Read Service", err.Error())
		return
	}

	populateDiags := state.populateFromService(ctx, service)
	resp.Diagnostics.Append(populateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BunkerWebResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var plan BunkerWebResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	variables, diags := mapFromTerraform(ctx, plan.Variables)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverName := plan.ServerName.ValueString()
	isDraft := plan.IsDraft.ValueBool()

	service, err := r.client.UpdateService(ctx, plan.ID.ValueString(), ServiceUpdateRequest{
		ServerName: &serverName,
		IsDraft:    &isDraft,
		Variables:  variables,
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to Update Service", err.Error())
		return
	}

	populateDiags := plan.populateFromService(ctx, service)
	resp.Diagnostics.Append(populateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "updated bunkerweb service", map[string]any{"id": service.ID})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BunkerWebResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var state BunkerWebResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteService(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to Delete Service", err.Error())
	}
}

func (r *BunkerWebResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (m *BunkerWebResourceModel) populateFromService(ctx context.Context, svc *bunkerWebService) diag.Diagnostics {
	var diags diag.Diagnostics

	if svc == nil {
		diags.AddError("Missing Service Data", "Service payload returned by BunkerWeb API was empty")
		return diags
	}

	m.ID = types.StringValue(svc.ID)
	m.ServerName = types.StringValue(svc.ServerName)
	m.IsDraft = types.BoolValue(svc.IsDraft)

	variables, mapDiags := mapToTerraform(ctx, svc.Variables)
	diags.Append(mapDiags...)
	if diags.HasError() {
		return diags
	}

	m.Variables = variables

	return diags
}
