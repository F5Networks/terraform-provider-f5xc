//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

// Package listuniquevalidator provides a Terraform list validator that enforces
// element uniqueness while safely ignoring elements containing unknown values.
//
// Why: Standard listvalidator.UniqueValues compares whole elements and treats <unknown>
// fields in nested objects as equal during terraform plan. When referencing newly created
// resources, this produces false-positive Duplicate List Value errors. This validator
// skips elements containing unknown attributes during plan time and validates uniqueness
// once all element fields become fully known.
package listuniquevalidator

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ validator.List = listUniqueValuesIgnoreUnknownValidator{}

type listUniqueValuesIgnoreUnknownValidator struct{}

// ListUniqueValuesIgnoreUnknown returns a list validator that enforces element
// uniqueness while skipping elements containing unknown attributes.
func ListUniqueValuesIgnoreUnknown() validator.List {
	return listUniqueValuesIgnoreUnknownValidator{}
}

// Description returns the human-readable validator description.
func (v listUniqueValuesIgnoreUnknownValidator) Description(_ context.Context) string {
	return "ensures that all known values in a list are unique"
}

// MarkdownDescription returns the markdown-formatted validator description.
func (v listUniqueValuesIgnoreUnknownValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateList enforces that all fully known elements in the list are distinct.
// If any element is unknown or contains nested unknown attributes, it is skipped
// until the apply phase when all attribute values are resolved.
func (v listUniqueValuesIgnoreUnknownValidator) ValidateList(_ context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	elements := req.ConfigValue.Elements()
	if len(elements) <= 1 {
		return
	}

	// Filter out elements that are not fully known (i.e. contain unknown values).
	knownIndices := make([]int, 0, len(elements))
	for i, elem := range elements {
		if isFullyKnown(elem) {
			knownIndices = append(knownIndices, i)
		}
	}

	// Compare pairs of fully known elements.
	for i := 0; i < len(knownIndices); i++ {
		idxI := knownIndices[i]
		elemI := elements[idxI]

		for j := i + 1; j < len(knownIndices); j++ {
			idxJ := knownIndices[j]
			elemJ := elements[idxJ]

			if elemI.Equal(elemJ) {
				resp.Diagnostics.AddAttributeError(
					req.Path,
					"Duplicate List Value",
					"Attribute contains duplicate values.",
				)
				return
			}
		}
	}
}

// isFullyKnown recursively checks if a value and all its nested attributes contain no unknown values.
// Note: Null values (e.g. omitted optional fields) are known values in Terraform and do not cause isFullyKnown to return false.
func isFullyKnown(val attr.Value) bool {
	if val == nil || val.IsUnknown() {
		return false
	}
	if obj, ok := val.(basetypes.ObjectValue); ok {
		for _, attrVal := range obj.Attributes() {
			if !isFullyKnown(attrVal) {
				return false
			}
		}
	}
	if listVal, ok := val.(basetypes.ListValue); ok {
		for _, elem := range listVal.Elements() {
			if !isFullyKnown(elem) {
				return false
			}
		}
	}
	if mapVal, ok := val.(basetypes.MapValue); ok {
		for _, elem := range mapVal.Elements() {
			if !isFullyKnown(elem) {
				return false
			}
		}
	}
	if setVal, ok := val.(basetypes.SetValue); ok {
		for _, elem := range setVal.Elements() {
			if !isFullyKnown(elem) {
				return false
			}
		}
	}
	return true
}
