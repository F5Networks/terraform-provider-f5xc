//
// Copyright (c) 2025 F5 Inc. All rights reserved.
//

package timetypes

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var (
	_ basetypes.StringTypable = (*RFC3339Type)(nil)
)

// RFC3339Type is an attribute type that represents a valid RFC 3339 string. Semantic equality logic is defined
// for RFC3339Type such that inconsequential differences between the `Z` suffix and a `00:00` UTC offset are ignored.
type RFC3339Type struct {
	basetypes.StringType
}

// String returns a human-readable string of the type name.
func (t RFC3339Type) String() string {
	return "timetypes.RFC3339Type"
}

// ValueType returns the Value type.
func (t RFC3339Type) ValueType(ctx context.Context) attr.Value {
	return RFC3339{}
}

// Equal returns true if the given type is equivalent.
func (t RFC3339Type) Equal(o attr.Type) bool {
	other, ok := o.(RFC3339Type)

	if !ok {
		return false
	}

	return t.StringType.Equal(other.StringType)
}

// ValueFromString returns a StringValuable type given a StringValue.
func (t RFC3339Type) ValueFromString(ctx context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	var diags diag.Diagnostics

	if in.IsNull() {
		// Return a Null value if the input is Null
		return NewRFC3339Null(), diags
	}

	// For Unknown values, we trust the framework and return the Unknown value
	if in.IsUnknown() {
		return NewRFC3339Unknown(), diags
	}

	rawString := in.ValueString()

	// If the raw string is empty, treat it as a Null value.
	// This safely handles empty strings from the configuration.
	if rawString == "" {
		return NewRFC3339Null(), diags
	}

	// Check for the literal nil string if it originates from an upstream system:
	if rawString == "(timestamp: nil Timestamp)" {
		return NewRFC3339Null(), diags
	}

	// Parse and Validate the format
	_, err := time.Parse(time.RFC3339, rawString)

	if err != nil {
		// Add diagnostic error if parsing fails
		diags.AddError(
			"Invalid RFC3339 Format",
			fmt.Sprintf("Cannot parse '%s' as RFC3339 time: %s", rawString, err.Error()),
		)
		// Return a Null value because the configuration is invalid
		return NewRFC3339Null(), diags
	}

	// If validation passes, return the RFC3339 value struct wrapping the input string.
	return RFC3339{
		StringValue: in,
	}, diags
}

// ValueFromTerraform returns a Value given a tftypes.Value.  This is meant to convert the tftypes.Value into a more convenient Go type
// for the provider to consume the data with.
func (t RFC3339Type) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)

	if err != nil {
		return nil, err
	}

	stringValue, ok := attrValue.(basetypes.StringValue)

	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T", attrValue)
	}

	stringValuable, diags := t.ValueFromString(ctx, stringValue)

	if diags.HasError() {
		return nil, fmt.Errorf("unexpected error converting StringValue to StringValuable: %v", diags)
	}

	return stringValuable, nil
}
