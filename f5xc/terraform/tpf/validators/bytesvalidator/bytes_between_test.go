//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

package bytesvalidator_test

import (
	"context"
	"testing"

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
