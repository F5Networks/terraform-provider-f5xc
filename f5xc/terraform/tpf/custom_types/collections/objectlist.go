//
// Copyright (c) 2025 F5 Inc. All rights reserved.
//

package collections

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
func (v *ListValue[T]) ValueFrom(ctx context.Context, source []T) diag.Diagnostics {
	v.Elements = source

	// 1. Convert the Go struct 'T' into a TPF ObjectValue to infer its attribute schema.
	var zeroT T
	// ObjectValueFrom creates an ObjectValue, which contains the schema map.
	objValue, typeDiags := types.ObjectValueFrom(ctx, map[string]attr.Type{}, zeroT)
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
