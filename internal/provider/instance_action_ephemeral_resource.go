// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ ephemeral.EphemeralResource = &BunkerWebInstanceActionEphemeralResource{}

// BunkerWebInstanceActionEphemeralResource executes fleet or per-host instance operations.
type BunkerWebInstanceActionEphemeralResource struct {
	client *bunkerWebClient
}

// BunkerWebInstanceActionModel captures Terraform configuration.
type BunkerWebInstanceActionModel struct {
	Operation types.String `tfsdk:"operation"`
	Hostnames types.List   `tfsdk:"hostnames"`
	Test      types.Bool   `tfsdk:"test"`
	Result    types.String `tfsdk:"result"`
}

func NewBunkerWebInstanceActionEphemeralResource() ephemeral.EphemeralResource {
	return &BunkerWebInstanceActionEphemeralResource{}
}

func (r *BunkerWebInstanceActionEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_action"
}

func (r *BunkerWebInstanceActionEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Runs operations against BunkerWeb instances during planning/apply.",
		Attributes: map[string]schema.Attribute{
			"operation": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Operation to execute: one of `ping`, `reload`, `stop`, or `delete`.",
			},
			"hostnames": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Target hostnames. When omitted, the action runs against all instances (for ping/reload/stop only).",
			},
			"test": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "For reload operations, whether to run in test mode (defaults to true). Ignored for other operations.",
			},
			"result": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "JSON-encoded response payload returned by the API.",
				Sensitive:           true,
			},
		},
	}
}

func (r *BunkerWebInstanceActionEphemeralResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
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

func (r *BunkerWebInstanceActionEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "Expected BunkerWeb client to be configured during provider setup.")
		return
	}

	var data BunkerWebInstanceActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Operation.IsNull() || data.Operation.IsUnknown() {
		resp.Diagnostics.AddAttributeError(path.Root("operation"), "Missing Operation", "Set the `operation` attribute to one of ping, reload, stop, or delete.")
		return
	}

	op := strings.ToLower(strings.TrimSpace(data.Operation.ValueString()))
	switch op {
	case "ping", "reload", "stop", "delete":
	default:
		resp.Diagnostics.AddAttributeError(path.Root("operation"), "Unsupported Operation", fmt.Sprintf("Operation %q is not supported. Use ping, reload, stop, or delete.", op))
		return
	}

	hostnames, diags := listToStrings(ctx, data.Hostnames)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result any
	var err error

	switch op {
	case "ping":
		result, err = r.handlePing(ctx, hostnames)
	case "reload":
		result, err = r.handleReload(ctx, hostnames, data.Test)
	case "stop":
		result, err = r.handleStop(ctx, hostnames)
	case "delete":
		result, err = r.handleDelete(ctx, hostnames)
	}

	if err != nil {
		resp.Diagnostics.AddError("Instance Action", err.Error())
		return
	}

	encoded, err := encodeResult(result)
	if err != nil {
		resp.Diagnostics.AddError("Encode Result", err.Error())
		return
	}

	data.Result = types.StringValue(encoded)
	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}

func (r *BunkerWebInstanceActionEphemeralResource) Close(context.Context, ephemeral.CloseRequest, *ephemeral.CloseResponse) {
	// No-op.
}

func (r *BunkerWebInstanceActionEphemeralResource) handlePing(ctx context.Context, hostnames []string) (any, error) {
	if len(hostnames) == 0 {
		return r.client.PingInstances(ctx)
	}

	responses := make(map[string]any, len(hostnames))
	for _, host := range hostnames {
		payload, err := r.client.PingInstance(ctx, host)
		if err != nil {
			return nil, err
		}
		responses[host] = payload
	}

	return responses, nil
}

func (r *BunkerWebInstanceActionEphemeralResource) handleReload(ctx context.Context, hostnames []string, testAttr types.Bool) (any, error) {
	var testPtr *bool
	if !testAttr.IsNull() && !testAttr.IsUnknown() {
		val := testAttr.ValueBool()
		testPtr = &val
	}

	if len(hostnames) == 0 {
		return r.client.ReloadInstances(ctx, testPtr)
	}

	responses := make(map[string]any, len(hostnames))
	for _, host := range hostnames {
		payload, err := r.client.ReloadInstance(ctx, host, testPtr)
		if err != nil {
			return nil, err
		}
		responses[host] = payload
	}

	return responses, nil
}

func (r *BunkerWebInstanceActionEphemeralResource) handleStop(ctx context.Context, hostnames []string) (any, error) {
	if len(hostnames) == 0 {
		return r.client.StopInstances(ctx)
	}

	responses := make(map[string]any, len(hostnames))
	for _, host := range hostnames {
		payload, err := r.client.StopInstance(ctx, host)
		if err != nil {
			return nil, err
		}
		responses[host] = payload
	}

	return responses, nil
}

func (r *BunkerWebInstanceActionEphemeralResource) handleDelete(ctx context.Context, hostnames []string) (any, error) {
	if len(hostnames) == 0 {
		return nil, fmt.Errorf("provide at least one hostname when operation is delete")
	}

	if err := r.client.DeleteInstances(ctx, hostnames); err != nil {
		return nil, err
	}

	return map[string]any{"deleted": hostnames}, nil
}

func encodeResult(result any) (string, error) {
	if result == nil {
		return "{}", nil
	}

	raw, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("encode result payload: %w", err)
	}

	return string(raw), nil
}

func listToStrings(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics

	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}

	var elems []types.String
	diags.Append(list.ElementsAs(ctx, &elems, false)...)
	if diags.HasError() {
		return nil, diags
	}

	values := make([]string, 0, len(elems))
	for idx, elem := range elems {
		if elem.IsNull() || elem.IsUnknown() {
			diags.AddAttributeError(path.Root("hostnames").AtListIndex(idx), "Invalid Hostname", "Hostname values must be set and non-null.")
			continue
		}
		host := strings.TrimSpace(elem.ValueString())
		if host == "" {
			diags.AddAttributeError(path.Root("hostnames").AtListIndex(idx), "Invalid Hostname", "Hostname cannot be empty.")
			continue
		}
		values = append(values, host)
	}

	if diags.HasError() {
		return nil, diags
	}

	return values, diags
}
