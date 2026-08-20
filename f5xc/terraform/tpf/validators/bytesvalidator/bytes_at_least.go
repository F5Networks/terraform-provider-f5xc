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

var _ validator.String = BytesAtLeastValidator{}
var _ function.StringParameterValidator = BytesAtLeastValidator{}

// BytesAtLeastValidator enforces that a string attribute or function parameter has a byte size
// greater than or equal to a specified minimum length.
type BytesAtLeastValidator struct {
	minBytes int
}

// invalidUsageMessage returns a formatted error message when minBytes is configured to an invalid negative value.
//
// Returns:
//   - string: the error message explaining the invalid minBytes value.
func (validator BytesAtLeastValidator) invalidUsageMessage() string {
	return fmt.Sprintf("minBytes cannot be less than zero - minBytes: %d", validator.minBytes)
}

// Description returns a human-readable description of the minimum byte length constraint.
//
// Parameters:
//   - ctx: context.Context for context-aware operations.
//
// Returns:
//   - string: description indicating the minimum byte size constraint.
func (validator BytesAtLeastValidator) Description(_ context.Context) string {
	return fmt.Sprintf("bytes size must be at least %d", validator.minBytes)
}

// MarkdownDescription returns a markdown-formatted description of the minimum byte length constraint.
//
// Parameters:
//   - ctx: context.Context for context-aware operations.
//
// Returns:
//   - string: markdown description of the constraint.
func (validator BytesAtLeastValidator) MarkdownDescription(ctx context.Context) string {
	return validator.Description(ctx)
}

// ValidateString validates that a schema attribute string value contains at least minBytes bytes.
//
// Why: Enforces byte length limits on string resource schema attributes to match API constraints.
//
// How: Converts the string value to a byte slice and compares its length against minBytes.
// Null and unknown values are skipped.
//
// Parameters:
//   - ctx: context.Context for context-aware operations.
//   - request: validator.StringRequest containing the attribute path and config value.
//   - response: *validator.StringResponse to record validation errors or diagnostics.
func (validator BytesAtLeastValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	// Return an error if the validator has been created in an invalid state
	if validator.minBytes < 0 {
		response.Diagnostics.Append(
			validatordiag.InvalidValidatorUsageDiagnostic(
				request.Path,
				"BytesAtLeast",
				validator.invalidUsageMessage(),
			),
		)

		return
	}

	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value := []byte(request.ConfigValue.ValueString())

	if l := len(value); l < validator.minBytes {
		response.Diagnostics.Append(validatordiag.InvalidAttributeValueLengthDiagnostic(
			request.Path,
			validator.Description(ctx),
			fmt.Sprintf("%d", l),
		))

		return
	}
}

// ValidateParameterString validates that a function parameter string value contains at least minBytes bytes.
//
// Why: Enforces byte length limits on Terraform function string parameters.
//
// How: Converts the parameter string value to a byte slice and compares its length against minBytes.
// Null and unknown values are skipped.
//
// Parameters:
//   - ctx: context.Context for context-aware operations.
//   - request: function.StringParameterValidatorRequest containing argument position and value.
//   - response: *function.StringParameterValidatorResponse to record validation errors.
func (validator BytesAtLeastValidator) ValidateParameterString(ctx context.Context, request function.StringParameterValidatorRequest, response *function.StringParameterValidatorResponse) {
	// Return an error if the validator has been created in an invalid state
	if validator.minBytes < 0 {
		response.Error = validatorfuncerr.InvalidValidatorUsageFuncError(
			request.ArgumentPosition,
			"BytesAtLeast",
			validator.invalidUsageMessage(),
		)

		return
	}

	if request.Value.IsNull() || request.Value.IsUnknown() {
		return
	}

	value := []byte(request.Value.ValueString())

	if l := len(value); l < validator.minBytes {
		response.Error = validatorfuncerr.InvalidParameterValueLengthFuncError(
			request.ArgumentPosition,
			validator.Description(ctx),
			fmt.Sprintf("%d", l),
		)

		return
	}
}

// BytesAtLeast returns a validator which ensures that any configured string attribute
// or function parameter value has a byte length greater than or equal to minLength.
//
// Why: Used by Omnigen and Terraform Provider resources to enforce minimum byte length constraints derived from proto schemas.
//
// How: Returns a BytesAtLeastValidator initialized with minLength.
//
// Parameters:
//   - minLength: non-negative integer representing the minimum required byte length.
//
// Returns:
//   - BytesAtLeastValidator: a validator enforcing the minimum byte length constraint.
//
// Example:
//
//	"payload": schema.StringAttribute{
//	    Validators: []validator.String{
//	        bytesvalidator.BytesAtLeast(64),
//	    },
//	}
func BytesAtLeast(minLength int) BytesAtLeastValidator {
	return BytesAtLeastValidator{
		minBytes: minLength,
	}
}
