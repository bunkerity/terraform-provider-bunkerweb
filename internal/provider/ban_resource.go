// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &BunkerWebBanResource{}
var _ resource.ResourceWithImportState = &BunkerWebBanResource{}

// BunkerWebBanResource models the ban lifecycle via the API.
type BunkerWebBanResource struct {
	client *bunkerWebClient
}

// BunkerWebBanResourceModel carries Terraform state.
type BunkerWebBanResourceModel struct {
	ID                types.String `tfsdk:"id"`
	IP                types.String `tfsdk:"ip"`
	Service           types.String `tfsdk:"service"`
	Reason            types.String `tfsdk:"reason"`
	ExpirationSeconds types.Int64  `tfsdk:"expiration_seconds"`
}

func NewBunkerWebBanResource() resource.Resource {
	return &BunkerWebBanResource{}
}

func (r *BunkerWebBanResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ban"
}

func (r *BunkerWebBanResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a BunkerWeb ban across instances.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Internal identifier composed of ip/service.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ip": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "IPv4/IPv6 address to ban.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Optional service identifier for service-specific bans.",
				Default:             stringdefault.StaticString(""),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"reason": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Reason stored alongside the ban.",
				Default:             stringdefault.StaticString("api"),
			},
			"expiration_seconds": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Ban expiration in seconds. Zero makes the ban permanent.",
				Default:             int64default.StaticInt64(86400),
			},
		},
	}
}

func (r *BunkerWebBanResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BunkerWebBanResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var plan BunkerWebBanResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	banReq := BanRequest{
		IP: plan.IP.ValueString(),
	}

	if !plan.Reason.IsNull() && !plan.Reason.IsUnknown() {
		reason := plan.Reason.ValueString()
		banReq.Reason = &reason
	}
	if !plan.ExpirationSeconds.IsNull() && !plan.ExpirationSeconds.IsUnknown() {
		exp := int(plan.ExpirationSeconds.ValueInt64())
		banReq.Exp = &exp
	}
	if !plan.Service.IsNull() && !plan.Service.IsUnknown() {
		service := strings.TrimSpace(plan.Service.ValueString())
		if service != "" {
			banReq.Service = &service
		}
	}

	if err := r.client.Ban(ctx, banReq); err != nil {
		resp.Diagnostics.AddError("Unable to Create Ban", err.Error())
		return
	}

	resp.Diagnostics.Append(plan.refreshFromAPI(ctx, r.client)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "created bunkerweb ban", map[string]any{"id": plan.ID.ValueString()})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BunkerWebBanResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var state BunkerWebBanResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags := state.refreshFromAPI(ctx, r.client)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BunkerWebBanResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update Not Supported", "BunkerWeb bans cannot be updated in-place; recreate the resource with new arguments.")
}

func (r *BunkerWebBanResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var state BunkerWebBanResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.IP.IsNull() || state.IP.IsUnknown() {
		return
	}

	unbanReq := UnbanRequest{IP: state.IP.ValueString()}
	if !state.Service.IsNull() && !state.Service.IsUnknown() {
		service := strings.TrimSpace(state.Service.ValueString())
		if service != "" {
			unbanReq.Service = &service
		}
	}

	if err := r.client.Unban(ctx, unbanReq); err != nil {
		resp.Diagnostics.AddError("Unable to Delete Ban", err.Error())
		return
	}
}

func (r *BunkerWebBanResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) > 2 {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected ip or ip/service, got %q", req.ID),
		)
		return
	}

	service := ""
	if len(parts) == 2 {
		service = parts[1]
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &BunkerWebBanResourceModel{
		ID:      types.StringValue(buildBanID(parts[0], service)),
		IP:      types.StringValue(parts[0]),
		Service: types.StringValue(service),
	})...)
}

func (m *BunkerWebBanResourceModel) refreshFromAPI(ctx context.Context, client *bunkerWebClient) diag.Diagnostics {
	if m.IP.IsNull() || m.IP.IsUnknown() {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Refresh Ban", "IP must be known")}
	}

	service := ""
	if !m.Service.IsNull() && !m.Service.IsUnknown() {
		service = strings.TrimSpace(m.Service.ValueString())
	}

	bans, err := client.ListBans(ctx)
	if err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("List Bans", err.Error())}
	}

	for _, ban := range bans {
		if ban.IP != m.IP.ValueString() {
			continue
		}
		currentService := ""
		if ban.Service != nil {
			currentService = strings.TrimSpace(*ban.Service)
		}
		if currentService != service {
			continue
		}

		m.ID = types.StringValue(buildBanID(ban.IP, currentService))
		m.IP = types.StringValue(ban.IP)
		m.Service = types.StringValue(currentService)
		if ban.Reason != "" {
			m.Reason = types.StringValue(ban.Reason)
		} else {
			m.Reason = types.StringValue("api")
		}
		m.ExpirationSeconds = types.Int64Value(int64(ban.Exp))
		return nil
	}

	m.ID = types.StringNull()
	return nil
}

func buildBanID(ip, service string) string {
	if service == "" {
		return ip
	}
	return fmt.Sprintf("%s/%s", ip, service)
}
