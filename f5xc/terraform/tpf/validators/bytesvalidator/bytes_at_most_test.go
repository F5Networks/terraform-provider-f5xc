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
