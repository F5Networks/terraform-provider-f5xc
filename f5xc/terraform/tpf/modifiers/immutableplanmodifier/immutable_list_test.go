// Copyright (c) 2026 F5 Inc. All rights reserved.

package immutableplanmodifier

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

// TestImmutableList_Constructor verifies ImmutableList() returns a planmodifier.List implementation.
func TestImmutableList_Constructor(t *testing.T) {
	m := ImmutableList()
	assert.Implements(t, (*planmodifier.List)(nil), m)
}

// TestImmutableList_Description verifies Description returns a non-empty string.
func TestImmutableList_Description(t *testing.T) {
	m := immutableListModifier{}
	desc := m.Description(context.Background())
	assert.NotEmpty(t, desc)
}

// TestImmutableList_MarkdownDescription verifies MarkdownDescription delegates to Description.
func TestImmutableList_MarkdownDescription(t *testing.T) {
	m := immutableListModifier{}
	assert.Equal(t, m.Description(context.Background()), m.MarkdownDescription(context.Background()))
}

// TestPlanModifyList_NullState_Create verifies that on first create (state=null)
// any plan value is allowed without error.
func TestPlanModifyList_NullState_Create(t *testing.T) {
	m := immutableListModifier{}

	planVal, _ := types.ListValue(types.ObjectType{AttrTypes: map[string]attr.Type{
		"name":      types.StringType,
		"namespace": types.StringType,
	}}, []attr.Value{})

	req := planmodifier.ListRequest{
		StateValue: types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{
			"name":      types.StringType,
			"namespace": types.StringType,
		}}),
		PlanValue: planVal,
	}
	resp := &planmodifier.ListResponse{PlanValue: req.PlanValue}

	m.PlanModifyList(context.Background(), req, resp)

	assert.False(t, resp.Diagnostics.HasError(), "expected no error on first create (state=null)")
}

// TestPlanModifyList_UnknownState_Create verifies that when state is unknown
// (e.g. during interpolation on first create) any plan value is allowed.
func TestPlanModifyList_UnknownState_Create(t *testing.T) {
	m := immutableListModifier{}

	planVal, _ := types.ListValue(types.ObjectType{AttrTypes: map[string]attr.Type{
		"name": types.StringType,
	}}, []attr.Value{})

	req := planmodifier.ListRequest{
		StateValue: types.ListUnknown(types.ObjectType{AttrTypes: map[string]attr.Type{
			"name": types.StringType,
		}}),
		PlanValue: planVal,
	}
	resp := &planmodifier.ListResponse{PlanValue: req.PlanValue}

	m.PlanModifyList(context.Background(), req, resp)

	assert.False(t, resp.Diagnostics.HasError(), "expected no error when state is unknown")
}

// TestPlanModifyList_UnknownPlan_NoError verifies that when the plan value is unknown
// (e.g. during interpolation) no error is surfaced.
func TestPlanModifyList_UnknownPlan_NoError(t *testing.T) {
	m := immutableListModifier{}

	stateVal, _ := types.ListValue(types.ObjectType{AttrTypes: map[string]attr.Type{
		"name": types.StringType,
	}}, []attr.Value{})

	req := planmodifier.ListRequest{
		StateValue: stateVal,
		PlanValue: types.ListUnknown(types.ObjectType{AttrTypes: map[string]attr.Type{
			"name": types.StringType,
		}}),
	}
	resp := &planmodifier.ListResponse{PlanValue: req.PlanValue}

	m.PlanModifyList(context.Background(), req, resp)

	assert.False(t, resp.Diagnostics.HasError(), "expected no error when plan is unknown")
}

// TestPlanModifyList_NoChange_NoError verifies that when plan equals state no error is surfaced.
func TestPlanModifyList_NoChange_NoError(t *testing.T) {
	m := immutableListModifier{}

	elemType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"name":      types.StringType,
		"namespace": types.StringType,
	}}
	elem, _ := types.ObjectValue(map[string]attr.Type{
		"name":      types.StringType,
		"namespace": types.StringType,
	}, map[string]attr.Value{
		"name":      types.StringValue("my-intf"),
		"namespace": types.StringValue("system"),
	})
	listVal, _ := types.ListValue(elemType, []attr.Value{elem})

	req := planmodifier.ListRequest{
		StateValue: listVal,
		PlanValue:  listVal,
	}
	resp := &planmodifier.ListResponse{PlanValue: req.PlanValue}

	m.PlanModifyList(context.Background(), req, resp)

	assert.False(t, resp.Diagnostics.HasError(), "expected no error when plan equals state")
}

// TestPlanModifyList_Change_Error verifies that when the plan differs from state
// an error diagnostic is added.
func TestPlanModifyList_Change_Error(t *testing.T) {
	m := immutableListModifier{}

	elemType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"name":      types.StringType,
		"namespace": types.StringType,
	}}

	stateElem, _ := types.ObjectValue(map[string]attr.Type{
		"name":      types.StringType,
		"namespace": types.StringType,
	}, map[string]attr.Value{
		"name":      types.StringValue("old-intf"),
		"namespace": types.StringValue("system"),
	})
	stateVal, _ := types.ListValue(elemType, []attr.Value{stateElem})

	planElem, _ := types.ObjectValue(map[string]attr.Type{
		"name":      types.StringType,
		"namespace": types.StringType,
	}, map[string]attr.Value{
		"name":      types.StringValue("new-intf"),
		"namespace": types.StringValue("system"),
	})
	planVal, _ := types.ListValue(elemType, []attr.Value{planElem})

	req := planmodifier.ListRequest{
		StateValue: stateVal,
		PlanValue:  planVal,
	}
	resp := &planmodifier.ListResponse{PlanValue: req.PlanValue}

	m.PlanModifyList(context.Background(), req, resp)

	assert.True(t, resp.Diagnostics.HasError(), "expected error when plan differs from state")
	assert.Contains(t, resp.Diagnostics[0].Summary(), "Immutable Attribute Change")
}
