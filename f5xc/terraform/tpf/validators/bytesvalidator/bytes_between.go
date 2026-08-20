//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

// Package bytesvalidator provides a String validator and function parameter validator which ensures that the length
// of the string value in bytes is between the given minimum and maximum.
// This is particularly useful when validating strings that may contain multi-byte characters, ensuring that
// the actual byte size of the string meets specified constraints rather than just the character count.
package bytesvalidator

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatorfuncerr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = bytesBetweenValidator{}
var _ function.StringParameterValidator = bytesBetweenValidator{}

// bytesBetweenValidator enforces that a string attribute or function parameter has a byte size
// between a specified minimum and maximum length.
type bytesBetweenValidator struct {
	minBytes, maxBytes int
}

// invalidUsageMessage returns a formatted error message when minBytes or maxBytes are configured to an invalid range.
//
// Returns:
//   - string: the error message explaining the invalid minBytes/maxBytes configuration.
func (validator bytesBetweenValidator) invalidUsageMessage() string {
	return fmt.Sprintf("minBytes cannot be less than zero or greater than maxBytes - minBytes: %d, maxBytes: %d", validator.minBytes, validator.maxBytes)
}

// Description returns a human-readable description of the byte length range constraint.
//
// Parameters:
//   - ctx: context.Context for context-aware operations.
//
// Returns:
//   - string: description indicating the byte size range constraint.
func (validator bytesBetweenValidator) Description(_ context.Context) string {
	return fmt.Sprintf("bytes size must be between %d and %d", validator.minBytes, validator.maxBytes)
}

// MarkdownDescription returns a markdown-formatted description of the byte length range constraint.
//
// Parameters:
//   - ctx: context.Context for context-aware operations.
//
// Returns:
//   - string: markdown description of the constraint.
func (validator bytesBetweenValidator) MarkdownDescription(ctx context.Context) string {
	return validator.Description(ctx)
}

// ValidateString validates that a schema attribute string value contains between minBytes and maxBytes bytes.
//
// Why: Enforces byte length range limits on string resource schema attributes to match API constraints.
//
// How: Converts the string value to a byte slice and checks if its length falls within [minBytes, maxBytes].
// Null and unknown values are skipped.
//
// Parameters:
//   - ctx: context.Context for context-aware operations.
//   - request: validator.StringRequest containing the attribute path and config value.
//   - response: *validator.StringResponse to record validation errors or diagnostics.
func (validator bytesBetweenValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	// Return an error if the validator has been created in an invalid state
	if validator.minBytes < 0 || validator.minBytes > validator.maxBytes {
		response.Diagnostics.Append(
			validatordiag.InvalidValidatorUsageDiagnostic(
				request.Path,
				"BytesBetween",
				validator.invalidUsageMessage(),
			),
		)

		return
	}

	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value := []byte(request.ConfigValue.ValueString())

	if l := len(value); l < validator.minBytes || l > validator.maxBytes {
		response.Diagnostics.Append(validatordiag.InvalidAttributeValueLengthDiagnostic(
			request.Path,
			validator.Description(ctx),
			fmt.Sprintf("%d", l),
		))

		return
	}
}

// ValidateParameterString validates that a function parameter string value contains between minBytes and maxBytes bytes.
//
// Why: Enforces byte length range limits on Terraform function string parameters.
//
// How: Converts the parameter string value to a byte slice and checks if its length falls within [minBytes, maxBytes].
// Null and unknown values are skipped.
//
// Parameters:
//   - ctx: context.Context for context-aware operations.
//   - request: function.StringParameterValidatorRequest containing argument position and value.
//   - response: *function.StringParameterValidatorResponse to record validation errors.
func (validator bytesBetweenValidator) ValidateParameterString(ctx context.Context, request function.StringParameterValidatorRequest, response *function.StringParameterValidatorResponse) {
	// Return an error if the validator has been created in an invalid state
	if validator.minBytes < 0 || validator.minBytes > validator.maxBytes {
		response.Error = validatorfuncerr.InvalidValidatorUsageFuncError(
			request.ArgumentPosition,
			"BytesBetween",
			validator.invalidUsageMessage(),
		)

		return
	}

	if request.Value.IsNull() || request.Value.IsUnknown() {
		return
	}

	value := []byte(request.Value.ValueString())

	if l := len(value); l < validator.minBytes || l > validator.maxBytes {
		response.Error = validatorfuncerr.InvalidParameterValueLengthFuncError(
			request.ArgumentPosition,
			validator.Description(ctx),
			fmt.Sprintf("%d", l),
		)

		return
	}
}

// BytesBetween returns a validator which ensures that any configured string attribute
// or function parameter value has a byte length between minLength and maxLength inclusive.
//
// Why: Used by Omnigen and Terraform Provider resources to enforce range byte length constraints derived from proto schemas.
//
// How: Returns a bytesBetweenValidator initialized with minLength and maxLength.
//
// Parameters:
//   - minLength: non-negative integer representing the minimum required byte length.
//   - maxLength: integer greater than or equal to minLength representing the maximum permitted byte length.
//
// Returns:
//   - bytesBetweenValidator: a validator enforcing the byte length range constraint.
//
// Example:
//
//	"key": schema.StringAttribute{
//	    Validators: []validator.String{
//	        bytesvalidator.BytesBetween(16, 256),
//	    },
//	}
func BytesBetween(minLength, maxLength int) bytesBetweenValidator {
	return bytesBetweenValidator{
		minBytes: minLength,
		maxBytes: maxLength,
	}
}
