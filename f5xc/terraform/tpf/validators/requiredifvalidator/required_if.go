//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

// Package requiredifvalidator provides a Terraform string validator that ensures a field is set
// when a sibling field holds one of the specified values.
//
// This is useful for enforcing conditional required fields, such as when a field becomes mandatory
// only when another field is set to a specific value.
//
// The RequiredIfValidator checks the specified sibling field and reports an error if the current
// field is null (not set) while the sibling field matches one of the specified values.
// The sibling field may be either a string attribute or a list attribute; both are handled
// transparently.
//
// Example usage in a resource schema:
//
//	"site_name": schema.StringAttribute{
//	  Optional: true,
//	  Validators: []validator.String{
//	    requiredifvalidator.RequiredIf("type", "JWT"),
//	  },
//	},
//
// In this example, site_name becomes required when the type field is set to "JWT" (string) or
// contains "JWT" as a list element (list of strings).
package requiredifvalidator

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure RequiredIfValidator implements the validator.String interface.
var _ validator.String = RequiredIfValidator{}

// RequiredIfValidator is a Terraform string validator that ensures a field is set when a sibling
// field holds one of the specified values.
//
// The sibling field is read as types.String first. If that fails (e.g. because the sibling is
// actually a list attribute), it is re-read as types.List and each element is checked against the
// target values. This makes the validator work transparently for both string and list siblings.
//
// Fields:
//   - SiblingField string: Name of the sibling field to check (e.g., "type").
//   - SiblingValues []string: List of values that trigger the requirement. If the sibling field
//     is set to one of these values (or contains one of them when it is a list), the current
//     field must also be set.
type RequiredIfValidator struct {
	SiblingField  string   // Name of the sibling field to check
	SiblingValues []string // Values that trigger the requirement
}

// Description returns a human-readable description of the validator.
//
// Parameters:
//   - _ context.Context: Context for the operation (unused).
//
// Returns:
//   - string: Description of the validator.
func (v RequiredIfValidator) Description(_ context.Context) string {
	return fmt.Sprintf("This field is required when %q is set to one of: %v", v.SiblingField, v.SiblingValues)
}

// MarkdownDescription returns a markdown-formatted description of the validator.
//
// Parameters:
//   - _ context.Context: Context for the operation (unused).
//
// Returns:
//   - string: Markdown description of the validator.
func (v RequiredIfValidator) MarkdownDescription(_ context.Context) string {
	var valuesList string
	for i, val := range v.SiblingValues {
		if i > 0 {
			if i == len(v.SiblingValues)-1 {
				valuesList += " or "
			} else {
				valuesList += ", "
			}
		}
		valuesList += fmt.Sprintf("`%s`", val)
	}
	return fmt.Sprintf("This field is required when `%s` is set to %s.", v.SiblingField, valuesList)
}

// siblingMatchesString attempts to read the sibling field as a types.String and checks whether
// its value is one of the target values.
//
// Why: The sibling field may be a plain string attribute. This helper encapsulates the
// string-specific retrieval and comparison so that ValidateString can fall back to the list
// variant when this returns (false, false).
//
// How: Calls GetAttribute with a *types.String target. On success it compares the value against
// each entry in v.SiblingValues.
//
// Parameters:
//   - ctx context.Context: Context for the operation.
//   - req validator.StringRequest: The current validation request (provides Config).
//
// Returns:
//   - matched bool: true when the sibling value equals one of v.SiblingValues.
//   - ok bool: true when the attribute was successfully read as a string (even if not matched).
func (v RequiredIfValidator) siblingMatchesString(ctx context.Context, req validator.StringRequest) (matched bool, ok bool) {
	var siblingVal types.String
	diags := req.Config.GetAttribute(ctx, req.Path.ParentPath().AtName(v.SiblingField), &siblingVal)
	if diags.HasError() {
		return false, false
	}
	if siblingVal.IsUnknown() || siblingVal.IsNull() {
		return false, true
	}
	for _, val := range v.SiblingValues {
		if siblingVal.ValueString() == val {
			return true, true
		}
	}
	return false, true
}

// siblingMatchesList attempts to read the sibling field as a types.List of strings and checks
// whether any element is one of the target values.
//
// Why: The sibling field may be a list attribute (e.g. the token "type" field is mode/ Why: The sibling field may be
// a list attribute (e.g. the token "type" field is modeled as []string in the proto and rendered as schema.ListAttribute).
// A plain types.String read would fail with a type-mismatch diagnostic, silently skipping the validator. This helper handles
// that case.
//
// How: Calls GetAttribute with a *types.List target. On success it iterates over the elements,
// casts each to types.String, and compares against v.SiblingValues.
//
// Parameters:
//   - ctx context.Context: Context for the operation.
//   - req validator.StringRequest: The current validation request (provides Config).
//
// Returns:
//   - matched bool: true when any list element equals one of v.SiblingValues.
//   - ok bool: true when the attribute was successfully read as a list (even if not matched).
func (v RequiredIfValidator) siblingMatchesList(ctx context.Context, req validator.StringRequest) (matched bool, ok bool) {
	var siblingList types.List
	diags := req.Config.GetAttribute(ctx, req.Path.ParentPath().AtName(v.SiblingField), &siblingList)
	if diags.HasError() {
		return false, false
	}
	if siblingList.IsUnknown() || siblingList.IsNull() {
		return false, true
	}
	for _, elem := range siblingList.Elements() {
		strVal, isStr := elem.(types.String)
		if !isStr {
			continue
		}
		if strVal.IsUnknown() || strVal.IsNull() {
			continue
		}
		for _, val := range v.SiblingValues {
			if strVal.ValueString() == val {
				return true, true
			}
		}
	}
	return false, true
}

// ValidateString validates that the current field is set when the sibling field holds one of
// the specified values.
//
// Why: Conditional required fields enforce business logic where certain fields become mandatory
// only under specific conditions (e.g., site_name is required only when type is "JWT").
//
// How: First checks if the current field is unknown (skip). Then tries to read the sibling field
// as a string; if that fails due to a type mismatch (the sibling is a list), it retries as a
// list of strings. If either read succeeds and the sibling value matches, an error is reported
// when the current field is null.
//
// Parameters:
//   - ctx context.Context: Context for the operation.
//   - req validator.StringRequest: Request containing the string value and its path.
//   - resp *validator.StringResponse: Response to append diagnostics to.
//
// Returns:
//   - None (results are written to resp.Diagnostics).
func (v RequiredIfValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// If the current field is unknown (e.g. during planning), skip validation.
	if req.ConfigValue.IsUnknown() {
		return
	}

	// Try to read the sibling as a plain string first.
	siblingMatches, ok := v.siblingMatchesString(ctx, req)
	if !ok {
		// String read failed — sibling is likely a list attribute. Retry as list.
		siblingMatches, _ = v.siblingMatchesList(ctx, req)
	}

	// If the sibling field matches one of the specified values and the current field is null,
	// report an error.
	if siblingMatches && req.ConfigValue.IsNull() {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Conditional Required Field",
			fmt.Sprintf(
				"Field %q is required when %q is set to one of %v, but it is not set.",
				req.Path.String(),
				v.SiblingField,
				v.SiblingValues,
			),
		)
	}
}

// RequiredIf creates a new RequiredIfValidator.
//
// Why: Conditional required fields enforce business logic where certain fields become mandatory
// only under specific conditions. The validator handles both string and list sibling attributes
// transparently.
//
// Parameters:
//   - siblingField: Name of the sibling field to check (e.g., "type").
//   - siblingValues: Values that trigger the requirement (e.g., "JWT", "API_CERTIFICATE").
//
// Returns:
//   - validator.String: The RequiredIfValidator instance.
//
// Example:
//
//	requiredifvalidator.RequiredIf("type", "JWT")
func RequiredIf(siblingField string, siblingValues ...string) validator.String {
	return RequiredIfValidator{
		SiblingField:  siblingField,
		SiblingValues: siblingValues,
	}
}
