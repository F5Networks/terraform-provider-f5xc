//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

package base64validator_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/validators/base64validator"
)

func TestIsBase64_ValidBase64(t *testing.T) {
	t.Parallel()

	v := base64validator.IsBase64()
	req := validator.StringRequest{
		ConfigValue: types.StringValue("aGVsbG8gd29ybGQ="), // "hello world"
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for valid base64, got: %s", resp.Diagnostics)
	}
}

func TestIsBase64_InvalidBase64(t *testing.T) {
	t.Parallel()

	v := base64validator.IsBase64()
	req := validator.StringRequest{
		ConfigValue: types.StringValue("not-valid-base64!!!"),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for invalid base64, got none")
	}
}

func TestIsBase64_NullValue(t *testing.T) {
	t.Parallel()

	v := base64validator.IsBase64()
	req := validator.StringRequest{
		ConfigValue: types.StringNull(),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for null value, got: %s", resp.Diagnostics)
	}
}

func TestIsBase64_UnknownValue(t *testing.T) {
	t.Parallel()

	v := base64validator.IsBase64()
	req := validator.StringRequest{
		ConfigValue: types.StringUnknown(),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for unknown value, got: %s", resp.Diagnostics)
	}
}

func TestIsBase64_EmptyString(t *testing.T) {
	t.Parallel()

	v := base64validator.IsBase64()
	req := validator.StringRequest{
		ConfigValue: types.StringValue(""),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	// Empty string is valid base64
	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for empty string (valid base64), got: %s", resp.Diagnostics)
	}
}

func TestIsBase64_Description(t *testing.T) {
	t.Parallel()

	v := base64validator.IsBase64()
	desc := v.Description(context.Background())
	if desc == "" {
		t.Error("expected non-empty description")
	}
}

func TestIsBase64_MarkdownDescription(t *testing.T) {
	t.Parallel()

	v := base64validator.IsBase64()
	desc := v.MarkdownDescription(context.Background())
	if desc == "" {
		t.Error("expected non-empty markdown description")
	}
}
