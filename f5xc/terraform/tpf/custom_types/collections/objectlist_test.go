//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package collections

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testItem struct {
	Name types.String `tfsdk:"name"`
	Port types.Int64  `tfsdk:"port"`
}

type emptyItem struct{}

// TestListValue_Type verifies that Type always reports a list of objects,
// whatever the element type parameter is.
func TestListValue_Type(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		got  func() (tftypes.Type, bool)
	}{
		{
			name: "struct element",
			got: func() (tftypes.Type, bool) {
				typ, diags := ListValue[testItem]{}.Type(ctx)
				return typ, diags.HasError()
			},
		},
		{
			name: "empty struct element",
			got: func() (tftypes.Type, bool) {
				typ, diags := ListValue[emptyItem]{}.Type(ctx)
				return typ, diags.HasError()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			typ, hasErr := tc.got()
			require.False(t, hasErr)
			// tftypes.Object.Equal is false when AttributeTypes is nil, so compare values.
			assert.Equal(t, tftypes.List{ElementType: tftypes.Object{}}, typ)
		})
	}
}

// TestListValue_NullAndUnknown verifies that Null and Unknown set the list state,
// clear Elements, and return the receiver for chaining.
func TestListValue_NullAndUnknown(t *testing.T) {
	tests := []struct {
		name        string
		apply       func(v *ListValue[testItem]) *ListValue[testItem]
		wantNull    bool
		wantUnknown bool
	}{
		{name: "null", apply: (*ListValue[testItem]).Null, wantNull: true},
		{name: "unknown", apply: (*ListValue[testItem]).Unknown, wantUnknown: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := &ListValue[testItem]{Elements: []testItem{{Name: types.StringValue("a")}}}

			got := tc.apply(v)

			assert.Same(t, v, got)
			assert.Nil(t, got.Elements)
			assert.Equal(t, tc.wantNull, got.IsNull())
			assert.Equal(t, tc.wantUnknown, got.IsUnknown())
			assert.Equal(t, types.ObjectType{}, got.ElementType(context.Background()))
		})
	}
}

// TestListValue_ValueFrom_EmptyStruct verifies that a slice of an attribute-free
// struct converts to a known list with one element per source item.
func TestListValue_ValueFrom_EmptyStruct(t *testing.T) {
	tests := []struct {
		name    string
		source  []emptyItem
		wantLen int
	}{
		{name: "empty slice", source: []emptyItem{}, wantLen: 0},
		{name: "two items", source: []emptyItem{{}, {}}, wantLen: 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var v ListValue[emptyItem]

			diags := v.ValueFrom(context.Background(), tc.source)

			require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
			assert.False(t, v.IsNull())
			assert.False(t, v.IsUnknown())
			assert.Len(t, v.List.Elements(), tc.wantLen)
			assert.Equal(t, tc.source, v.Elements)
		})
	}
}

// TestListValue_ValueFrom_NonStructElement verifies that a non-object element type
// returns error diagnostics and leaves the TPF list unset.
func TestListValue_ValueFrom_NonStructElement(t *testing.T) {
	var v ListValue[string]

	diags := v.ValueFrom(context.Background(), []string{"a"})

	require.True(t, diags.HasError())
	assert.True(t, v.IsNull(), "list must stay at its zero (null) value on error")
}

// TestListValue_ValueFrom_PointerElement verifies that a pointer element type,
// whose zero value infers no attributes, fails the list conversion with error
// diagnostics and leaves the TPF list unset.
func TestListValue_ValueFrom_PointerElement(t *testing.T) {
	var v ListValue[*testItem]

	diags := v.ValueFrom(context.Background(), []*testItem{{}})

	require.True(t, diags.HasError())
	assert.True(t, v.IsNull(), "list must stay at its zero (null) value on error")
}

// TestListValue_ValueFrom_StructWithFields verifies that the element object
// type comes from the tfsdk fields of the element struct, so a struct with
// attributes converts to a known list that keeps every item value.
func TestListValue_ValueFrom_StructWithFields(t *testing.T) {
	ctx := context.Background()
	source := []testItem{
		{Name: types.StringValue("a"), Port: types.Int64Value(80)},
		{Name: types.StringValue("b"), Port: types.Int64Null()},
	}
	var v ListValue[testItem]

	diags := v.ValueFrom(ctx, source)

	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
	wantElemType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"name": types.StringType,
		"port": types.Int64Type,
	}}
	assert.Equal(t, wantElemType, v.ElementType(ctx))
	assert.Equal(t, source, v.Elements)

	var got []testItem
	require.False(t, v.List.ElementsAs(ctx, &got, false).HasError())
	assert.Equal(t, source, got)
}

type taggedItem struct {
	Name types.String `tfsdk:"name"`
	Tags types.List   `tfsdk:"tags"`
}

// TestListValue_ValueFrom_CollectionField verifies that a collection field
// takes its element type from the source items, so a struct with a typed list
// converts and keeps every item value.
func TestListValue_ValueFrom_CollectionField(t *testing.T) {
	ctx := context.Background()
	source := []taggedItem{
		{Name: types.StringValue("a"), Tags: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("x")})},
		{Name: types.StringValue("b"), Tags: types.ListNull(types.StringType)},
	}
	var v ListValue[taggedItem]

	diags := v.ValueFrom(ctx, source)

	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
	wantElemType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"name": types.StringType,
		"tags": types.ListType{ElemType: types.StringType},
	}}
	assert.Equal(t, wantElemType, v.ElementType(ctx))

	var got []taggedItem
	require.False(t, v.List.ElementsAs(ctx, &got, false).HasError())
	assert.Equal(t, source, got)
}

// TestListValue_ValueFrom_CollectionFieldWithoutElementType verifies that a
// collection field whose element type cannot be inferred returns error
// diagnostics, does not panic, and leaves the TPF list unset.
func TestListValue_ValueFrom_CollectionFieldWithoutElementType(t *testing.T) {
	tests := []struct {
		name   string
		source []taggedItem
	}{
		{name: "empty source", source: []taggedItem{}},
		{name: "zero list field", source: []taggedItem{{Name: types.StringValue("a")}}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var v ListValue[taggedItem]

			var diags diag.Diagnostics
			require.NotPanics(t, func() { diags = v.ValueFrom(context.Background(), tc.source) })

			require.True(t, diags.HasError())
			assert.Equal(t, "Cannot Infer List Element Type", diags.Errors()[0].Summary())
			assert.True(t, v.IsNull(), "list must stay at its zero (null) value on error")
		})
	}
}

// TestListValue_ValueFrom_MismatchedElementTypes verifies that items whose
// collection fields disagree on the element type return error diagnostics.
func TestListValue_ValueFrom_MismatchedElementTypes(t *testing.T) {
	source := []taggedItem{
		{Name: types.StringValue("a"), Tags: types.ListNull(types.StringType)},
		{Name: types.StringValue("b"), Tags: types.ListNull(types.Int64Type)},
	}
	var v ListValue[taggedItem]

	var diags diag.Diagnostics
	require.NotPanics(t, func() { diags = v.ValueFrom(context.Background(), source) })

	require.True(t, diags.HasError())
	assert.True(t, v.IsNull(), "list must stay at its zero (null) value on error")
}

// TestStructAttrTypes verifies that only exported, tfsdk-tagged attr.Value
// fields become attributes, that a non-struct value has no attributes, and
// that a collection without an element type is an error.
func TestStructAttrTypes(t *testing.T) {
	ctx := context.Background()
	type mixed struct {
		Name     types.String `tfsdk:"name"`
		Skipped  types.String `tfsdk:"-"`
		Untagged types.String
		Plain    string       `tfsdk:"plain"`
		hidden   types.String `tfsdk:"hidden"`
	}

	got, diags := structAttrTypes(ctx, mixed{hidden: types.StringValue("h")})
	require.False(t, diags.HasError())
	assert.Equal(t, map[string]attr.Type{"name": types.StringType}, got)

	for _, v := range []any{"not a struct", nil} {
		got, diags = structAttrTypes(ctx, v)
		require.False(t, diags.HasError())
		assert.Empty(t, got)
	}

	type collections struct {
		List   types.List `tfsdk:"list"`
		Set    types.Set  `tfsdk:"set"`
		Map    types.Map  `tfsdk:"map"`
		Nested types.List `tfsdk:"nested"`
		Typed  types.List `tfsdk:"typed"`
	}
	got, diags = structAttrTypes(ctx, collections{
		Nested: types.ListNull(types.ListType{}),
		Typed:  types.ListNull(types.SetType{ElemType: types.StringType}),
	})
	require.Len(t, diags.Errors(), 4, "list, set, map and the nested list lack an element type")
	assert.Equal(t, map[string]attr.Type{
		"typed": types.ListType{ElemType: types.SetType{ElemType: types.StringType}},
	}, got)
}
