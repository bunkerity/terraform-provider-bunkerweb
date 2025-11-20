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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &BunkerWebInstanceResource{}
var _ resource.ResourceWithImportState = &BunkerWebInstanceResource{}

func NewBunkerWebInstanceResource() resource.Resource {
	return &BunkerWebInstanceResource{}
}

// BunkerWebInstanceResource represents the bunkerweb_instance Terraform resource.
type BunkerWebInstanceResource struct {
	client *bunkerWebClient
}

type BunkerWebInstanceResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Hostname    types.String `tfsdk:"hostname"`
	Name        types.String `tfsdk:"name"`
	Port        types.Int64  `tfsdk:"port"`
	ListenHTTPS types.Bool   `tfsdk:"listen_https"`
	HTTPSPort   types.Int64  `tfsdk:"https_port"`
	ServerName  types.String `tfsdk:"server_name"`
	Method      types.String `tfsdk:"method"`
}

func (r *BunkerWebInstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (r *BunkerWebInstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a BunkerWeb instance registered with the BunkerWeb API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identifier of the instance (hostname).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"hostname": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Hostname of the BunkerWeb instance.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Friendly display name for the instance.",
			},
			"port": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "HTTP port exposed by the instance API.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"listen_https": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the instance API listens over HTTPS.",
			},
			"https_port": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "HTTPS port exposed by the instance API.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"server_name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Server name used by the instance API when making requests.",
			},
			"method": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Method tag describing how the instance was registered.",
			},
		},
	}
}

func (r *BunkerWebInstanceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BunkerWebInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var plan BunkerWebInstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := InstanceCreateRequest{
		Hostname:    plan.Hostname.ValueString(),
		Name:        optionalString(plan.Name),
		Port:        optionalInt(plan.Port),
		ListenHTTPS: optionalBool(plan.ListenHTTPS),
		HTTPSPort:   optionalInt(plan.HTTPSPort),
		ServerName:  optionalString(plan.ServerName),
		Method:      optionalString(plan.Method),
	}

	instance, err := r.client.CreateInstance(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Instance", err.Error())
		return
	}

	diags := plan.populateFromInstance(instance)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BunkerWebInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var state BunkerWebInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.client.GetInstance(ctx, state.ID.ValueString())
	if err != nil {
		var apiErr *bunkerWebAPIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Unable to Read Instance", err.Error())
		return
	}

	diags := state.populateFromInstance(instance)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BunkerWebInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var plan BunkerWebInstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := InstanceUpdateRequest{
		Name:        optionalString(plan.Name),
		Port:        optionalInt(plan.Port),
		ListenHTTPS: optionalBool(plan.ListenHTTPS),
		HTTPSPort:   optionalInt(plan.HTTPSPort),
		ServerName:  optionalString(plan.ServerName),
		Method:      optionalString(plan.Method),
	}

	instance, err := r.client.UpdateInstance(ctx, plan.ID.ValueString(), request)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Update Instance", err.Error())
		return
	}

	diags := plan.populateFromInstance(instance)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BunkerWebInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var state BunkerWebInstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteInstance(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to Delete Instance", err.Error())
	}
}

func (r *BunkerWebInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (m *BunkerWebInstanceResourceModel) populateFromInstance(instance *bunkerWebInstance) diag.Diagnostics {
	var diags diag.Diagnostics

	if instance == nil {
		diags.AddError("Missing Instance Data", "Instance payload returned by BunkerWeb API was empty")
		return diags
	}

	m.ID = types.StringValue(instance.Hostname)
	m.Hostname = types.StringValue(instance.Hostname)

	if instance.Name != nil {
		m.Name = types.StringValue(*instance.Name)
	} else {
		m.Name = types.StringNull()
	}

	if instance.Port != nil {
		m.Port = types.Int64Value(int64(*instance.Port))
	} else {
		m.Port = types.Int64Null()
	}

	if instance.ListenHTTPS != nil {
		m.ListenHTTPS = types.BoolValue(*instance.ListenHTTPS)
	} else {
		m.ListenHTTPS = types.BoolNull()
	}

	if instance.HTTPSPort != nil {
		m.HTTPSPort = types.Int64Value(int64(*instance.HTTPSPort))
	} else {
		m.HTTPSPort = types.Int64Null()
	}

	if instance.ServerName != nil {
		m.ServerName = types.StringValue(*instance.ServerName)
	} else {
		m.ServerName = types.StringNull()
	}

	if instance.Method != nil {
		m.Method = types.StringValue(*instance.Method)
	} else {
		m.Method = types.StringNull()
	}

	return diags
}

func optionalString(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	v := value.ValueString()
	return &v
}

func optionalInt(value types.Int64) *int {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	v := int(value.ValueInt64())
	return &v
}

func optionalBool(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	v := value.ValueBool()
	return &v
}
