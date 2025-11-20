// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func mapFromTerraform(ctx context.Context, value types.Map) (map[string]string, diag.Diagnostics) {
	result := make(map[string]string)

	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}

	diags := value.ElementsAs(ctx, &result, false)
	if diags.HasError() {
		return nil, diags
	}

	return result, diags
}

func mapToTerraform(ctx context.Context, value map[string]string) (types.Map, diag.Diagnostics) {
	if len(value) == 0 {
		return types.MapNull(types.StringType), nil
	}

	result, diags := types.MapValueFrom(ctx, types.StringType, value)
	if diags.HasError() {
		return types.MapNull(types.StringType), diags
	}

	return result, diags
}
