//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

// Package bytesvalidator contains schema and function validators which can be used to validate byte lengths of string
// attributes and function parameters. These validators are useful when you want to enforce limits on the size of
// string inputs in your Terraform provider.
package bytesvalidator

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatorfuncerr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = bytesAtMostValidator{}
var _ function.StringParameterValidator = bytesAtMostValidator{}

// bytesAtMostValidator enforces that a string attribute or function parameter has a byte size
// less than or equal to a specified maximum length.
type bytesAtMostValidator struct {
	maxBytes int
}

// invalidUsageMessage returns a formatted error message when maxBytes is configured to an invalid negative value.
//
// Returns:
//   - string: the error message explaining the invalid maxBytes value.
func (validator bytesAtMostValidator) invalidUsageMessage() string {
	return fmt.Sprintf("maxBytes cannot be less than zero - maxBytes: %d", validator.maxBytes)
}

// Description returns a human-readable description of the maximum byte length constraint.
//
// Parameters:
//   - ctx: context.Context for context-aware operations.
//
// Returns:
//   - string: description indicating the maximum byte size constraint.
func (validator bytesAtMostValidator) Description(_ context.Context) string {
	return fmt.Sprintf("bytes size must be at most %d", validator.maxBytes)
}

// MarkdownDescription returns a markdown-formatted description of the maximum byte length constraint.
//
// Parameters:
//   - ctx: context.Context for context-aware operations.
//
// Returns:
//   - string: markdown description of the constraint.
func (validator bytesAtMostValidator) MarkdownDescription(ctx context.Context) string {
	return validator.Description(ctx)
}

// ValidateString validates that a schema attribute string value contains at most maxBytes bytes.
//
// Why: Enforces byte length limits on string resource schema attributes to match API constraints.
//
// How: Converts the string value to a byte slice and compares its length against maxBytes.
// Null and unknown values are skipped.
//
// Parameters:
//   - ctx: context.Context for context-aware operations.
//   - request: validator.StringRequest containing the attribute path and config value.
//   - response: *validator.StringResponse to record validation errors or diagnostics.
func (validator bytesAtMostValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	// Return an error if the validator has been created in an invalid state
	if validator.maxBytes < 0 {
		response.Diagnostics.Append(
			validatordiag.InvalidValidatorUsageDiagnostic(
				request.Path,
				"BytesAtMost",
				validator.invalidUsageMessage(),
			),
		)

		return
	}

	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value := []byte(request.ConfigValue.ValueString())

	if l := len(value); l > validator.maxBytes {
		response.Diagnostics.Append(validatordiag.InvalidAttributeValueLengthDiagnostic(
			request.Path,
			validator.Description(ctx),
			fmt.Sprintf("%d", l),
		))

		return
	}
}

// ValidateParameterString validates that a function parameter string value contains at most maxBytes bytes.
//
// Why: Enforces byte length limits on Terraform function string parameters.
//
// How: Converts the parameter string value to a byte slice and compares its length against maxBytes.
// Null and unknown values are skipped.
//
// Parameters:
//   - ctx: context.Context for context-aware operations.
//   - request: function.StringParameterValidatorRequest containing argument position and value.
//   - response: *function.StringParameterValidatorResponse to record validation errors.
func (validator bytesAtMostValidator) ValidateParameterString(ctx context.Context, request function.StringParameterValidatorRequest, response *function.StringParameterValidatorResponse) {
	// Return an error if the validator has been created in an invalid state
	if validator.maxBytes < 0 {
		response.Error = validatorfuncerr.InvalidValidatorUsageFuncError(
			request.ArgumentPosition,
			"BytesAtMost",
			validator.invalidUsageMessage(),
		)

		return
	}

	if request.Value.IsNull() || request.Value.IsUnknown() {
		return
	}

	value := []byte(request.Value.ValueString())

	if l := len(value); l > validator.maxBytes {
		response.Error = validatorfuncerr.InvalidParameterValueLengthFuncError(
			request.ArgumentPosition,
			validator.Description(ctx),
			fmt.Sprintf("%d", l),
		)

		return
	}
}

// BytesAtMost returns a validator which ensures that any configured string attribute
// or function parameter value has a byte length less than or equal to maxBytes.
//
// Why: Used by Omnigen and Terraform Provider resources to enforce maximum byte length constraints derived from proto schemas.
//
// How: Returns a bytesAtMostValidator initialized with maxBytes.
//
// Parameters:
//   - maxBytes: non-negative integer representing the maximum permitted byte length.
//
// Returns:
//   - bytesAtMostValidator: a validator enforcing the maximum byte length constraint.
//
// Example:
//
//	"certificate": schema.StringAttribute{
//	    Validators: []validator.String{
//	        bytesvalidator.BytesAtMost(4096),
//	    },
//	}
func BytesAtMost(maxBytes int) bytesAtMostValidator {
	return bytesAtMostValidator{
		maxBytes: maxBytes,
	}
}
