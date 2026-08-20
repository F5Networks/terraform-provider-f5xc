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

// TestImmutableObject_Constructor verifies ImmutableObject() returns a planmodifier.Object implementation.
func TestImmutableObject_Constructor(t *testing.T) {
	m := ImmutableObject()
	assert.Implements(t, (*planmodifier.Object)(nil), m)
}

// TestImmutableObject_Description verifies Description returns a non-empty string.
func TestImmutableObject_Description(t *testing.T) {
	m := immutableObjectModifier{}
	assert.NotEmpty(t, m.Description(context.Background()))
}

// TestImmutableObject_MarkdownDescription verifies MarkdownDescription delegates to Description.
func TestImmutableObject_MarkdownDescription(t *testing.T) {
	m := immutableObjectModifier{}
	assert.Equal(t, m.Description(context.Background()), m.MarkdownDescription(context.Background()))
}

var testObjAttrTypes = map[string]attr.Type{
	"expressions": types.ListType{ElemType: types.StringType},
}

func mustObjectValue(attrs map[string]attr.Value) types.Object {
	obj, _ := types.ObjectValue(testObjAttrTypes, attrs)
	return obj
}

func exprList(vals ...string) attr.Value {
	elems := make([]attr.Value, len(vals))
	for i, v := range vals {
		elems[i] = types.StringValue(v)
	}
	l, _ := types.ListValue(types.StringType, elems)
	return l
}

// TestPlanModifyObject_NullState_NoError verifies that on first create (state=null)
// any plan value is allowed without error.
func TestPlanModifyObject_NullState_NoError(t *testing.T) {
	m := immutableObjectModifier{}
	planVal := mustObjectValue(map[string]attr.Value{
		"expressions": exprList("env == production"),
	})
	req := planmodifier.ObjectRequest{
		StateValue: types.ObjectNull(testObjAttrTypes),
		PlanValue:  planVal,
	}
	resp := &planmodifier.ObjectResponse{PlanValue: req.PlanValue}

	m.PlanModifyObject(context.Background(), req, resp)

	assert.False(t, resp.Diagnostics.HasError(), "expected no error on first create (state=null)")
}

// TestPlanModifyObject_UnknownState_NoError verifies that when state is unknown no error is surfaced.
func TestPlanModifyObject_UnknownState_NoError(t *testing.T) {
	m := immutableObjectModifier{}
	planVal := mustObjectValue(map[string]attr.Value{
		"expressions": exprList("env == staging"),
	})
	req := planmodifier.ObjectRequest{
		StateValue: types.ObjectUnknown(testObjAttrTypes),
		PlanValue:  planVal,
	}
	resp := &planmodifier.ObjectResponse{PlanValue: req.PlanValue}

	m.PlanModifyObject(context.Background(), req, resp)

	assert.False(t, resp.Diagnostics.HasError(), "expected no error when state is unknown")
}

// TestPlanModifyObject_UnknownPlan_NoError verifies that when the plan value is unknown no error is surfaced.
func TestPlanModifyObject_UnknownPlan_NoError(t *testing.T) {
	m := immutableObjectModifier{}
	stateVal := mustObjectValue(map[string]attr.Value{
		"expressions": exprList("env == production"),
	})
	req := planmodifier.ObjectRequest{
		StateValue: stateVal,
		PlanValue:  types.ObjectUnknown(testObjAttrTypes),
	}
	resp := &planmodifier.ObjectResponse{PlanValue: req.PlanValue}

	m.PlanModifyObject(context.Background(), req, resp)

	assert.False(t, resp.Diagnostics.HasError(), "expected no error when plan is unknown")
}

// TestPlanModifyObject_NoChange_NoError verifies that when plan equals state no error is surfaced.
func TestPlanModifyObject_NoChange_NoError(t *testing.T) {
	m := immutableObjectModifier{}
	val := mustObjectValue(map[string]attr.Value{
		"expressions": exprList("env == production"),
	})
	req := planmodifier.ObjectRequest{
		StateValue: val,
		PlanValue:  val,
	}
	resp := &planmodifier.ObjectResponse{PlanValue: req.PlanValue}

	m.PlanModifyObject(context.Background(), req, resp)

	assert.False(t, resp.Diagnostics.HasError(), "expected no error when plan equals state")
}

// TestPlanModifyObject_Change_Error verifies that when the plan differs from state
// an error diagnostic is added.
func TestPlanModifyObject_Change_Error(t *testing.T) {
	m := immutableObjectModifier{}
	stateVal := mustObjectValue(map[string]attr.Value{
		"expressions": exprList("env == production"),
	})
	planVal := mustObjectValue(map[string]attr.Value{
		"expressions": exprList("env == staging"),
	})
	req := planmodifier.ObjectRequest{
		StateValue: stateVal,
		PlanValue:  planVal,
	}
	resp := &planmodifier.ObjectResponse{PlanValue: req.PlanValue}

	m.PlanModifyObject(context.Background(), req, resp)

	assert.True(t, resp.Diagnostics.HasError(), "expected error when plan differs from state")
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Immutable Attribute Change")
}
