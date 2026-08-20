//
// Copyright (c) 2025 F5 Inc. All rights reserved.
//

// Package exactlyoneofvalidator provides a Terraform object validator that ensures exactly one of a specified set
// of attributes is configured within a OneOf choice block.
//
// This is useful for enforcing mutual exclusivity among fields in a resource schema, such as when only one type
// of configuration should be allowed at a time.
//
// The ExactlyOneOfValidator checks the specified attributes in the object being validated and reports errors if
// more than one is set, or if none are set when the block is present.
//
// Two constructors are provided:
//   - ExactlyOneOf: for optional OneOf blocks — an absent (null) block is silently accepted.
//   - RequiredExactlyOneOf: for required OneOf blocks — an absent (null) block is treated as a validation error.
//
// Example usage in a resource schema (optional OneOf):
//
//	"bot_defense_choice": schema.SingleNestedAttribute{
//	  Optional: true,
//	  Computed: true,
//	  Validators: []validator.Object{
//	    exactlyoneofvalidator.ExactlyOneOf("disable_bot_defense", "bot_defense"),
//	  },
//	},
//
// Example usage in a resource schema (required OneOf):
//
//	"network_prefix_choice": schema.SingleNestedAttribute{
//	  Required: true,
//	  Validators: []validator.Object{
//	    exactlyoneofvalidator.RequiredExactlyOneOf("network_prefix", "network_prefix_v6"),
//	  },
//	},
package exactlyoneofvalidator

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// Ensure ExactlyOneOfValidator implements the validator.Object interface.
var _ validator.Object = ExactlyOneOfValidator{}

// ExactlyOneOfValidator is a Terraform object validator that ensures exactly one of a set of attributes is specified
// within a OneOf choice block.
//
// Fields:
//   - KeysToCheck []string: List of attribute names to check for exclusivity. If more than one is set, or none
//     are set when the block is present, validation fails.
//   - Required bool: If true, an absent (null) block is treated as a validation error. If false, an absent block
//     is silently accepted (the OneOf field is optional).
type ExactlyOneOfValidator struct {
	KeysToCheck []string // List of keys (attribute names) to check for exclusivity
	Required    bool     // If true, a null (absent) block is a validation error
}

// Description returns a human-readable description of the validator.
//
// Parameters:
//   - _ context.Context: Context for the operation (unused).
//
// Returns:
//   - string: Description of the validator.
func (v ExactlyOneOfValidator) Description(_ context.Context) string {
	if v.Required {
		return fmt.Sprintf("Ensures that exactly one of the following attributes is specified (block is required): %v", v.KeysToCheck)
	}
	return fmt.Sprintf("Ensures that exactly one of the following attributes is specified: %v", v.KeysToCheck)
}

// MarkdownDescription returns a markdown-formatted description of the validator.
//
// Parameters:
//   - _ context.Context: Context for the operation (unused).
//
// Returns:
//   - string: Markdown description of the validator.
func (v ExactlyOneOfValidator) MarkdownDescription(_ context.Context) string {
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

	if v.Required {
		return fmt.Sprintf("This block is required. Exactly one of the following attributes must be defined: %s.", keysList)
	}
	return fmt.Sprintf("Exactly one of the following attributes must be defined: %s.", keysList)
}

// ValidateObject validates that exactly one of the specified attributes is set in the object.
//
// Why: OneOf choice blocks must have exactly one option selected. For required OneOf blocks, the block itself
// must also be present (not null). For optional OneOf blocks, an absent block is acceptable.
//
// How: First checks if the block is unknown (skip) or null (error if Required, silent if not). Then counts
// how many of the specified keys are set (not null) and reports an error if the count is not exactly one.
// Unknown values (e.g. variable references) are treated as "set" since the attribute IS configured even
// though its concrete value is not yet resolved.
//
// Parameters:
//   - ctx context.Context: Context for the operation.
//   - req validator.ObjectRequest: Request containing the object value and its path.
//   - resp *validator.ObjectResponse: Response to append diagnostics to.
//
// Returns:
//   - None (results are written to resp.Diagnostics).
func (v ExactlyOneOfValidator) ValidateObject(_ context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	// If the parent object is unknown (e.g. during planning), skip validation —
	// the value is not yet known and cannot be checked.
	if req.ConfigValue.IsUnknown() {
		return
	}

	// If the parent object is null the entire choice block was omitted by the practitioner.
	// For required OneOf blocks this is an error; for optional OneOf blocks it is acceptable.
	if req.ConfigValue.IsNull() {
		if v.Required {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Required Choice Block Not Specified",
				fmt.Sprintf("This block is required. Exactly one of the following attributes must be specified: %v", v.KeysToCheck),
			)
		}
		return
	}

	// Get the raw attribute map for the object being validated.
	attrValues := req.ConfigValue.Attributes()

	var setAttributes []string

	// Iterate over the specific keys we care about and collect those that are set.
	for _, key := range v.KeysToCheck {
		value, ok := attrValues[key]
		if !ok {
			// Attribute is not in the model map, skip.
			continue
		}

		// An unknown value means the attribute IS configured but its value is not
		// yet known (e.g. it references a variable or another resource output).
		// We must treat unknown as "set" so that variable-backed attributes are
		// correctly recognized during the validation phase.
		if !value.IsNull() {
			setAttributes = append(setAttributes, key)
		}
	}

	// Exactly zero attributes are set — at least one is required.
	if len(setAttributes) == 0 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"No Exclusive Attributes Specified",
			fmt.Sprintf("Exactly one of the following attributes must be specified, but none were set: %v", v.KeysToCheck),
		)
	}

	// More than one attribute is set — only one is allowed.
	if len(setAttributes) > 1 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Too Many Exclusive Attributes Specified",
			fmt.Sprintf("At most one of the following attributes may be specified, but %d were set: %v", len(setAttributes), setAttributes),
		)
	}
}

// ExactlyOneOf creates a new ExactlyOneOfValidator for optional OneOf blocks.
//
// Why: Optional OneOf blocks allow the practitioner to omit the entire block. When the block is present,
// exactly one of the specified keys must be set.
//
// Parameters:
//   - keys ...string: Attribute names to check for exclusivity.
//
// Returns:
//   - validator.Object: The ExactlyOneOfValidator instance with Required=false.
//
// Example:
//
//	exactlyoneofvalidator.ExactlyOneOf("disable_bot_defense", "bot_defense")
func ExactlyOneOf(keys ...string) validator.Object {
	return ExactlyOneOfValidator{KeysToCheck: keys, Required: false}
}

// RequiredExactlyOneOf creates a new ExactlyOneOfValidator for required OneOf blocks.
//
// Why: Required OneOf blocks must always be present and must have exactly one option selected.
// Omitting the block entirely is treated as a validation error.
//
// Parameters:
//   - keys ...string: Attribute names to check for exclusivity.
//
// Returns:
//   - validator.Object: The ExactlyOneOfValidator instance with Required=true.
//
// Example:
//
//	exactlyoneofvalidator.RequiredExactlyOneOf("network_prefix", "network_prefix_v6")
func RequiredExactlyOneOf(keys ...string) validator.Object {
	return ExactlyOneOfValidator{KeysToCheck: keys, Required: true}
}
