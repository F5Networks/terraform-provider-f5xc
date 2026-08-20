//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

// Package oneofbranchplanmodifier provides a Terraform object plan modifier for
// oneof-style nested objects whose backend representation may canonicalize from
// one branch to another semantically equivalent branch.
package oneofbranchplanmodifier

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ planmodifier.Object = preserveSelectedBranchModifier{}

// preserveSelectedBranchModifier keeps only the configured oneof branch in the
// planned object so Terraform does not show drift when the API stores the same
// intent under a different sibling branch.
type preserveSelectedBranchModifier struct{}

// PreserveSelectedBranch returns a plan modifier that keeps the configured oneof
// branch selected in the planned object and clears all sibling branches to typed
// null values.
func PreserveSelectedBranch() planmodifier.Object {
	return preserveSelectedBranchModifier{}
}

// Description returns a human-readable description of the modifier.
func (m preserveSelectedBranchModifier) Description(_ context.Context) string {
	return "Preserves the configured oneof branch in the planned object and clears canonicalized sibling branches " +
		"to avoid false diffs when the API represents the same intent with a different branch."
}

// MarkdownDescription returns a Markdown description of the modifier.
func (m preserveSelectedBranchModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

// PlanModifyObject rewrites the planned object to keep only the configured oneof
// branch. The selected branch value comes from the plan when available so any
// legitimate plan-time data within that branch is preserved.
func (m preserveSelectedBranchModifier) PlanModifyObject(ctx context.Context, req planmodifier.ObjectRequest, resp *planmodifier.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	selectedBranch, ok := selectedKnownBranch(req.ConfigValue)
	if !ok {
		return
	}

	attributeTypes := req.ConfigValue.AttributeTypes(ctx)
	if len(attributeTypes) == 0 {
		attributeTypes = req.PlanValue.AttributeTypes(ctx)
	}
	if len(attributeTypes) == 0 {
		return
	}

	planAttrs := req.PlanValue.Attributes()
	configAttrs := req.ConfigValue.Attributes()

	normalizedAttrs := make(map[string]attr.Value, len(attributeTypes))
	for attrName, attrType := range attributeTypes {
		normalizedAttrs[attrName] = attrType.ValueType(ctx)
	}

	branchValue, exists := planAttrs[selectedBranch]
	if !exists || branchValue.IsNull() || branchValue.IsUnknown() {
		branchValue = configAttrs[selectedBranch]
	}
	normalizedAttrs[selectedBranch] = branchValue

	normalizedPlan, diags := types.ObjectValue(attributeTypes, normalizedAttrs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.PlanValue = normalizedPlan
}

// selectedKnownBranch returns the single known branch selected in a oneof object.
// If none or multiple branches are selected, it returns false so the modifier does
// not interfere with Terraform's normal validation path.
func selectedKnownBranch(obj types.Object) (string, bool) {
	selectedBranch := ""
	for branchName, branchValue := range obj.Attributes() {
		if branchValue.IsNull() || branchValue.IsUnknown() {
			continue
		}
		if selectedBranch != "" {
			return "", false
		}
		selectedBranch = branchName
	}

	return selectedBranch, selectedBranch != ""
}
