// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

var (
	_ function.Function = BunkerWebFunction{}
)

func NewBunkerWebFunction() function.Function {
	return BunkerWebFunction{}
}

type BunkerWebFunction struct{}

func (r BunkerWebFunction) Metadata(_ context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "service_identifier"
}

func (r BunkerWebFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Derive a BunkerWeb service identifier",
		MarkdownDescription: "Normalizes the provided server name into the identifier expected by the BunkerWeb API.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:                "server_name",
				MarkdownDescription: "Fully qualified domain name used when creating the service in BunkerWeb.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (r BunkerWebFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var serverName string

	resp.Error = function.ConcatFuncErrors(req.Arguments.Get(ctx, &serverName))
	if resp.Error != nil {
		return
	}

	if strings.TrimSpace(serverName) == "" {
		resp.Error = function.NewFuncError("server_name must not be empty")
		return
	}

	identifier := deriveServiceIdentifier(serverName)
	resp.Error = function.ConcatFuncErrors(resp.Result.Set(ctx, identifier))
}
