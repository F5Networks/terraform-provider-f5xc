// Copyright (c) 2026 F5 Inc. All rights reserved.

package oneofbranchplanmodifier

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPreserveSelectedBranch verifies the constructor returns a planmodifier.Object implementation.
func TestPreserveSelectedBranch(t *testing.T) {
	m := PreserveSelectedBranch()
	assert.Implements(t, (*planmodifier.Object)(nil), m)
}

// TestPreserveSelectedBranch_Description verifies Description returns a non-empty string.
func TestPreserveSelectedBranch_Description(t *testing.T) {
	m := preserveSelectedBranchModifier{}
	assert.NotEmpty(t, m.Description(context.Background()))
}

// TestPreserveSelectedBranch_MarkdownDescription verifies MarkdownDescription delegates to Description.
func TestPreserveSelectedBranch_MarkdownDescription(t *testing.T) {
	m := preserveSelectedBranchModifier{}
	assert.Equal(t, m.Description(context.Background()), m.MarkdownDescription(context.Background()))
}

// TestPlanModifyObject_UnknownConfig verifies interpolation/unknown config leaves the plan unchanged.
func TestPlanModifyObject_UnknownConfig(t *testing.T) {
	m := preserveSelectedBranchModifier{}
	attrTypes := endpointChoiceAttrTypes()
	planned := endpointChoiceObject(t, map[string]attr.Value{
		"any":         types.BoolValue(true),
		"prefix_list": types.ObjectUnknown(prefixListAttrTypes()),
	})

	req := planmodifier.ObjectRequest{
		ConfigValue: types.ObjectUnknown(attrTypes),
		PlanValue:   planned,
	}
	resp := &planmodifier.ObjectResponse{PlanValue: req.PlanValue}

	m.PlanModifyObject(context.Background(), req, resp)

	assert.True(t, planned.Equal(resp.PlanValue))
	assert.False(t, resp.Diagnostics.HasError())
}

// TestPlanModifyObject_NullConfig verifies an unconfigured oneof object is left unchanged.
func TestPlanModifyObject_NullConfig(t *testing.T) {
	m := preserveSelectedBranchModifier{}
	attrTypes := endpointChoiceAttrTypes()
	planned := types.ObjectUnknown(attrTypes)

	req := planmodifier.ObjectRequest{
		ConfigValue: types.ObjectNull(attrTypes),
		PlanValue:   planned,
	}
	resp := &planmodifier.ObjectResponse{PlanValue: req.PlanValue}

	m.PlanModifyObject(context.Background(), req, resp)

	assert.True(t, planned.Equal(resp.PlanValue))
	assert.False(t, resp.Diagnostics.HasError())
}

// TestPlanModifyObject_NormalizesCanonicalizedSibling verifies the configured branch wins
// over a canonicalized sibling branch that leaked into the plan.
func TestPlanModifyObject_NormalizesCanonicalizedSibling(t *testing.T) {
	m := preserveSelectedBranchModifier{}
	config := endpointChoiceObject(t, map[string]attr.Value{
		"any":         types.BoolValue(true),
		"prefix_list": types.ObjectNull(prefixListAttrTypes()),
	})
	planned := endpointChoiceObject(t, map[string]attr.Value{
		"any":         types.BoolValue(true),
		"prefix_list": types.ObjectUnknown(prefixListAttrTypes()),
	})
	expected := endpointChoiceObject(t, map[string]attr.Value{
		"any":         types.BoolValue(true),
		"prefix_list": types.ObjectNull(prefixListAttrTypes()),
	})

	req := planmodifier.ObjectRequest{
		ConfigValue: config,
		PlanValue:   planned,
	}
	resp := &planmodifier.ObjectResponse{PlanValue: req.PlanValue}

	m.PlanModifyObject(context.Background(), req, resp)

	assert.True(t, expected.Equal(resp.PlanValue))
	assert.False(t, resp.Diagnostics.HasError())
}

// TestPlanModifyObject_PreservesSelectedPlanBranch verifies the selected branch value comes
// from the plan so legitimate data inside that branch is retained while siblings are cleared.
func TestPlanModifyObject_PreservesSelectedPlanBranch(t *testing.T) {
	m := preserveSelectedBranchModifier{}
	selectedPrefixList := prefixListObject(t, map[string]attr.Value{
		"prefixes":      stringListValue(t, "0.0.0.0/0"),
		"ipv6_prefixes": types.ListNull(types.StringType),
	})
	config := endpointChoiceObject(t, map[string]attr.Value{
		"any":         types.BoolNull(),
		"prefix_list": selectedPrefixList,
	})
	planned := endpointChoiceObject(t, map[string]attr.Value{
		"any":         types.BoolUnknown(),
		"prefix_list": selectedPrefixList,
	})
	expected := endpointChoiceObject(t, map[string]attr.Value{
		"any":         types.BoolNull(),
		"prefix_list": selectedPrefixList,
	})

	req := planmodifier.ObjectRequest{
		ConfigValue: config,
		PlanValue:   planned,
	}
	resp := &planmodifier.ObjectResponse{PlanValue: req.PlanValue}

	m.PlanModifyObject(context.Background(), req, resp)

	assert.True(t, expected.Equal(resp.PlanValue))
	assert.False(t, resp.Diagnostics.HasError())
}

// TestPlanModifyObject_MultipleConfiguredBranches verifies invalid multi-branch config is not rewritten.
func TestPlanModifyObject_MultipleConfiguredBranches(t *testing.T) {
	m := preserveSelectedBranchModifier{}
	config := endpointChoiceObject(t, map[string]attr.Value{
		"any": types.BoolValue(true),
		"prefix_list": prefixListObject(t, map[string]attr.Value{
			"prefixes":      stringListValue(t, "0.0.0.0/0"),
			"ipv6_prefixes": types.ListNull(types.StringType),
		}),
	})
	planned := types.ObjectUnknown(endpointChoiceAttrTypes())

	req := planmodifier.ObjectRequest{
		ConfigValue: config,
		PlanValue:   planned,
	}
	resp := &planmodifier.ObjectResponse{PlanValue: req.PlanValue}

	m.PlanModifyObject(context.Background(), req, resp)

	assert.True(t, planned.Equal(resp.PlanValue))
	assert.False(t, resp.Diagnostics.HasError())
}

// endpointChoiceAttrTypes returns the minimal oneof object shape used by the modifier tests.
func endpointChoiceAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"any":         types.BoolType,
		"prefix_list": types.ObjectType{AttrTypes: prefixListAttrTypes()},
	}
}

// prefixListAttrTypes returns the nested object shape used by the prefix_list branch tests.
func prefixListAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"prefixes":      types.ListType{ElemType: types.StringType},
		"ipv6_prefixes": types.ListType{ElemType: types.StringType},
	}
}

// endpointChoiceObject creates a oneof object value for tests.
func endpointChoiceObject(t *testing.T, attrs map[string]attr.Value) types.Object {
	t.Helper()
	obj, diags := types.ObjectValue(endpointChoiceAttrTypes(), attrs)
	require.False(t, diags.HasError(), "failed to create endpoint choice object: %v", diags)
	return obj
}

// prefixListObject creates a prefix_list branch object value for tests.
func prefixListObject(t *testing.T, attrs map[string]attr.Value) types.Object {
	t.Helper()
	obj, diags := types.ObjectValue(prefixListAttrTypes(), attrs)
	require.False(t, diags.HasError(), "failed to create prefix_list object: %v", diags)
	return obj
}

// stringListValue creates a known list of string values for tests.
func stringListValue(t *testing.T, values ...string) types.List {
	t.Helper()
	elems := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elems = append(elems, types.StringValue(value))
	}
	list, diags := types.ListValue(types.StringType, elems)
	require.False(t, diags.HasError(), "failed to create string list: %v", diags)
	return list
}
