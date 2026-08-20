//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

package portsvalidator_test

import (
	"context"
	"testing"

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
