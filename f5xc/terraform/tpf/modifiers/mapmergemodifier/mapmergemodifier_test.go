// Copyright (c) 2026 F5 Inc. All rights reserved.

package mapmergemodifier

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

// TestMapMergeModifier_Description verifies the Description method returns a non-empty string.
func TestMapMergeModifier_Description(t *testing.T) {
	m := MapMergeModifier{}
	desc := m.Description(context.Background())
	assert.NotEmpty(t, desc)
}

// TestMapMergeModifier_MarkdownDescription verifies MarkdownDescription delegates to Description.
func TestMapMergeModifier_MarkdownDescription(t *testing.T) {
	m := MapMergeModifier{}
	assert.Equal(t, m.Description(context.Background()), m.MarkdownDescription(context.Background()))
}

// TestNew verifies the constructor returns a planmodifier.Map implementation.
func TestNew(t *testing.T) {
	m := New()
	assert.Implements(t, (*planmodifier.Map)(nil), m)
}

// Scenario 1 (re-apply): config=null, state=null → plan=null (no diff).
// This is the bug-fix case: after a previous apply where the API returned no labels,
// state is null. On re-apply the plan must also be null so Terraform shows no diff
// and does not display "+ labels = (known after apply)".
func TestPlanModifyMap_NullConfig_NullState_ReApply(t *testing.T) {
	m := MapMergeModifier{}

	req := planmodifier.MapRequest{
		ConfigValue: types.MapNull(types.StringType),
		StateValue:  types.MapNull(types.StringType),
		PlanValue:   types.MapNull(types.StringType),
	}
	resp := &planmodifier.MapResponse{PlanValue: req.PlanValue}

	m.PlanModifyMap(context.Background(), req, resp)

	assert.True(t, resp.PlanValue.IsNull(),
		"expected plan=null when config=null and state=null (re-apply, no labels at all — no diff)")
	assert.False(t, resp.PlanValue.IsUnknown(),
		"plan must NOT be Unknown on re-apply with no labels — that would cause a spurious diff")
}

// Scenario 1 (first create): config=null, state=unknown → plan=Unknown.
// On the very first create, the framework sets state to Unknown. We must mark the
// plan as Unknown so Terraform defers the consistency check and allows the backend
// to inject labels into state after apply.
func TestPlanModifyMap_NullConfig_UnknownState_FirstCreate(t *testing.T) {
	m := MapMergeModifier{}

	req := planmodifier.MapRequest{
		ConfigValue: types.MapNull(types.StringType),
		StateValue:  types.MapUnknown(types.StringType),
		PlanValue:   types.MapNull(types.StringType),
	}
	resp := &planmodifier.MapResponse{PlanValue: req.PlanValue}

	m.PlanModifyMap(context.Background(), req, resp)

	assert.True(t, resp.PlanValue.IsUnknown(),
		"expected Unknown when config=null, state=unknown (first create, no labels — allow backend injection)")
}

// Scenario 3 update: config=null, state=known → plan=Unknown
// Covers: user removes labels on update; backend-injected labels were in state.
func TestPlanModifyMap_NullConfig_KnownState(t *testing.T) {
	m := MapMergeModifier{}

	stateMap, diags := types.MapValue(types.StringType, map[string]attr.Value{
		"ves.io/provider": types.StringValue("azure"),
	})
	if diags.HasError() {
		t.Fatalf("failed to create state map: %v", diags)
	}

	req := planmodifier.MapRequest{
		ConfigValue: types.MapNull(types.StringType),
		StateValue:  stateMap,
		PlanValue:   types.MapNull(types.StringType),
	}
	resp := &planmodifier.MapResponse{PlanValue: req.PlanValue}

	m.PlanModifyMap(context.Background(), req, resp)

	assert.True(t, resp.PlanValue.IsUnknown(),
		"expected Unknown when config=null, state=known (update, user removed labels)")
}

// Guard: config=unknown → plan unchanged (interpolation not yet resolved).
// Mirrors UseStateForUnknown behaviour: do not touch the plan during interpolation.
func TestPlanModifyMap_UnknownConfig(t *testing.T) {
	m := MapMergeModifier{}

	configMap, diags := types.MapValue(types.StringType, map[string]attr.Value{
		"keyA": types.StringValue("valueA"),
	})
	if diags.HasError() {
		t.Fatalf("failed to create config map: %v", diags)
	}

	req := planmodifier.MapRequest{
		ConfigValue: types.MapUnknown(types.StringType), // config not yet resolved
		StateValue:  types.MapNull(types.StringType),
		PlanValue:   configMap, // whatever the framework set
	}
	resp := &planmodifier.MapResponse{PlanValue: req.PlanValue}

	m.PlanModifyMap(context.Background(), req, resp)

	// Plan must remain unchanged when config is unknown.
	assert.Equal(t, configMap, resp.PlanValue,
		"expected plan to remain unchanged when config is unknown (interpolation guard)")
}

// Scenario 2: config=known, state=null → plan=config (first create with user labels)
// Backend-injected keys are not yet in state; plan must equal config to satisfy Terraform Core.
func TestPlanModifyMap_KnownConfig_NullState(t *testing.T) {
	m := MapMergeModifier{}

	configMap, diags := types.MapValue(types.StringType, map[string]attr.Value{
		"keyA": types.StringValue("valueA"),
	})
	if diags.HasError() {
		t.Fatalf("failed to create config map: %v", diags)
	}

	req := planmodifier.MapRequest{
		ConfigValue: configMap,
		StateValue:  types.MapNull(types.StringType),
		PlanValue:   configMap,
	}
	resp := &planmodifier.MapResponse{PlanValue: req.PlanValue}

	m.PlanModifyMap(context.Background(), req, resp)

	assert.False(t, resp.PlanValue.IsUnknown(),
		"expected plan to remain known (= config) on first create with user labels")
	assert.Equal(t, configMap, resp.PlanValue,
		"expected plan value to equal config value on first create")
}

// Scenario 4a: config=known, state=known → plan = config ∪ state["ves.io/*"]
// User keeps all their labels; backend-injected keys are preserved.
func TestPlanModifyMap_KnownConfig_KnownState(t *testing.T) {
	m := MapMergeModifier{}

	configMap, diags := types.MapValue(types.StringType, map[string]attr.Value{
		"keyA": types.StringValue("valueA"),
	})
	if diags.HasError() {
		t.Fatalf("failed to create config map: %v", diags)
	}

	// State contains merged user+backend labels from previous apply.
	stateMap, diags := types.MapValue(types.StringType, map[string]attr.Value{
		"keyA":            types.StringValue("valueA"),
		"ves.io/provider": types.StringValue("azure"),
	})
	if diags.HasError() {
		t.Fatalf("failed to create state map: %v", diags)
	}

	// Expected plan = config keys + ves.io/* state keys.
	expectedMap, diags := types.MapValue(types.StringType, map[string]attr.Value{
		"keyA":            types.StringValue("valueA"),
		"ves.io/provider": types.StringValue("azure"),
	})
	if diags.HasError() {
		t.Fatalf("failed to create expected map: %v", diags)
	}

	req := planmodifier.MapRequest{
		ConfigValue: configMap,
		StateValue:  stateMap,
		PlanValue:   configMap, // framework sets plan = config initially
	}
	resp := &planmodifier.MapResponse{PlanValue: req.PlanValue}

	m.PlanModifyMap(context.Background(), req, resp)

	assert.False(t, resp.PlanValue.IsUnknown(),
		"expected plan to remain known on update with user+backend labels")
	assert.Equal(t, expectedMap, resp.PlanValue,
		"expected plan = config keys ∪ ves.io/* state keys")
}

// Scenario 4b: user deletes a key from HCL config → plan equals config to satisfy
// Terraform Core's constraint (plan must equal config or prior state, never a hybrid).
// Backend-injected keys (ves.io/*) are re-populated after the Update + Read cycle.
func TestPlanModifyMap_KnownConfig_KnownState_UserDeletesKey(t *testing.T) {
	m := MapMergeModifier{}

	// User removed keyB from HCL — only keyA remains in config.
	configMap, diags := types.MapValue(types.StringType, map[string]attr.Value{
		"keyA": types.StringValue("valueA"),
	})
	if diags.HasError() {
		t.Fatalf("failed to create config map: %v", diags)
	}

	// Prior state had keyA, keyB (user) and ves.io/provider (backend).
	stateMap, diags := types.MapValue(types.StringType, map[string]attr.Value{
		"keyA":            types.StringValue("valueA"),
		"keyB":            types.StringValue("valueB"),
		"ves.io/provider": types.StringValue("azure"),
	})
	if diags.HasError() {
		t.Fatalf("failed to create state map: %v", diags)
	}

	req := planmodifier.MapRequest{
		ConfigValue: configMap,
		StateValue:  stateMap,
		PlanValue:   configMap,
	}
	resp := &planmodifier.MapResponse{PlanValue: req.PlanValue}

	m.PlanModifyMap(context.Background(), req, resp)

	assert.False(t, resp.PlanValue.IsUnknown(),
		"expected plan to be known after user deletes a key")
	assert.Equal(t, configMap, resp.PlanValue,
		"expected plan to equal config when user deletes a key")
}

// Scenario 4c: user adds a new key to HCL config → plan equals config to satisfy
// Terraform Core's constraint (plan must equal config or prior state, never a hybrid).
func TestPlanModifyMap_KnownConfig_KnownState_UserAddsKey(t *testing.T) {
	m := MapMergeModifier{}

	// User added keyB to HCL.
	configMap, diags := types.MapValue(types.StringType, map[string]attr.Value{
		"keyA": types.StringValue("valueA"),
		"keyB": types.StringValue("valueB"),
	})
	if diags.HasError() {
		t.Fatalf("failed to create config map: %v", diags)
	}

	// Prior state had only keyA (user) and ves.io/provider (backend).
	stateMap, diags := types.MapValue(types.StringType, map[string]attr.Value{
		"keyA":            types.StringValue("valueA"),
		"ves.io/provider": types.StringValue("azure"),
	})
	if diags.HasError() {
		t.Fatalf("failed to create state map: %v", diags)
	}

	req := planmodifier.MapRequest{
		ConfigValue: configMap,
		StateValue:  stateMap,
		PlanValue:   configMap,
	}
	resp := &planmodifier.MapResponse{PlanValue: req.PlanValue}

	m.PlanModifyMap(context.Background(), req, resp)

	assert.False(t, resp.PlanValue.IsUnknown(),
		"expected plan to be known after user adds a key")
	assert.Equal(t, configMap, resp.PlanValue,
		"expected plan to equal config when user adds a key")
}

// Scenario 4d: user changes value of an existing key → new value must appear in plan.
// Backend-injected keys (ves.io/*) must still be preserved.
func TestPlanModifyMap_KnownConfig_KnownState_UserChangesValue(t *testing.T) {
	m := MapMergeModifier{}

	// User changed env from "test" to "stage" and team from "network" to "netops".
	configMap, diags := types.MapValue(types.StringType, map[string]attr.Value{
		"env":  types.StringValue("stage"),
		"team": types.StringValue("netops"),
	})
	if diags.HasError() {
		t.Fatalf("failed to create config map: %v", diags)
	}

	// Prior state has old values; no backend keys.
	stateMap, diags := types.MapValue(types.StringType, map[string]attr.Value{
		"env":  types.StringValue("test"),
		"team": types.StringValue("network"),
	})
	if diags.HasError() {
		t.Fatalf("failed to create state map: %v", diags)
	}

	req := planmodifier.MapRequest{
		ConfigValue: configMap,
		StateValue:  stateMap,
		PlanValue:   configMap,
	}
	resp := &planmodifier.MapResponse{PlanValue: req.PlanValue}

	m.PlanModifyMap(context.Background(), req, resp)

	assert.False(t, resp.PlanValue.IsUnknown(),
		"expected plan to be known after user changes label values")
	assert.Equal(t, configMap, resp.PlanValue,
		"expected plan to reflect new config values when user changes label values")
}

// Scenario 4e: user changes a value AND backend keys are present in state.
// Plan must equal config to satisfy Terraform Core's constraint (no hybrid of config and state).
func TestPlanModifyMap_KnownConfig_KnownState_UserChangesValue_WithBackendKey(t *testing.T) {
	m := MapMergeModifier{}

	// User changed env from "test" to "stage".
	configMap, diags := types.MapValue(types.StringType, map[string]attr.Value{
		"env": types.StringValue("stage"),
	})
	if diags.HasError() {
		t.Fatalf("failed to create config map: %v", diags)
	}

	// Prior state has old user value + backend-injected key.
	stateMap, diags := types.MapValue(types.StringType, map[string]attr.Value{
		"env":             types.StringValue("test"),
		"ves.io/provider": types.StringValue("azure"),
	})
	if diags.HasError() {
		t.Fatalf("failed to create state map: %v", diags)
	}

	req := planmodifier.MapRequest{
		ConfigValue: configMap,
		StateValue:  stateMap,
		PlanValue:   configMap,
	}
	resp := &planmodifier.MapResponse{PlanValue: req.PlanValue}

	m.PlanModifyMap(context.Background(), req, resp)

	assert.False(t, resp.PlanValue.IsUnknown(),
		"expected plan to be known after user changes a value with backend key present")
	assert.Equal(t, configMap, resp.PlanValue,
		"expected plan to equal config when user changes a value")
}

// TestPlanModifyMap_KnownConfigMultipleKeys verifies that a config with multiple
// labels and no prior state is preserved as-is in the plan (first create).
func TestPlanModifyMap_KnownConfigMultipleKeys(t *testing.T) {
	m := MapMergeModifier{}

	configMap, diags := types.MapValue(types.StringType, map[string]attr.Value{
		"keyA": types.StringValue("valueA"),
		"keyB": types.StringValue("valueB"),
	})
	if diags.HasError() {
		t.Fatalf("failed to create config map: %v", diags)
	}

	req := planmodifier.MapRequest{
		ConfigValue: configMap,
		StateValue:  types.MapNull(types.StringType),
		PlanValue:   configMap,
	}
	resp := &planmodifier.MapResponse{PlanValue: req.PlanValue}

	m.PlanModifyMap(context.Background(), req, resp)

	assert.False(t, resp.PlanValue.IsUnknown(),
		"expected plan value to remain known when config has multiple values (first create)")
	assert.Equal(t, configMap, resp.PlanValue)
}

// TestPlanModifyMap_EmptyConfig verifies that an empty (but non-null) config map
// with no prior state is preserved as-is in the plan (not marked Unknown).
func TestPlanModifyMap_EmptyConfig(t *testing.T) {
	m := MapMergeModifier{}

	emptyMap, diags := types.MapValue(types.StringType, map[string]attr.Value{})
	if diags.HasError() {
		t.Fatalf("failed to create empty map: %v", diags)
	}

	req := planmodifier.MapRequest{
		ConfigValue: emptyMap,
		StateValue:  types.MapNull(types.StringType),
		PlanValue:   emptyMap,
	}
	resp := &planmodifier.MapResponse{PlanValue: req.PlanValue}

	m.PlanModifyMap(context.Background(), req, resp)

	assert.False(t, resp.PlanValue.IsUnknown(),
		"expected plan value to remain known when config is an empty (non-null) map")
}
