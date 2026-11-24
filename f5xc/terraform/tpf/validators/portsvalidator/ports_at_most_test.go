//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

package portsvalidator_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/validators/portsvalidator"
)

func TestPortsAtMost_SinglePort_Valid(t *testing.T) {
	t.Parallel()

	v := portsvalidator.PortsAtMost(3)
	req := validator.StringRequest{
		ConfigValue: types.StringValue("80"),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for single port within limit, got: %s", resp.Diagnostics)
	}
}

func TestPortsAtMost_MultiplePorts_Valid(t *testing.T) {
	t.Parallel()

	v := portsvalidator.PortsAtMost(3)
	req := validator.StringRequest{
		ConfigValue: types.StringValue("80,443,8080"),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for ports within limit, got: %s", resp.Diagnostics)
	}
}

func TestPortsAtMost_ExactLimit(t *testing.T) {
	t.Parallel()

	v := portsvalidator.PortsAtMost(2)
	req := validator.StringRequest{
		ConfigValue: types.StringValue("80,443"),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for ports exactly at limit, got: %s", resp.Diagnostics)
	}
}

func TestPortsAtMost_TooManyPorts(t *testing.T) {
	t.Parallel()

	v := portsvalidator.PortsAtMost(2)
	req := validator.StringRequest{
		ConfigValue: types.StringValue("80,443,8080"),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for too many ports, got none")
	}
}

func TestPortsAtMost_NullValue(t *testing.T) {
	t.Parallel()

	v := portsvalidator.PortsAtMost(3)
	req := validator.StringRequest{
		ConfigValue: types.StringNull(),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for null value, got: %s", resp.Diagnostics)
	}
}

func TestPortsAtMost_UnknownValue(t *testing.T) {
	t.Parallel()

	v := portsvalidator.PortsAtMost(3)
	req := validator.StringRequest{
		ConfigValue: types.StringUnknown(),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for unknown value, got: %s", resp.Diagnostics)
	}
}

func TestPortsAtMost_InvalidState(t *testing.T) {
	t.Parallel()

	v := portsvalidator.PortsAtMost(-1)
	req := validator.StringRequest{
		ConfigValue: types.StringValue("80"),
	}
	resp := &validator.StringResponse{}

	v.ValidateString(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for invalid validator state (negative maxPorts), got none")
	}
}

func TestPortsAtMost_Description(t *testing.T) {
	t.Parallel()

	v := portsvalidator.PortsAtMost(3)
	desc := v.Description(context.Background())
	if desc == "" {
		t.Error("expected non-empty description")
	}
}

// TestPortsAtMost_ParameterString verifies that function parameter validation
// skips null and unknown values, rejects too many ports, and reports an
// invalid negative limit.
func TestPortsAtMost_ParameterString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		max     int
		value   types.String
		wantErr bool
	}{
		{"valid", 3, types.StringValue("80,443"), false},
		{"too_many", 1, types.StringValue("80,443"), true},
		{"null", 1, types.StringNull(), false},
		{"unknown", 1, types.StringUnknown(), false},
		{"invalid_usage", -1, types.StringValue("80"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := function.StringParameterValidatorRequest{Value: tt.value}
			resp := &function.StringParameterValidatorResponse{}
			portsvalidator.PortsAtMost(tt.max).ValidateParameterString(context.Background(), req, resp)

			if gotErr := resp.Error != nil; gotErr != tt.wantErr {
				t.Errorf("ValidateParameterString() error = %v, wantErr %v", resp.Error, tt.wantErr)
			}
		})
	}
}

// TestPortsAtMost_MarkdownDescription verifies that the markdown description
// equals the plain description.
func TestPortsAtMost_MarkdownDescription(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	v := portsvalidator.PortsAtMost(2)
	if got, want := v.MarkdownDescription(ctx), v.Description(ctx); got != want {
		t.Errorf("MarkdownDescription() = %q, want %q", got, want)
	}
}
