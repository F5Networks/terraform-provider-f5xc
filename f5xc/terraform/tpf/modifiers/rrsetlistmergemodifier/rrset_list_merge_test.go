//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

package rrsetlistmergemodifier

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew verifies the constructor returns a planmodifier.List implementation.
func TestNew(t *testing.T) {
	m := New()
	assert.Implements(t, (*planmodifier.List)(nil), m)
}

// TestRRSetListMergeModifier_Description verifies Description returns a non-empty string.
func TestRRSetListMergeModifier_Description(t *testing.T) {
	m := rrsetListMergeModifier{}
	assert.NotEmpty(t, m.Description(context.Background()))
	assert.Equal(t, m.Description(context.Background()), m.MarkdownDescription(context.Background()))
}

// TestPlanModifyList_UnchangedConfigWithBackendRRSet verifies the modifier does
// not pin prior state: a state-only RRSet must surface as drift, so the plan
// follows the config rather than the (larger) prior state.
func TestPlanModifyList_UnchangedConfigWithBackendRRSet(t *testing.T) {
	m := rrsetListMergeModifier{}

	configList := rrsetList(t,
		aRecordRRSet(t, "www.example.com", "1.2.3.4"),
	)
	stateList := rrsetList(t,
		aRecordRRSet(t, "www.example.com", "1.2.3.4"),
		aliasRecordRRSet(t, "lb.example.net"),
	)

	resp := runPlanModifyList(t, m, configList, stateList)
	assert.True(t, configList.Equal(resp.PlanValue))
	assert.False(t, resp.Diagnostics.HasError())
}

// TestPlanModifyList_ChangedConfiguredRRSet verifies user changes keep the plan at config.
func TestPlanModifyList_ChangedConfiguredRRSet(t *testing.T) {
	m := rrsetListMergeModifier{}

	configList := rrsetList(t,
		aRecordRRSet(t, "www.example.com", "5.6.7.8"),
	)
	stateList := rrsetList(t,
		aRecordRRSet(t, "www.example.com", "1.2.3.4"),
		aliasRecordRRSet(t, "lb.example.net"),
	)

	resp := runPlanModifyList(t, m, configList, stateList)
	assert.True(t, configList.Equal(resp.PlanValue))
	assert.False(t, resp.Diagnostics.HasError())
}

// TestPlanModifyList_AddedConfiguredRRSet verifies adding a configured RRSet keeps plan=config.
func TestPlanModifyList_AddedConfiguredRRSet(t *testing.T) {
	m := rrsetListMergeModifier{}

	configList := rrsetList(t,
		aRecordRRSet(t, "www.example.com", "1.2.3.4"),
		aaaaRecordRRSet(t, "api.example.com", "2001:db8::1"),
	)
	stateList := rrsetList(t,
		aRecordRRSet(t, "www.example.com", "1.2.3.4"),
	)

	resp := runPlanModifyList(t, m, configList, stateList)
	assert.True(t, configList.Equal(resp.PlanValue))
	assert.False(t, resp.Diagnostics.HasError())
}

// TestPlanModifyList_CreateUsesPlannedValue verifies create-time fallbacks keep
// the framework-computed plan so nested defaults such as ttl are preserved.
func TestPlanModifyList_CreateUsesPlannedValue(t *testing.T) {
	m := rrsetListMergeModifier{}

	configList := rrsetList(t,
		rrsetObject(t, map[string]attr.Value{
			"description": types.StringValue("rrset"),
			"ttl":         types.Int64Null(),
			"type_record_set_choice": typeRecordSetChoiceObject(t, map[string]attr.Value{
				"a_record": branchObject(t, map[string]attr.Value{
					"name":   types.StringValue("www.example.com"),
					"values": stringListValue(t, "1.2.3.4"),
				}),
				"aaaa_record":  types.ObjectNull(aaaaRecordAttrTypes()),
				"alias_record": types.ObjectNull(aliasRecordAttrTypes()),
				"mx_record":    types.ObjectNull(mxRecordAttrTypes()),
			}),
		}),
	)
	planList := rrsetList(t,
		aRecordRRSet(t, "www.example.com", "1.2.3.4"),
	)

	resp := runPlanModifyListWithPlan(
		t,
		m,
		configList,
		types.ListNull(types.ObjectType{AttrTypes: rrsetAttrTypes()}),
		planList,
	)
	assert.True(t, planList.Equal(resp.PlanValue))
	assert.False(t, resp.Diagnostics.HasError())
}

// TestPlanModifyList_StateOnlyRRSetShownAsDrift verifies the modifier no longer
// preserves state-only RRsets: an unexpected backend addition must surface as
// drift, so the plan follows config and the extra state record is dropped.
func TestPlanModifyList_StateOnlyRRSetShownAsDrift(t *testing.T) {
	m := rrsetListMergeModifier{}

	configList := rrsetList(t,
		aRecordRRSet(t, "www.example.com", "1.2.3.4"),
	)
	stateList := rrsetList(t,
		aRecordRRSet(t, "www.example.com", "1.2.3.4"),
		aaaaRecordRRSet(t, "api.example.com", "2001:db8::1"),
	)

	resp := runPlanModifyList(t, m, configList, stateList)
	assert.True(t, configList.Equal(resp.PlanValue))
	assert.False(t, resp.Diagnostics.HasError())
}

// TestPlanModifyList_ConfigNullStateNull verifies an unconfigured, unstored list
// plans to null.
func TestPlanModifyList_ConfigNullStateNull(t *testing.T) {
	m := rrsetListMergeModifier{}

	nullList := types.ListNull(types.ObjectType{AttrTypes: rrsetAttrTypes()})

	resp := runPlanModifyListWithPlan(t, m, nullList, nullList, nullList)
	assert.True(t, resp.PlanValue.IsNull())
	assert.False(t, resp.Diagnostics.HasError())
}

// TestPlanModifyList_ConfigNullStatePresent verifies removing a previously
// configured list forces a re-read (plan unknown) so the removal reconciles.
func TestPlanModifyList_ConfigNullStatePresent(t *testing.T) {
	m := rrsetListMergeModifier{}

	nullList := types.ListNull(types.ObjectType{AttrTypes: rrsetAttrTypes()})
	stateList := rrsetList(t,
		aRecordRRSet(t, "www.example.com", "1.2.3.4"),
	)

	resp := runPlanModifyListWithPlan(t, m, nullList, stateList, nullList)
	assert.True(t, resp.PlanValue.IsUnknown())
	assert.False(t, resp.Diagnostics.HasError())
}

// TestPlanModifyList_ConfigUnknown verifies an unknown config leaves the framework
// plan untouched.
func TestPlanModifyList_ConfigUnknown(t *testing.T) {
	m := rrsetListMergeModifier{}

	unknownList := types.ListUnknown(types.ObjectType{AttrTypes: rrsetAttrTypes()})
	planList := rrsetList(t,
		aRecordRRSet(t, "www.example.com", "1.2.3.4"),
	)

	resp := runPlanModifyListWithPlan(t, m, unknownList, planList, planList)
	assert.True(t, planList.Equal(resp.PlanValue))
	assert.False(t, resp.Diagnostics.HasError())
}

// TestPlanModifyList_ChangedConfiguredRRSetUsesPlannedValue verifies mismatch
// fallbacks keep the framework-computed plan, not the raw config.
func TestPlanModifyList_ChangedConfiguredRRSetUsesPlannedValue(t *testing.T) {
	m := rrsetListMergeModifier{}

	configList := rrsetList(t,
		rrsetObject(t, map[string]attr.Value{
			"description": types.StringValue("rrset"),
			"ttl":         types.Int64Null(),
			"type_record_set_choice": typeRecordSetChoiceObject(t, map[string]attr.Value{
				"a_record": branchObject(t, map[string]attr.Value{
					"name":   types.StringValue("www.example.com"),
					"values": stringListValue(t, "5.6.7.8"),
				}),
				"aaaa_record":  types.ObjectNull(aaaaRecordAttrTypes()),
				"alias_record": types.ObjectNull(aliasRecordAttrTypes()),
				"mx_record":    types.ObjectNull(mxRecordAttrTypes()),
			}),
		}),
	)
	stateList := rrsetList(t,
		aRecordRRSet(t, "www.example.com", "1.2.3.4"),
	)
	planList := rrsetList(t,
		aRecordRRSet(t, "www.example.com", "5.6.7.8"),
	)

	resp := runPlanModifyListWithPlan(t, m, configList, stateList, planList)
	assert.True(t, planList.Equal(resp.PlanValue))
	assert.False(t, resp.Diagnostics.HasError())
}

// TestPlanModifyList_RRSetGroup_ConfigNullStatePresentPlansNull verifies that removing
// a non-computed rr_set_group list from HCL (config null, state non-null) plans to null
// rather than unknown so Terraform does not reject the plan.
//
// Why: In f5xc_dns_zone, rr_set_group is Optional: true, Computed: false. Returning an
// unknown plan value for a non-computed attribute when it is removed from configuration
// is rejected by Terraform core.
//
// How: Constructs an rr_set_group element type (containing "metadata" and "rr_set"),
// populates state with one group while config is null, and asserts PlanValue.IsNull().
//
// Parameters:
//   - t: testing context.
//
// Returns:
//   - None.
func TestPlanModifyList_RRSetGroup_ConfigNullStatePresentPlansNull(t *testing.T) {
	m := rrsetListMergeModifier{}

	groupElemType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"metadata": types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"name":        types.StringType,
					"description": types.StringType,
				},
			},
			"rr_set": types.ListType{ElemType: types.ObjectType{AttrTypes: rrsetAttrTypes()}},
		},
	}

	metaObj, diags := types.ObjectValue(
		map[string]attr.Type{
			"name":        types.StringType,
			"description": types.StringType,
		},
		map[string]attr.Value{
			"name":        types.StringValue("g1"),
			"description": types.StringNull(),
		},
	)
	require.False(t, diags.HasError())

	groupObj, diags := types.ObjectValue(
		groupElemType.AttrTypes,
		map[string]attr.Value{
			"metadata": metaObj,
			"rr_set":   rrsetList(t, aRecordRRSet(t, "www.example.com", "1.2.3.4")),
		},
	)
	require.False(t, diags.HasError())

	nullList := types.ListNull(groupElemType)
	stateList, diags := types.ListValue(groupElemType, []attr.Value{groupObj})
	require.False(t, diags.HasError())

	resp := runPlanModifyListWithPlan(t, m, nullList, stateList, nullList)
	assert.True(t, resp.PlanValue.IsNull())
	assert.False(t, resp.PlanValue.IsUnknown())
	assert.False(t, resp.Diagnostics.HasError())
}

// runPlanModifyList executes the modifier with the given config/state list values.
func runPlanModifyList(t *testing.T, modifier rrsetListMergeModifier, configValue, stateValue types.List) *planmodifier.ListResponse {
	t.Helper()
	return runPlanModifyListWithPlan(t, modifier, configValue, stateValue, configValue)
}

// runPlanModifyListWithPlan executes the modifier with an explicit incoming plan value.
func runPlanModifyListWithPlan(t *testing.T, modifier rrsetListMergeModifier, configValue, stateValue, planValue types.List) *planmodifier.ListResponse {
	t.Helper()

	req := planmodifier.ListRequest{
		ConfigValue: configValue,
		StateValue:  stateValue,
		PlanValue:   planValue,
	}
	resp := &planmodifier.ListResponse{PlanValue: req.PlanValue}

	modifier.PlanModifyList(context.Background(), req, resp)

	return resp
}

// rrsetList creates a typed RRSet list value for tests.
func rrsetList(t *testing.T, rrsets ...types.Object) types.List {
	t.Helper()

	elements := make([]attr.Value, 0, len(rrsets))
	for _, rrset := range rrsets {
		elements = append(elements, rrset)
	}

	list, diags := types.ListValue(types.ObjectType{AttrTypes: rrsetAttrTypes()}, elements)
	require.False(t, diags.HasError(), "failed to create rrset list: %v", diags)
	return list
}

// aRecordRRSet creates an A record RRSet object.
func aRecordRRSet(t *testing.T, name, value string) types.Object {
	t.Helper()
	return rrsetObject(t, map[string]attr.Value{
		"description": types.StringValue("rrset"),
		"ttl":         types.Int64Value(3600),
		"type_record_set_choice": typeRecordSetChoiceObject(t, map[string]attr.Value{
			"a_record": branchObject(t, map[string]attr.Value{
				"name":   types.StringValue(name),
				"values": stringListValue(t, value),
			}),
			"aaaa_record":  types.ObjectNull(aaaaRecordAttrTypes()),
			"alias_record": types.ObjectNull(aliasRecordAttrTypes()),
			"mx_record":    types.ObjectNull(mxRecordAttrTypes()),
		}),
	})
}

// aaaaRecordRRSet creates an AAAA record RRSet object.
func aaaaRecordRRSet(t *testing.T, name, value string) types.Object {
	t.Helper()
	return rrsetObject(t, map[string]attr.Value{
		"description": types.StringValue("rrset"),
		"ttl":         types.Int64Value(3600),
		"type_record_set_choice": typeRecordSetChoiceObject(t, map[string]attr.Value{
			"a_record": types.ObjectNull(aRecordAttrTypes()),
			"aaaa_record": branchObject(t, map[string]attr.Value{
				"name":   types.StringValue(name),
				"values": stringListValue(t, value),
			}),
			"alias_record": types.ObjectNull(aliasRecordAttrTypes()),
			"mx_record":    types.ObjectNull(mxRecordAttrTypes()),
		}),
	})
}

// aliasRecordRRSet creates an ALIAS record RRSet object.
func aliasRecordRRSet(t *testing.T, value string) types.Object {
	t.Helper()
	return rrsetObject(t, map[string]attr.Value{
		"description": types.StringValue("rrset"),
		"ttl":         types.Int64Value(3600),
		"type_record_set_choice": typeRecordSetChoiceObject(t, map[string]attr.Value{
			"a_record":     types.ObjectNull(aRecordAttrTypes()),
			"aaaa_record":  types.ObjectNull(aaaaRecordAttrTypes()),
			"alias_record": branchObject(t, map[string]attr.Value{"value": types.StringValue(value)}),
			"mx_record":    types.ObjectNull(mxRecordAttrTypes()),
		}),
	})
}

// rrsetObject creates an RRSet object value for tests.
func rrsetObject(t *testing.T, attrs map[string]attr.Value) types.Object {
	t.Helper()
	obj, diags := types.ObjectValue(rrsetAttrTypes(), attrs)
	require.False(t, diags.HasError(), "failed to create rrset object: %v", diags)
	return obj
}

// typeRecordSetChoiceObject creates a minimal type_record_set_choice object for tests.
func typeRecordSetChoiceObject(t *testing.T, attrs map[string]attr.Value) types.Object {
	t.Helper()
	obj, diags := types.ObjectValue(typeRecordSetChoiceAttrTypes(), attrs)
	require.False(t, diags.HasError(), "failed to create type_record_set_choice object: %v", diags)
	return obj
}

// branchObject creates a generic object value for a record-set branch.
func branchObject(t *testing.T, attrs map[string]attr.Value) types.Object {
	t.Helper()
	attrTypes := make(map[string]attr.Type, len(attrs))
	for name, value := range attrs {
		attrTypes[name] = value.Type(context.Background())
	}
	obj, diags := types.ObjectValue(attrTypes, attrs)
	require.False(t, diags.HasError(), "failed to create branch object: %v", diags)
	return obj
}

// stringListValue creates a string list value for tests.
func stringListValue(t *testing.T, values ...string) types.List {
	t.Helper()

	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.StringValue(value))
	}

	list, diags := types.ListValue(types.StringType, elements)
	require.False(t, diags.HasError(), "failed to create string list: %v", diags)
	return list
}

// rrsetAttrTypes returns the minimal RRSet object shape used in tests.
func rrsetAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"description":            types.StringType,
		"ttl":                    types.Int64Type,
		"type_record_set_choice": types.ObjectType{AttrTypes: typeRecordSetChoiceAttrTypes()},
	}
}

// typeRecordSetChoiceAttrTypes returns the minimal branch map used in tests.
func typeRecordSetChoiceAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"a_record":     types.ObjectType{AttrTypes: aRecordAttrTypes()},
		"aaaa_record":  types.ObjectType{AttrTypes: aaaaRecordAttrTypes()},
		"alias_record": types.ObjectType{AttrTypes: aliasRecordAttrTypes()},
		"mx_record":    types.ObjectType{AttrTypes: mxRecordAttrTypes()},
	}
}

// aRecordAttrTypes returns the A record branch object shape.
func aRecordAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":   types.StringType,
		"values": types.ListType{ElemType: types.StringType},
	}
}

// aaaaRecordAttrTypes returns the AAAA record branch object shape.
func aaaaRecordAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":   types.StringType,
		"values": types.ListType{ElemType: types.StringType},
	}
}

// aliasRecordAttrTypes returns the ALIAS record branch object shape.
func aliasRecordAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"value": types.StringType,
	}
}

// mxRecordAttrTypes returns an additional name-bearing branch shape for key tests.
func mxRecordAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name": types.StringType,
	}
}
