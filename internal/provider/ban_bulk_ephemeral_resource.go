// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ ephemeral.EphemeralResource = &BunkerWebBanBulkEphemeralResource{}

// BunkerWebBanBulkEphemeralResource processes batch ban/unban operations.
type BunkerWebBanBulkEphemeralResource struct {
	client *bunkerWebClient
}

// BunkerWebBanBulkEphemeralResourceModel maps Terraform inputs/results.
type BunkerWebBanBulkEphemeralResourceModel struct {
	Bans   []BunkerWebBanBulkEntryModel `tfsdk:"bans"`
	Unbans []BunkerWebUnbanEntryModel   `tfsdk:"unbans"`
	Result types.String                 `tfsdk:"result"`
}

// BunkerWebBanBulkEntryModel describes a single ban request.
type BunkerWebBanBulkEntryModel struct {
	IP        types.String `tfsdk:"ip"`
	Service   types.String `tfsdk:"service"`
	Reason    types.String `tfsdk:"reason"`
	ExpiresIn types.Int64  `tfsdk:"expires_in"`
}

// BunkerWebUnbanEntryModel describes a single unban request.
type BunkerWebUnbanEntryModel struct {
	IP      types.String `tfsdk:"ip"`
	Service types.String `tfsdk:"service"`
}

func NewBunkerWebBanBulkEphemeralResource() ephemeral.EphemeralResource {
	return &BunkerWebBanBulkEphemeralResource{}
}

func (r *BunkerWebBanBulkEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ban_bulk"
}

func (r *BunkerWebBanBulkEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Executes batch ban and unban operations during apply, useful for synchronizing large ban lists.",
		Attributes: map[string]schema.Attribute{
			"bans": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "IP addresses to ban in this batch.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "IPv4/IPv6 address to ban.",
						},
						"service": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Optional service identifier to scope the ban.",
						},
						"reason": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Reason recorded with the ban (defaults to API behavior).",
						},
						"expires_in": schema.Int64Attribute{
							Optional:            true,
							MarkdownDescription: "Expiration in seconds; zero makes the ban permanent.",
						},
					},
				},
			},
			"unbans": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "IP addresses to unban in this batch.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "IPv4/IPv6 address to unban.",
						},
						"service": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Optional service identifier that scopes the existing ban.",
						},
					},
				},
			},
			"result": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "JSON encoded summary of performed operations.",
			},
		},
	}
}

func (r *BunkerWebBanBulkEphemeralResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
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

func (r *BunkerWebBanBulkEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var data BunkerWebBanBulkEphemeralResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	banReqs, diags := data.toBanRequests()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	unbanReqs, diags := data.toUnbanRequests()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	summary := map[string]any{
		"bans":   len(banReqs),
		"unbans": len(unbanReqs),
	}

	if len(banReqs) > 0 {
		if err := r.client.BanBulk(ctx, banReqs); err != nil {
			resp.Diagnostics.AddError("Ban Bulk", err.Error())
			return
		}
	}

	if len(unbanReqs) > 0 {
		if err := r.client.UnbanBulk(ctx, unbanReqs); err != nil {
			resp.Diagnostics.AddError("Unban Bulk", err.Error())
			return
		}
	}

	encoded, err := encodeResult(summary)
	if err != nil {
		resp.Diagnostics.AddError("Encode Result", err.Error())
		return
	}

	data.Result = types.StringValue(encoded)
	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}

func (r *BunkerWebBanBulkEphemeralResource) Close(context.Context, ephemeral.CloseRequest, *ephemeral.CloseResponse) {
	// No cleanup required.
}

func (m *BunkerWebBanBulkEphemeralResourceModel) toBanRequests() ([]BanRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(m.Bans) == 0 {
		return nil, diags
	}

	reqs := make([]BanRequest, 0, len(m.Bans))
	for idx, entry := range m.Bans {
		if entry.IP.IsNull() || entry.IP.IsUnknown() || strings.TrimSpace(entry.IP.ValueString()) == "" {
			diags.AddAttributeError(path.Root("bans").AtListIndex(idx).AtName("ip"), "Missing IP", "Each ban entry requires a non-empty IP address.")
			continue
		}

		req := BanRequest{IP: strings.TrimSpace(entry.IP.ValueString())}
		if !entry.Service.IsNull() && !entry.Service.IsUnknown() {
			service := strings.TrimSpace(entry.Service.ValueString())
			if service != "" {
				req.Service = &service
			}
		}
		if !entry.Reason.IsNull() && !entry.Reason.IsUnknown() {
			reason := strings.TrimSpace(entry.Reason.ValueString())
			if reason != "" {
				req.Reason = &reason
			}
		}
		if !entry.ExpiresIn.IsNull() && !entry.ExpiresIn.IsUnknown() {
			exp := int(entry.ExpiresIn.ValueInt64())
			req.Exp = &exp
		}
		reqs = append(reqs, req)
	}

	return reqs, diags
}

func (m *BunkerWebBanBulkEphemeralResourceModel) toUnbanRequests() ([]UnbanRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(m.Unbans) == 0 {
		return nil, diags
	}

	reqs := make([]UnbanRequest, 0, len(m.Unbans))
	for idx, entry := range m.Unbans {
		if entry.IP.IsNull() || entry.IP.IsUnknown() || strings.TrimSpace(entry.IP.ValueString()) == "" {
			diags.AddAttributeError(path.Root("unbans").AtListIndex(idx).AtName("ip"), "Missing IP", "Each unban entry requires a non-empty IP address.")
			continue
		}

		req := UnbanRequest{IP: strings.TrimSpace(entry.IP.ValueString())}
		if !entry.Service.IsNull() && !entry.Service.IsUnknown() {
			service := strings.TrimSpace(entry.Service.ValueString())
			if service != "" {
				req.Service = &service
			}
		}
		reqs = append(reqs, req)
	}

	return reqs, diags
}
