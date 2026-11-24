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

func TestBytesAtMost_Valid(t *testing.T) {
	t.Parallel()

	v := bytesvalidator.BytesAtMost(10)
	req := validator.StringRequest{
		ConfigValue: types.StringValue("hello"),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for value within max bytes, got: %s", resp.Diagnostics)
	}
}

func TestBytesAtMost_Exact(t *testing.T) {
	t.Parallel()

	v := bytesvalidator.BytesAtMost(5)
	req := validator.StringRequest{
		ConfigValue: types.StringValue("hello"),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for value exactly at max bytes, got: %s", resp.Diagnostics)
	}
}

func TestBytesAtMost_TooLong(t *testing.T) {
	t.Parallel()

	v := bytesvalidator.BytesAtMost(3)
	req := validator.StringRequest{
		ConfigValue: types.StringValue("hello"),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for value exceeding max bytes, got none")
	}
}

func TestBytesAtMost_NullValue(t *testing.T) {
	t.Parallel()

	v := bytesvalidator.BytesAtMost(5)
	req := validator.StringRequest{
		ConfigValue: types.StringNull(),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for null value, got: %s", resp.Diagnostics)
	}
}

func TestBytesAtMost_UnknownValue(t *testing.T) {
	t.Parallel()

	v := bytesvalidator.BytesAtMost(5)
	req := validator.StringRequest{
		ConfigValue: types.StringUnknown(),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for unknown value, got: %s", resp.Diagnostics)
	}
}

func TestBytesAtMost_InvalidState(t *testing.T) {
	t.Parallel()

	v := bytesvalidator.BytesAtMost(-1)
	req := validator.StringRequest{
		ConfigValue: types.StringValue("hello"),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for invalid validator state (negative maxBytes), got none")
	}
}

func TestBytesAtMost_Description(t *testing.T) {
	t.Parallel()

	v := bytesvalidator.BytesAtMost(5)
	desc := v.Description(context.Background())
	if desc == "" {
		t.Error("expected non-empty description")
	}
}

// TestBytesAtMost_ParameterString verifies that function parameter
// validation matches attribute validation: null and unknown values pass, a
// too-long value fails, and a negative maximum reports an error.
func TestBytesAtMost_ParameterString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		validator function.StringParameterValidator
		value     types.String
		wantErr   bool
	}{
		{"valid", bytesvalidator.BytesAtMost(10), types.StringValue("hello"), false},
		{"too_long", bytesvalidator.BytesAtMost(2), types.StringValue("hello"), true},
		{"null", bytesvalidator.BytesAtMost(2), types.StringNull(), false},
		{"unknown", bytesvalidator.BytesAtMost(2), types.StringUnknown(), false},
		{"invalid_usage", bytesvalidator.BytesAtMost(-1), types.StringValue("x"), true},
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

// TestBytesAtMost_MarkdownDescription verifies that the markdown description
// equals the plain description.
func TestBytesAtMost_MarkdownDescription(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	v := bytesvalidator.BytesAtMost(2)
	if got, want := v.MarkdownDescription(ctx), v.Description(ctx); got != want || got == "" {
		t.Errorf("MarkdownDescription() = %q, want %q", got, want)
	}
}
