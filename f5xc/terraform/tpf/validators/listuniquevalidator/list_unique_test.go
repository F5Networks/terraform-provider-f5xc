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

// TestListUniqueValuesIgnoreUnknown_Description verifies that the markdown
// description equals the plain description.
func TestListUniqueValuesIgnoreUnknown_Description(t *testing.T) {
	ctx := context.Background()
	v := listuniquevalidator.ListUniqueValuesIgnoreUnknown()

	if v.Description(ctx) == "" {
		t.Fatal("Description() is empty")
	}
	if got, want := v.MarkdownDescription(ctx), v.Description(ctx); got != want {
		t.Errorf("MarkdownDescription() = %q, want %q", got, want)
	}
}

// TestListUniqueValuesIgnoreUnknown_NestedCollections verifies that elements
// with an unknown value nested in a list, map, or set are skipped, and fully
// known duplicates of each collection kind are reported.
func TestListUniqueValuesIgnoreUnknown_NestedCollections(t *testing.T) {
	ctx := context.Background()
	v := listuniquevalidator.ListUniqueValuesIgnoreUnknown()

	listType := types.ListType{ElemType: types.StringType}
	mapType := types.MapType{ElemType: types.StringType}
	setType := types.SetType{ElemType: types.StringType}
	objType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"l": listType,
		"m": mapType,
		"s": setType,
	}}

	known := func() attr.Value {
		return types.ObjectValueMust(objType.AttrTypes, map[string]attr.Value{
			"l": types.ListValueMust(types.StringType, []attr.Value{types.StringValue("a")}),
			"m": types.MapValueMust(types.StringType, map[string]attr.Value{"k": types.StringValue("v")}),
			"s": types.SetValueMust(types.StringType, []attr.Value{types.StringValue("x")}),
		})
	}
	withUnknown := func(field string) attr.Value {
		attrs := map[string]attr.Value{
			"l": types.ListValueMust(types.StringType, []attr.Value{types.StringValue("a")}),
			"m": types.MapValueMust(types.StringType, map[string]attr.Value{"k": types.StringValue("v")}),
			"s": types.SetValueMust(types.StringType, []attr.Value{types.StringValue("x")}),
		}
		switch field {
		case "l":
			attrs["l"] = types.ListValueMust(types.StringType, []attr.Value{types.StringUnknown()})
		case "m":
			attrs["m"] = types.MapValueMust(types.StringType, map[string]attr.Value{"k": types.StringUnknown()})
		case "s":
			attrs["s"] = types.SetValueMust(types.StringType, []attr.Value{types.StringUnknown()})
		}
		return types.ObjectValueMust(objType.AttrTypes, attrs)
	}

	tests := []struct {
		name    string
		elems   []attr.Value
		wantErr bool
	}{
		{"known_duplicates", []attr.Value{known(), known()}, true},
		{"unknown_in_list", []attr.Value{withUnknown("l"), withUnknown("l")}, false},
		{"unknown_in_map", []attr.Value{withUnknown("m"), withUnknown("m")}, false},
		{"unknown_in_set", []attr.Value{withUnknown("s"), withUnknown("s")}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &validator.ListResponse{}
			v.ValidateList(ctx, validator.ListRequest{
				Path:        path.Root("items"),
				ConfigValue: types.ListValueMust(objType, tt.elems),
			}, resp)

			if got := resp.Diagnostics.HasError(); got != tt.wantErr {
				t.Errorf("HasError() = %v, want %v: %v", got, tt.wantErr, resp.Diagnostics)
			}
		})
	}
}

// TestListUniqueValuesIgnoreUnknown_NullOrUnknownList verifies that a null,
// unknown, or single-element list is not validated.
func TestListUniqueValuesIgnoreUnknown_NullOrUnknownList(t *testing.T) {
	ctx := context.Background()
	v := listuniquevalidator.ListUniqueValuesIgnoreUnknown()

	for _, lv := range []types.List{
		types.ListNull(types.StringType),
		types.ListUnknown(types.StringType),
		types.ListValueMust(types.StringType, []attr.Value{types.StringValue("a")}),
	} {
		resp := &validator.ListResponse{}
		v.ValidateList(ctx, validator.ListRequest{Path: path.Root("items"), ConfigValue: lv}, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("unexpected error for %v: %v", lv, resp.Diagnostics)
		}
	}
}
