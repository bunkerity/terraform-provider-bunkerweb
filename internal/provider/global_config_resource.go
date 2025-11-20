// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &BunkerWebGlobalConfigResource{}
var _ resource.ResourceWithImportState = &BunkerWebGlobalConfigResource{}

// BunkerWebGlobalConfigResource reconciles individual global configuration keys.
type BunkerWebGlobalConfigResource struct {
	client *bunkerWebClient
}

// BunkerWebGlobalConfigResourceModel models Terraform state for a single setting.
type BunkerWebGlobalConfigResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Key       types.String `tfsdk:"key"`
	Value     types.String `tfsdk:"value"`
	ValueJSON types.String `tfsdk:"value_json"`
}

func NewBunkerWebGlobalConfigResource() resource.Resource {
	return &BunkerWebGlobalConfigResource{}
}

func (r *BunkerWebGlobalConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_global_config_setting"
}

func (r *BunkerWebGlobalConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a single key within the BunkerWeb global configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Internal identifier that matches the configuration key.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the global configuration setting to manage.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Scalar value as a string. Booleans and numbers are parsed automatically.",
			},
			"value_json": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Raw JSON payload for complex values. Use `jsonencode(...)` to build this string.",
			},
		},
	}
}

func (r *BunkerWebGlobalConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BunkerWebGlobalConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var plan BunkerWebGlobalConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, payload, preferJSON, diags := plan.toPatchPayload()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateGlobalConfig(ctx, payload)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Update Global Config", err.Error())
		return
	}

	value, ok := updated[key]
	if !ok {
		resp.Diagnostics.AddError("Global Config Response Missing Key", fmt.Sprintf("The API response did not include key %q", key))
		return
	}

	plan.ID = types.StringValue(key)
	plan.Key = types.StringValue(key)
	resp.Diagnostics.Append(plan.setStateValueFromAPI(value, preferJSON)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "applied bunkerweb global config setting", map[string]any{"key": key})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BunkerWebGlobalConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var state BunkerWebGlobalConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Key.IsNull() || state.Key.IsUnknown() {
		resp.Diagnostics.AddError("Missing Key", "Resource state is missing the global configuration key.")
		return
	}

	key := strings.TrimSpace(state.Key.ValueString())
	if key == "" {
		resp.Diagnostics.AddError("Invalid Key", "Global configuration key may not be empty.")
		return
	}

	settings, err := r.client.GetGlobalConfig(ctx, true, false)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Global Config", err.Error())
		return
	}

	value, ok := settings[key]
	if !ok || value == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	preferJSON := false
	if !state.ValueJSON.IsNull() && !state.ValueJSON.IsUnknown() {
		preferJSON = true
	}

	state.ID = types.StringValue(key)
	state.Key = types.StringValue(key)
	resp.Diagnostics.Append(state.setStateValueFromAPI(value, preferJSON)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BunkerWebGlobalConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var plan BunkerWebGlobalConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, payload, preferJSON, diags := plan.toPatchPayload()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateGlobalConfig(ctx, payload)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Update Global Config", err.Error())
		return
	}

	value, ok := updated[key]
	if !ok {
		resp.Diagnostics.AddError("Global Config Response Missing Key", fmt.Sprintf("The API response did not include key %q", key))
		return
	}

	plan.ID = types.StringValue(key)
	plan.Key = types.StringValue(key)
	resp.Diagnostics.Append(plan.setStateValueFromAPI(value, preferJSON)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BunkerWebGlobalConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var state BunkerWebGlobalConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Key.IsNull() || state.Key.IsUnknown() {
		return
	}

	key := strings.TrimSpace(state.Key.ValueString())
	if key == "" {
		return
	}

	if _, err := r.client.UpdateGlobalConfig(ctx, map[string]any{key: nil}); err != nil {
		resp.Diagnostics.AddError("Unable to Reset Global Config", err.Error())
		return
	}
}

func (r *BunkerWebGlobalConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	key := strings.TrimSpace(req.ID)
	if key == "" {
		resp.Diagnostics.AddError("Invalid Import Identifier", "Expected a non-empty global configuration key.")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &BunkerWebGlobalConfigResourceModel{
		ID:  types.StringValue(key),
		Key: types.StringValue(key),
	})...)
}

func (m *BunkerWebGlobalConfigResourceModel) toPatchPayload() (string, map[string]any, bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	if m.Key.IsNull() || m.Key.IsUnknown() {
		diags.AddAttributeError(path.Root("key"), "Missing Key", "Key must be provided to manage a global configuration setting.")
		return "", nil, false, diags
	}

	key := strings.TrimSpace(m.Key.ValueString())
	if key == "" {
		diags.AddAttributeError(path.Root("key"), "Invalid Key", "Key cannot be empty or whitespace.")
		return "", nil, false, diags
	}

	hasValue := !m.Value.IsNull() && !m.Value.IsUnknown()
	hasJSON := !m.ValueJSON.IsNull() && !m.ValueJSON.IsUnknown()

	if hasValue && hasJSON {
		diags.AddError("Conflicting Attributes", "Specify only one of value or value_json.")
		return "", nil, false, diags
	}
	if !hasValue && !hasJSON {
		diags.AddAttributeError(path.Root("value"), "Missing Value", "Provide either value or value_json to update the setting.")
		return "", nil, false, diags
	}

	if hasJSON {
		raw := m.ValueJSON.ValueString()
		var decoded any
		if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
			diags.AddAttributeError(path.Root("value_json"), "Invalid JSON", fmt.Sprintf("Unable to decode value_json: %v", err))
			return "", nil, false, diags
		}
		return key, map[string]any{key: decoded}, true, diags
	}

	parsed := parseScalarValue(m.Value.ValueString())
	return key, map[string]any{key: parsed}, false, diags
}

func (m *BunkerWebGlobalConfigResourceModel) setStateValueFromAPI(value any, preferJSON bool) diag.Diagnostics {
	if preferJSON {
		encoded, err := json.Marshal(value)
		if err != nil {
			return diag.Diagnostics{diag.NewErrorDiagnostic("Encode Global Config Value", fmt.Sprintf("Unable to encode value as JSON: %v", err))}
		}
		m.ValueJSON = types.StringValue(string(encoded))
		m.Value = types.StringNull()
		return nil
	}

	m.Value = types.StringValue(stringifyValue(value))
	m.ValueJSON = types.StringNull()
	return nil
}

func parseScalarValue(input string) any {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}

	switch strings.ToLower(trimmed) {
	case "true":
		return true
	case "false":
		return false
	}

	if i, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return i
	}

	if f, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return f
	}

	return input
}
