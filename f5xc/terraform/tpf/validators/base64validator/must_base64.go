//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

// Package base64validator provides a Terraform Plugin Framework string validator
// that ensures a given string is a valid base64-encoded value. It defines a custom
// validator type implementing the required interface, which can be used in Terraform
// schema definitions to enforce base64 encoding constraints on string attributes.
// If the value is not a valid base64 string, a descriptive error is added to the diagnostics.
// The package includes a helper function, IsBase64(), for convenient use in schema definitions.
package base64validator

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type base64Validator struct{}

func (v base64Validator) Description(ctx context.Context) string {
	return "string must be a valid base64 encoded string"
}

func (v base64Validator) MarkdownDescription(ctx context.Context) string {
	return "string must be a valid **base64** encoded string"
}

func (v base64Validator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// If the value is null or unknown, skip validation
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	// Only check the part after the schema (if present)
	// or the entire string if it should be directly base64.
	_, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Base64 Encoding",
			fmt.Sprintf("The value %q is not a valid base64 encoded string: %s", value, err.Error()),
		)
	}
}

// IsBase64 Helper function for usage
func IsBase64() validator.String {
	return base64Validator{}
}
