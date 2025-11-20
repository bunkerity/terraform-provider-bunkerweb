// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &BunkerWebPluginResource{}
var _ resource.ResourceWithImportState = &BunkerWebPluginResource{}

// BunkerWebPluginResource manages lifecycle of uploaded plugins.
type BunkerWebPluginResource struct {
	client *bunkerWebClient
}

// BunkerWebPluginResourceModel stores Terraform plan/state.
type BunkerWebPluginResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Method  types.String `tfsdk:"method"`
	Name    types.String `tfsdk:"name"`
	Content types.String `tfsdk:"content"`
}

func NewBunkerWebPluginResource() resource.Resource {
	return &BunkerWebPluginResource{}
}

func (r *BunkerWebPluginResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plugin"
}

func (r *BunkerWebPluginResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Uploads and manages a single BunkerWeb plugin package via the control plane.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique plugin identifier assigned by the API (derived from the uploaded file name).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"method": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional method field forwarded to the API (defaults to `ui`).",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "File name to associate with the uploaded plugin payload (for example `custom.lua`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"content": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Plugin file contents. Use functions such as `file()` to read local files.",
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *BunkerWebPluginResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BunkerWebPluginResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var plan BunkerWebPluginResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := strings.TrimSpace(plan.Name.ValueString())
	if name == "" {
		resp.Diagnostics.AddAttributeError(path.Root("name"), "Invalid Name", "Provide a non-empty plugin file name.")
		return
	}

	content := plan.Content.ValueString()
	uploadReq := PluginUploadRequest{
		Method: strings.TrimSpace(plan.Method.ValueString()),
		Files: []PluginUploadFile{
			{FileName: name, Content: []byte(content)},
		},
	}

	plugins, err := r.client.UploadPlugins(ctx, uploadReq)
	if err != nil {
		resp.Diagnostics.AddError("Upload Plugin", err.Error())
		return
	}
	if len(plugins) == 0 {
		resp.Diagnostics.AddError("Upload Plugin", "API response did not include uploaded plugin metadata")
		return
	}

	plan.ID = types.StringValue(plugins[0].ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BunkerWebPluginResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var state BunkerWebPluginResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.IsNull() || state.ID.IsUnknown() {
		resp.State.RemoveResource(ctx)
		return
	}

	plugins, err := r.client.ListPlugins(ctx, "all", false)
	if err != nil {
		resp.Diagnostics.AddError("Read Plugin", err.Error())
		return
	}

	id := state.ID.ValueString()
	for _, plugin := range plugins {
		if plugin.ID == id {
			return
		}
	}

	resp.State.RemoveResource(ctx)
}

func (r *BunkerWebPluginResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
	// Updates are modeled as force-new via plan modifiers on name/content.
}

func (r *BunkerWebPluginResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var state BunkerWebPluginResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.IsNull() || state.ID.IsUnknown() {
		return
	}

	if err := r.client.DeletePlugin(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Delete Plugin", err.Error())
	}
}

func (r *BunkerWebPluginResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.Set(ctx, &BunkerWebPluginResourceModel{
		ID: types.StringValue(strings.TrimSpace(req.ID)),
	})...)
}
