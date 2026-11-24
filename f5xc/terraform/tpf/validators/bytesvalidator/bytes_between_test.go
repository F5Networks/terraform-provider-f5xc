//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

package bytesvalidator_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/validators/bytesvalidator"
)

func TestBytesBetween_Valid(t *testing.T) {
	t.Parallel()

	v := bytesvalidator.BytesBetween(3, 10)
	req := validator.StringRequest{
		ConfigValue: types.StringValue("hello"),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for value within range, got: %s", resp.Diagnostics)
	}
}

func TestBytesBetween_AtMin(t *testing.T) {
	t.Parallel()

	v := bytesvalidator.BytesBetween(5, 10)
	req := validator.StringRequest{
		ConfigValue: types.StringValue("hello"),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for value at minimum, got: %s", resp.Diagnostics)
	}
}

func TestBytesBetween_AtMax(t *testing.T) {
	t.Parallel()

	v := bytesvalidator.BytesBetween(3, 5)
	req := validator.StringRequest{
		ConfigValue: types.StringValue("hello"),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for value at maximum, got: %s", resp.Diagnostics)
	}
}

func TestBytesBetween_TooShort(t *testing.T) {
	t.Parallel()

	v := bytesvalidator.BytesBetween(10, 20)
	req := validator.StringRequest{
		ConfigValue: types.StringValue("hi"),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for value below minimum, got none")
	}
}

func TestBytesBetween_TooLong(t *testing.T) {
	t.Parallel()

	v := bytesvalidator.BytesBetween(1, 3)
	req := validator.StringRequest{
		ConfigValue: types.StringValue("hello"),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for value above maximum, got none")
	}
}

func TestBytesBetween_NullValue(t *testing.T) {
	t.Parallel()

	v := bytesvalidator.BytesBetween(3, 10)
	req := validator.StringRequest{
		ConfigValue: types.StringNull(),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for null value, got: %s", resp.Diagnostics)
	}
}

func TestBytesBetween_UnknownValue(t *testing.T) {
	t.Parallel()

	v := bytesvalidator.BytesBetween(3, 10)
	req := validator.StringRequest{
		ConfigValue: types.StringUnknown(),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for unknown value, got: %s", resp.Diagnostics)
	}
}

func TestBytesBetween_InvalidState_NegativeMin(t *testing.T) {
	t.Parallel()

	v := bytesvalidator.BytesBetween(-1, 10)
	req := validator.StringRequest{
		ConfigValue: types.StringValue("hello"),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for invalid validator state (negative minBytes), got none")
	}
}

func TestBytesBetween_InvalidState_MinGreaterThanMax(t *testing.T) {
	t.Parallel()

	v := bytesvalidator.BytesBetween(10, 5)
	req := validator.StringRequest{
		ConfigValue: types.StringValue("hello"),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for invalid validator state (min > max), got none")
	}
}

func TestBytesBetween_Description(t *testing.T) {
	t.Parallel()

	v := bytesvalidator.BytesBetween(3, 10)
	desc := v.Description(context.Background())
	if desc == "" {
		t.Error("expected non-empty description")
	}
}

// TestBytesBetween_ParameterString verifies that function parameter
// validation matches attribute validation: null and unknown values pass,
// out-of-range lengths fail, and invalid bounds report an error.
func TestBytesBetween_ParameterString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		validator function.StringParameterValidator
		value     types.String
		wantErr   bool
	}{
		{"valid", bytesvalidator.BytesBetween(1, 10), types.StringValue("hello"), false},
		{"too_short", bytesvalidator.BytesBetween(6, 10), types.StringValue("hello"), true},
		{"too_long", bytesvalidator.BytesBetween(1, 3), types.StringValue("hello"), true},
		{"null", bytesvalidator.BytesBetween(1, 3), types.StringNull(), false},
		{"unknown", bytesvalidator.BytesBetween(1, 3), types.StringUnknown(), false},
		{"invalid_negative", bytesvalidator.BytesBetween(-1, 3), types.StringValue("x"), true},
		{"invalid_order", bytesvalidator.BytesBetween(5, 3), types.StringValue("x"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := function.StringParameterValidatorRequest{ArgumentPosition: 0, Value: tt.value}
			resp := &function.StringParameterValidatorResponse{}
			tt.validator.ValidateParameterString(context.Background(), req, resp)

			if gotErr := resp.Error != nil; gotErr != tt.wantErr {
				t.Errorf("ValidateParameterString() error = %v, wantErr %v", resp.Error, tt.wantErr)
			}
		})
	}
}

// TestBytesBetween_MarkdownDescription verifies that the markdown description
// equals the plain description.
func TestBytesBetween_MarkdownDescription(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	v := bytesvalidator.BytesBetween(1, 2)
	if got, want := v.MarkdownDescription(ctx), v.Description(ctx); got != want || got == "" {
		t.Errorf("MarkdownDescription() = %q, want %q", got, want)
	}
}
