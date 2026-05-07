//
// Copyright (c) 2025 F5 Inc. All rights reserved.
//

// Package portsvalidator provides a schema validator and function parameter validator for validating that the number
// of ports specified in a comma-separated string does not exceed a specified maximum. It ensures that the configured
// attribute or function parameter value is of single-byte character length less than or equal to the given maximum.
// Null (unconfigured) and unknown (known after apply) values are skipped. If the validator is created with an invalid
// state (e.g., maxPorts less than zero), it will return an implementation error message during validation.
package portsvalidator

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatorfuncerr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = portsAtMostValidator{}
var _ function.StringParameterValidator = portsAtMostValidator{}

type portsAtMostValidator struct {
	maxPorts int
}

func (validator portsAtMostValidator) invalidUsageMessage() string {
	return fmt.Sprintf("maxPorts cannot be less than zero - maxPorts: %d", validator.maxPorts)
}

func (validator portsAtMostValidator) Description(_ context.Context) string {
	return fmt.Sprintf("ports count must be at most %d", validator.maxPorts)
}

func (validator portsAtMostValidator) MarkdownDescription(ctx context.Context) string {
	return validator.Description(ctx)
}

func (validator portsAtMostValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	// Return an error if the validator has been created in an invalid state
	if validator.maxPorts < 0 {
		response.Diagnostics.Append(
			validatordiag.InvalidValidatorUsageDiagnostic(
				request.Path,
				"PortsAtMost",
				validator.invalidUsageMessage(),
			),
		)

		return
	}

	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}
	values := strings.Split(request.ConfigValue.ValueString(), ",")

	if l := len(values); l > validator.maxPorts {
		response.Diagnostics.Append(validatordiag.InvalidAttributeValueLengthDiagnostic(
			request.Path,
			validator.Description(ctx),
			fmt.Sprintf("%d", l),
		))

		return
	}
}

func (validator portsAtMostValidator) ValidateParameterString(ctx context.Context, request function.StringParameterValidatorRequest, response *function.StringParameterValidatorResponse) {
	// Return an error if the validator has been created in an invalid state
	if validator.maxPorts < 0 {
		response.Error = validatorfuncerr.InvalidValidatorUsageFuncError(
			request.ArgumentPosition,
			"PortsAtMost",
			validator.invalidUsageMessage(),
		)

		return
	}

	if request.Value.IsNull() || request.Value.IsUnknown() {
		return
	}

	values := strings.Split(request.Value.ValueString(), ",")

	if l := len(values); l > validator.maxPorts {
		response.Error = validatorfuncerr.InvalidParameterValueLengthFuncError(
			request.ArgumentPosition,
			validator.Description(ctx),
			fmt.Sprintf("%d", l),
		)

		return
	}
}

// PortsAtMost returns an validator which ensures that any configured
// attribute or function parameter value is of single-byte character length less than or equal
// to the given maximum. Null (unconfigured) and unknown (known after apply)
// values are skipped.
//
// maxPorts cannot be less than zero. Invalid input for maxPorts will result in an
// implementation error message during validation.
func PortsAtMost(maxPorts int) portsAtMostValidator {
	return portsAtMostValidator{
		maxPorts: maxPorts,
	}
}
