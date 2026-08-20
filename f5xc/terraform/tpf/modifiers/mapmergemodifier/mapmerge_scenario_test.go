package mapmergemodifier_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/modifiers/mapmergemodifier"
)

// TestMapMerge_UpdateScenario simulates the exact failing scenario:
// Step 1: Create with labels = {"env":"acceptance-test", "version":"v1"}
// After Read, state = {"env":"acceptance-test", "version":"v1", "ves.io/cpd":"RE"}
// Step 3: Update config to labels = {"env":"acceptance-test", "version":"v2"}
// ProposedNewState (PlanValue) = {"env":"acceptance-test", "version":"v2", "ves.io/cpd":"RE"}
// Expected: plan = merge(state, config) = {"env":"acceptance-test", "version":"v2", "ves.io/cpd":"RE"}
//
// The plan must equal the merge result so that the Update handler's restore logic
// (which also merges state + config) produces the same value → plan == apply result.
func TestMapMerge_UpdateScenario(t *testing.T) {
	ctx := context.Background()

	// Config: user changed version from v1 to v2
	configValue := types.MapValueMust(types.StringType, map[string]attr.Value{
		"env":     types.StringValue("acceptance-test"),
		"version": types.StringValue("v2"),
	})

	// State (prior): after Read, includes backend-injected ves.io/cpd
	stateValue := types.MapValueMust(types.StringType, map[string]attr.Value{
		"env":                                   types.StringValue("acceptance-test"),
		"version":                               types.StringValue("v1"),
		"ves.io/config-propagation-destination": types.StringValue("RE"),
	})

	// PlanValue (ProposedNewState): Terraform Core merges config + state computed keys
	planValue := types.MapValueMust(types.StringType, map[string]attr.Value{
		"env":                                   types.StringValue("acceptance-test"),
		"version":                               types.StringValue("v2"),
		"ves.io/config-propagation-destination": types.StringValue("RE"),
	})

	// Expected: merge(state, config) — config wins on conflicts, state keys preserved.
	expectedMerged := types.MapValueMust(types.StringType, map[string]attr.Value{
		"env":                                   types.StringValue("acceptance-test"),
		"version":                               types.StringValue("v2"),
		"ves.io/config-propagation-destination": types.StringValue("RE"),
	})

	req := planmodifier.MapRequest{
		ConfigValue: configValue,
		StateValue:  stateValue,
		PlanValue:   planValue,
	}

	resp := &planmodifier.MapResponse{
		PlanValue: req.PlanValue,
	}

	modifier := mapmergemodifier.New()
	modifier.(mapmergemodifier.MapMergeModifier).PlanModifyMap(ctx, req, resp)

	t.Logf("Config:  %v", configValue)
	t.Logf("State:   %v", stateValue)
	t.Logf("Plan in: %v", planValue)
	t.Logf("Plan out: %v", resp.PlanValue)

	// The plan should equal merge(state, config) — user's v2 + backend ves.io/cpd
	if !resp.PlanValue.Equal(expectedMerged) {
		t.Errorf("Expected plan = merge(state, config) %v, got %v", expectedMerged, resp.PlanValue)
	}

	// The plan should NOT equal config alone (which would miss ves.io/cpd)
	if resp.PlanValue.Equal(configValue) {
		t.Error("Plan should not equal config (missing backend keys)")
	}

	// The plan should NOT equal prior state (which has stale version=v1)
	if resp.PlanValue.Equal(stateValue) {
		t.Error("Plan should not equal prior state (has stale user values)")
	}
}
