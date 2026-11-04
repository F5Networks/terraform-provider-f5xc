//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package mapmergemodifier_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/modifiers/mapmergemodifier"
)

// TestMapMerge_UpdateScenario verifies that when a user updates a label value,
// PlanModifyMap sets PlanValue equal to ConfigValue (never a hybrid of config and state)
// to satisfy Terraform Core's Optional+Computed plan constraint.
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

	// When user keys change, plan must equal config (not a hybrid of config + state).
	assert.Equal(t, configValue, resp.PlanValue,
		"expected plan to equal config when user modifies label values")
	assert.NotEqual(t, stateValue, resp.PlanValue,
		"plan must not equal stale prior state")
}
