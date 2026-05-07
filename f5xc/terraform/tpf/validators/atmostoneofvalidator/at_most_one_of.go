//
// Copyright (c) 2025 F5 Inc. All rights reserved.
//

// Package atmostoneofvalidator provides a Terraform object validator that ensures at most one of a specified set
// of attributes is configured.
// This is useful for enforcing mutual exclusivity among fields in a resource schema, such as when only one type
// of configuration should be allowed.
//
// The AtMostOneOfValidator checks the specified attributes in the object being validated and reports errors if
// more than one is set or if none are set when at least one is required.
//
// Example usage in a resource schema:
//
//	"choice": schema.SingleNestedAttribute{
//	  Attributes: map[string]schema.Attribute{
//	    "option_a": schema.StringAttribute{},
//	    "option_b": schema.StringAttribute{},
//	    "option_c": schema.StringAttribute{},
//	  },
//	  Validators: []validator.Object{
//	    atmostoneofvalidator.AtMostOneOf("option_a", "option_b", "option_c"),
//	  },
//	},
package atmostoneofvalidator

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// Ensure AtMostOneOfValidator implements the validator.Object interface
var _ validator.Object = AtMostOneOfValidator{}

// AtMostOneOfValidator is a Terraform object validator that ensures at most one of a set of attributes is specified.
// It is used to enforce mutual exclusivity among a group of fields in a resource schema.
//
// Fields:
//   - KeysToCheck []string: List of attribute names to check for exclusivity. If more than one is set, validation fails.
type AtMostOneOfValidator struct {
	KeysToCheck []string // List of keys (attribute names) to check for exclusivity
}

// Description returns a description of the validator.
// Parameters:
//   - _ context.Context: Context for the operation (unused).
//
// Returns:
//   - string: Description of the validator.
func (v AtMostOneOfValidator) Description(_ context.Context) string {
	return fmt.Sprintf("Ensures that at most one of the following attributes is specified: %v", v.KeysToCheck)
}

// MarkdownDescription returns a markdown-formatted description of the validator.
// Parameters:
//   - _ context.Context: Context for the operation (unused).
//
// Returns:
//   - string: Markdown description of the validator.
func (v AtMostOneOfValidator) MarkdownDescription(_ context.Context) string {
	// Create a nice list of the keys for the Markdown output
	var keysList string
	for i, key := range v.KeysToCheck {
		if i > 0 {
			if i == len(v.KeysToCheck)-1 {
				keysList += " or "
			} else {
				keysList += ", "
			}
		}
		keysList += fmt.Sprintf("`%s`", key)
	}

	// Since your validation logic requires exactly one (error if 0, error if > 1),
	// we should document it as a "Choice" or "Exclusive" constraint.
	return fmt.Sprintf("Exactly one of the following attributes must be defined: %s.", keysList)
}

// ValidateObject validates that at most one of the specified attributes is set in the object.
// Parameters:
//   - ctx context.Context: Context for the operation.
//   - req validator.ObjectRequest: Request containing the object and path.
//   - resp *validator.ObjectResponse: Response to append diagnostics to.
//
// Returns:
//   - None (results are written to resp.Diagnostics).
func (v AtMostOneOfValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	// If the parent object itself is null or unknown, there's nothing to validate
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	// The Value (req.ConfigValue) is the object being validated ("choice" block)
	// Get the raw attribute map for the object being validated
	attrValues := req.ConfigValue.Attributes()

	var setAttributes []string

	// Iterate over the specific keys we care about
	for _, key := range v.KeysToCheck {
		value, ok := attrValues[key]
		if !ok {
			// Attribute is not in the model map, skip
			continue
		}

		// Check if the attribute is configured (not Null and not Unknown)
		if !value.IsNull() && !value.IsUnknown() {
			setAttributes = append(setAttributes, key)
		}
	}

	// Check the count == 0
	if len(setAttributes) == 0 {
		// No attributes are set, needs at least one
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"No Exclusive Attributes Specified",
			fmt.Sprintf("At least one of the following attributes may be specified, but %d were set: %v", len(setAttributes), v.KeysToCheck),
		)
	}

	// Check the count > 1
	if len(setAttributes) > 1 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Too Many Exclusive Attributes Specified",
			fmt.Sprintf("At most one of the following attributes may be specified, but %d were set: %v", len(setAttributes), setAttributes),
		)
	}
}

// AtMostOneOf creates a new AtMostOneOfValidator instance for the given keys.
// Parameters:
//   - keys ...string: Attribute names to check for exclusivity.
//
// Returns:
//   - validator.Object: The AtMostOneOfValidator instance.
func AtMostOneOf(keys ...string) validator.Object {
	return AtMostOneOfValidator{KeysToCheck: keys}
}
