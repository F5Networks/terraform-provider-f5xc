//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package inlinereferencevalidator_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/validators/inlinereferencevalidator"
)

func TestInlineReferenceValidator_ForbiddenChoiceTriggersError(t *testing.T) {
	v := inlinereferencevalidator.RestrictedInlineReference(
		[]string{"active_service_policies"},
		"f5xc_http_loadbalancer_service_policy_attachments",
	)

	attrTypes := map[string]attr.Type{
		"active_service_policies":         types.ObjectType{AttrTypes: map[string]attr.Type{}},
		"no_service_policies":             types.BoolType,
		"service_policies_from_namespace": types.BoolType,
	}

	// Case 1: active_service_policies is configured (triggers error)
	activeConfigVal, _ := types.ObjectValue(attrTypes, map[string]attr.Value{
		"active_service_policies":         types.ObjectValueMust(map[string]attr.Type{}, map[string]attr.Value{}),
		"no_service_policies":             types.BoolNull(),
		"service_policies_from_namespace": types.BoolNull(),
	})

	req := validator.ObjectRequest{
		Path:        path.Root("service_policy_choice"),
		ConfigValue: activeConfigVal,
	}
	var resp validator.ObjectResponse

	v.ValidateObject(context.Background(), req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostic error for active_service_policies, got none")
	}

	errDiag := resp.Diagnostics.Errors()[0]
	if errDiag.Summary() != "Conflicting Service Policy Configuration" {
		t.Errorf("unexpected summary: %s", errDiag.Summary())
	}

	// Case 2: no_service_policies is configured (valid inline choice - no error)
	nspConfigVal, _ := types.ObjectValue(attrTypes, map[string]attr.Value{
		"active_service_policies":         types.ObjectNull(map[string]attr.Type{}),
		"no_service_policies":             types.BoolValue(true),
		"service_policies_from_namespace": types.BoolNull(),
	})

	reqNsp := validator.ObjectRequest{
		Path:        path.Root("service_policy_choice"),
		ConfigValue: nspConfigVal,
	}
	var respNsp validator.ObjectResponse

	v.ValidateObject(context.Background(), reqNsp, &respNsp)

	if respNsp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostic error for no_service_policies, got: %v", respNsp.Diagnostics.Errors())
	}

	// Case 3: service_policy_choice is null (omitted in HCL - valid when using attachment resource)
	nullConfigVal := types.ObjectNull(attrTypes)
	reqNull := validator.ObjectRequest{
		Path:        path.Root("service_policy_choice"),
		ConfigValue: nullConfigVal,
	}
	var respNull validator.ObjectResponse

	v.ValidateObject(context.Background(), reqNull, &respNull)

	if respNull.Diagnostics.HasError() {
		t.Fatalf("expected no errors for null service_policy_choice, got: %v", respNull.Diagnostics.Errors())
	}
}
