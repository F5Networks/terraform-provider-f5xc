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

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatorfuncerr"
)

var _ validator.String = BytesAtLeastValidator{}
var _ function.StringParameterValidator = BytesAtLeastValidator{}

type BytesAtLeastValidator struct {
	minBytes int
}

func (validator BytesAtLeastValidator) invalidUsageMessage() string {
	return fmt.Sprintf("minBytes cannot be less than zero - minBytes: %d", validator.minBytes)
}

func (validator BytesAtLeastValidator) Description(_ context.Context) string {
	return fmt.Sprintf("bytes size must be at least %d", validator.minBytes)
}

func (validator BytesAtLeastValidator) MarkdownDescription(ctx context.Context) string {
	return validator.Description(ctx)
}

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

// BytesAtLeast returns an validator which ensures that any configured
// attribute or function parameter value is of single-byte character length greater than or equal
// to the given minimum. Null (unconfigured) and unknown (known after apply)
// values are skipped.
//
// minBytes cannot be less than zero. Invalid input for minBytes will result in an
// implementation error message during validation.
func BytesAtLeast(minLength int) BytesAtLeastValidator {
	return BytesAtLeastValidator{
		minBytes: minLength,
	}
}
