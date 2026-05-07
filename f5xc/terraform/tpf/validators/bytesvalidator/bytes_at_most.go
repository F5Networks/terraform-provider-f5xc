//
// Copyright (c) 2025 F5 Inc. All rights reserved.
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

type bytesAtMostValidator struct {
	maxBytes int
}

func (validator bytesAtMostValidator) invalidUsageMessage() string {
	return fmt.Sprintf("maxBytes cannot be less than zero - maxBytes: %d", validator.maxBytes)
}

func (validator bytesAtMostValidator) Description(_ context.Context) string {
	return fmt.Sprintf("bytes size must be at most %d", validator.maxBytes)
}

func (validator bytesAtMostValidator) MarkdownDescription(ctx context.Context) string {
	return validator.Description(ctx)
}

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

// BytesAtMost returns an validator which ensures that any configured
// attribute or function parameter value is of single-byte character length less than or equal
// to the given maximum. Null (unconfigured) and unknown (known after apply)
// values are skipped.
//
// maxBytes cannot be less than zero. Invalid input for maxBytes will result in an
// implementation error message during validation.
func BytesAtMost(maxBytes int) bytesAtMostValidator {
	return bytesAtMostValidator{
		maxBytes: maxBytes,
	}
}
