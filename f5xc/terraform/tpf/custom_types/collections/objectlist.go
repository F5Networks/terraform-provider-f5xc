//
// Copyright (c) 2025 F5 Inc. All rights reserved.
//

package collections

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// ListValue is a generic wrapper around types.List designed for lists of complex objects (structs).
// It embeds types.List for TPF compatibility and includes a Go slice (Elements) for easy,
// non-TPF-wrapped data access during conversion.
// T must be a struct that uses TPF types (types.String, types.Int64, etc.)
// To avoid to implement Type(), Null(), Unknown(), and ValueFrom() for every referenced struct type
// let generics do the work :).
type ListValue[T any] struct {
	types.List
	Elements []T // Standard Go slice for easy conversion (e.g., for API payload)
}

// Type reports the inferred type of the ListValue. This is crucial for Schema validation.
// This method remains a value receiver as required by the tftypes.Type interface.
func (v ListValue[T]) Type(_ context.Context) (tftypes.Type, diag.Diagnostics) {
	// The element type for a ListValue[T] where T is a struct is always an Object type.
	// We instantiate tftypes.Object{} to create a zero-value instance of the type.
	return tftypes.List{ElementType: tftypes.Object{}}, nil
}

// Null returns a null ListValue, used when a list has been explicitly set to null.
func (v *ListValue[T]) Null() *ListValue[T] {
	// Instantiate types.ObjectType{} to provide a value of the required type.
	v.List = types.ListNull(types.ObjectType{})
	v.Elements = nil
	return v
}

// Unknown returns an unknown ListValue, used when Terraform hasn't determined the value yet.
func (v *ListValue[T]) Unknown() *ListValue[T] {
	// Instantiate types.ObjectType{} to provide a value of the required type.
	v.List = types.ListUnknown(types.ObjectType{})
	v.Elements = nil
	return v
}

// ValueFrom sets the TPF internal value from a standard Go slice (Source).
// This remains a pointer receiver as it mutates the struct state.
//
// The element attribute types come from the first source item, because a
// collection field such as types.List carries its element type only in a
// value. With an empty source, the zero T is used. ValueFrom returns error
// diagnostics, and does not set the list, when an element type cannot be
// inferred, for example for a collection field of a zero T.
func (v *ListValue[T]) ValueFrom(ctx context.Context, source []T) diag.Diagnostics {
	v.Elements = source

	// 1. Convert the Go struct 'T' into a TPF ObjectValue to infer its attribute schema.
	var sample T
	if len(source) > 0 {
		sample = source[0]
	}
	attrTypes, attrDiags := structAttrTypes(ctx, sample)
	if attrDiags.HasError() {
		return attrDiags
	}
	// ObjectValueFrom creates an ObjectValue, which contains the schema map.
	objValue, typeDiags := types.ObjectValueFrom(ctx, attrTypes, sample)
	if typeDiags.HasError() {
		return typeDiags
	}

	// 2. Extract the Attribute Types map (map[string]attr.Type) from the ObjectValue.
	// This map represents the schema of the nested struct T.
	attrTypesMap := objValue.AttributeTypes(ctx)

	// 3. Construct the element's attr.Type (types.ObjectType) using the map.
	elementAttrType := types.ObjectType{AttrTypes: attrTypesMap}

	// 4. Create a new types.List from the Go slice, using the inferred elementAttrType.
	list, diags := types.ListValueFrom(ctx, elementAttrType, source)
	if diags.HasError() {
		return diags
	}

	v.List = list

	// Return diagnostics.
	return diags
}

// structAttrTypes returns the attribute types of the tfsdk-tagged fields of
// struct v. ObjectValueFrom needs these types to convert a struct that has
// attributes.
//
// The types come from the field values of v. Fields without a tfsdk tag,
// tagged "-", or whose type is not an attr.Value are skipped. A non-struct v
// returns an empty map, so ObjectValueFrom reports the type error.
//
// It returns error diagnostics when a field is a collection without an element
// type, such as a zero types.List. The framework panics on such a type, so the
// caller must stop.
func structAttrTypes(ctx context.Context, v any) (map[string]attr.Type, diag.Diagnostics) {
	var diags diag.Diagnostics
	attrTypes := map[string]attr.Type{}
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return attrTypes, diags
	}
	rt := rv.Type()
	for i := range rt.NumField() {
		field := rt.Field(i)
		name, _, _ := strings.Cut(field.Tag.Get("tfsdk"), ",")
		if name == "" || name == "-" || !field.IsExported() {
			continue
		}
		value, ok := rv.Field(i).Interface().(attr.Value)
		if !ok {
			continue
		}
		attrType := value.Type(ctx)
		if !hasElementTypes(attrType) {
			diags.AddError(
				"Cannot Infer List Element Type",
				fmt.Sprintf("Attribute %q is a collection without an element type. "+
					"Provide a non-empty source whose items have typed collection values.", name),
			)
			continue
		}
		attrTypes[name] = attrType
	}
	return attrTypes, diags
}

// hasElementTypes reports whether t and every collection type nested in it
// have an element type. A zero collection value, such as types.List{}, reports
// a type whose element type is nil.
func hasElementTypes(t attr.Type) bool {
	collection, ok := t.(attr.TypeWithElementType)
	if !ok {
		return true
	}
	switch c := collection.(type) {
	case basetypes.ListType:
		if c.ElemType == nil {
			return false
		}
	case basetypes.SetType:
		if c.ElemType == nil {
			return false
		}
	case basetypes.MapType:
		if c.ElemType == nil {
			return false
		}
	}
	return hasElementTypes(collection.ElementType())
}
