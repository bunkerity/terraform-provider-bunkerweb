// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &BunkerWebConfigResource{}
var _ resource.ResourceWithImportState = &BunkerWebConfigResource{}

// BunkerWebConfigResource manages API-driven custom configurations.
type BunkerWebConfigResource struct {
	client *bunkerWebClient
}

// BunkerWebConfigResourceModel is the Terraform state.
type BunkerWebConfigResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Service types.String `tfsdk:"service"`
	Type    types.String `tfsdk:"type"`
	Name    types.String `tfsdk:"name"`
	Data    types.String `tfsdk:"data"`
	Method  types.String `tfsdk:"method"`
}

func NewBunkerWebConfigResource() resource.Resource {
	return &BunkerWebConfigResource{}
}

func (r *BunkerWebConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config"
}

func (r *BunkerWebConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a BunkerWeb custom configuration snippet created via the API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Internal identifier composed of service/type/name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Service identifier this config belongs to. Defaults to `global`.",
				Default:             stringdefault.StaticString("global"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Configuration type, e.g. `http`, `server_http`, or `modsec`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Stable configuration name (^[\\w_-]{1,64}$).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"data": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Configuration content as UTF-8 text.",
			},
			"method": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Source method reported by the API.",
			},
		},
	}
}

func (r *BunkerWebConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BunkerWebConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var plan BunkerWebConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	service := normalizeTFService(plan.Service)
	cfg, err := r.client.CreateConfig(ctx, ConfigCreateRequest{
		Service: stringPointer(service),
		Type:    plan.Type.ValueString(),
		Name:    plan.Name.ValueString(),
		Data:    plan.Data.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Config", err.Error())
		return
	}

	resp.Diagnostics.Append(plan.populateFromConfig(cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "created bunkerweb config", map[string]any{"id": plan.ID.ValueString()})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BunkerWebConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var state BunkerWebConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, diags := state.toConfigKey()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.GetConfig(ctx, key, true)
	if err != nil {
		var apiErr *bunkerWebAPIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Unable to Read Config", err.Error())
		return
	}

	resp.Diagnostics.Append(state.populateFromConfig(cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BunkerWebConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var plan BunkerWebConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, diags := plan.toConfigKey()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data := plan.Data.ValueString()

	cfg, err := r.client.UpdateConfig(ctx, key, ConfigUpdateRequest{Data: &data})
	if err != nil {
		resp.Diagnostics.AddError("Unable to Update Config", err.Error())
		return
	}

	resp.Diagnostics.Append(plan.populateFromConfig(cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BunkerWebConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var state BunkerWebConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, diags := state.toConfigKey()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteConfig(ctx, key); err != nil {
		resp.Diagnostics.AddError("Unable to Delete Config", err.Error())
		return
	}
}

func (r *BunkerWebConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected identifier in the form service/type/name, got %q", req.ID),
		)
		return
	}

	service := parts[0]
	if service == "" {
		service = "global"
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &BunkerWebConfigResourceModel{
		ID:      types.StringValue(buildConfigID(service, parts[1], parts[2])),
		Service: types.StringValue(service),
		Type:    types.StringValue(parts[1]),
		Name:    types.StringValue(parts[2]),
	})...)
}

func (m *BunkerWebConfigResourceModel) populateFromConfig(cfg *bunkerWebConfig) diag.Diagnostics {
	if cfg == nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Populate Config", "received nil config")}
	}

	service := cfg.Service
	if service == "" {
		service = "global"
	}

	m.ID = types.StringValue(buildConfigID(service, cfg.Type, cfg.Name))
	m.Service = types.StringValue(service)
	m.Type = types.StringValue(cfg.Type)
	m.Name = types.StringValue(cfg.Name)
	m.Data = types.StringValue(cfg.Data)
	if cfg.Method != "" {
		m.Method = types.StringValue(cfg.Method)
	} else {
		m.Method = types.StringNull()
	}

	return nil
}

func (m *BunkerWebConfigResourceModel) toConfigKey() (ConfigKey, diag.Diagnostics) {
	var diags diag.Diagnostics

	if m.Service.IsNull() || m.Service.IsUnknown() {
		diags.AddAttributeError(path.Root("service"), "Missing Service", "Service must be known to address the config")
	}
	if m.Type.IsNull() || m.Type.IsUnknown() {
		diags.AddAttributeError(path.Root("type"), "Missing Type", "Type must be known to address the config")
	}
	if m.Name.IsNull() || m.Name.IsUnknown() {
		diags.AddAttributeError(path.Root("name"), "Missing Name", "Name must be known to address the config")
	}

	if diags.HasError() {
		return ConfigKey{}, diags
	}

	service := normalizeTFService(m.Service)

	return ConfigKey{
		Service: stringPointer(service),
		Type:    m.Type.ValueString(),
		Name:    m.Name.ValueString(),
	}, diags
}

func normalizeTFService(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return "global"
	}
	trimmed := strings.TrimSpace(v.ValueString())
	if trimmed == "" {
		return "global"
	}
	return trimmed
}

func buildConfigID(service, cfgType, name string) string {
	return fmt.Sprintf("%s/%s/%s", service, cfgType, name)
}

func stringPointer(value string) *string {
	if strings.EqualFold(value, "global") {
		return nil
	}
	v := value
	return &v
}
