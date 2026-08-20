//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

package listuniquevalidator_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/validators/listuniquevalidator"
)

func TestListUniqueValuesIgnoreUnknown(t *testing.T) {
	v := listuniquevalidator.ListUniqueValuesIgnoreUnknown()

	objType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":        types.StringType,
			"namespace":   types.StringType,
			"tenant":      types.StringType,
			"description": types.StringType,
		},
	}

	tests := []struct {
		name      string
		listVal   types.List
		expectErr bool
	}{
		{
			name: "unique scalar list",
			listVal: types.ListValueMust(
				types.StringType,
				[]attr.Value{
					types.StringValue("a"),
					types.StringValue("b"),
				},
			),
			expectErr: false,
		},
		{
			name: "duplicate scalar list",
			listVal: types.ListValueMust(
				types.StringType,
				[]attr.Value{
					types.StringValue("a"),
					types.StringValue("a"),
				},
			),
			expectErr: true,
		},
		{
			name: "unique object list with known values",
			listVal: types.ListValueMust(
				objType,
				[]attr.Value{
					types.ObjectValueMust(objType.AttrTypes, map[string]attr.Value{
						"name":        types.StringValue("sp1"),
						"namespace":   types.StringValue("default"),
						"tenant":      types.StringValue("t1"),
						"description": types.StringNull(),
					}),
					types.ObjectValueMust(objType.AttrTypes, map[string]attr.Value{
						"name":        types.StringValue("sp2"),
						"namespace":   types.StringValue("default"),
						"tenant":      types.StringValue("t1"),
						"description": types.StringNull(),
					}),
				},
			),
			expectErr: false,
		},
		{
			name: "duplicate object list with omitted/null optional attributes",
			listVal: types.ListValueMust(
				objType,
				[]attr.Value{
					types.ObjectValueMust(objType.AttrTypes, map[string]attr.Value{
						"name":        types.StringValue("sp1"),
						"namespace":   types.StringValue("default"),
						"tenant":      types.StringValue("t1"),
						"description": types.StringNull(),
					}),
					types.ObjectValueMust(objType.AttrTypes, map[string]attr.Value{
						"name":        types.StringValue("sp1"),
						"namespace":   types.StringValue("default"),
						"tenant":      types.StringValue("t1"),
						"description": types.StringNull(),
					}),
				},
			),
			expectErr: true,
		},
		{
			name: "object list with unknown attributes (plan phase reference)",
			listVal: types.ListValueMust(
				objType,
				[]attr.Value{
					types.ObjectValueMust(objType.AttrTypes, map[string]attr.Value{
						"name":        types.StringUnknown(),
						"namespace":   types.StringUnknown(),
						"tenant":      types.StringValue("t1"),
						"description": types.StringNull(),
					}),
					types.ObjectValueMust(objType.AttrTypes, map[string]attr.Value{
						"name":        types.StringUnknown(),
						"namespace":   types.StringUnknown(),
						"tenant":      types.StringValue("t1"),
						"description": types.StringNull(),
					}),
				},
			),
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.ListRequest{
				Path:        path.Root("test"),
				ConfigValue: tt.listVal,
			}
			resp := validator.ListResponse{}

			v.ValidateList(context.Background(), req, &resp)

			if tt.expectErr && !resp.Diagnostics.HasError() {
				t.Errorf("expected validation error, got none")
			}
			if !tt.expectErr && resp.Diagnostics.HasError() {
				t.Errorf("expected no error, got: %v", resp.Diagnostics)
			}
		})
	}
}
