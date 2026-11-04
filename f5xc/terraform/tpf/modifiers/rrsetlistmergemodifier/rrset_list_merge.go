//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

package rrsetlistmergemodifier

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ planmodifier.List = rrsetListMergeModifier{}

// rrsetListMergeModifier stabilizes the plan for Optional+Computed RRSet lists
// without hiding backend-side drift.
type rrsetListMergeModifier struct{}

// New returns a new RRSet list merge plan modifier.
func New() planmodifier.List {
	return rrsetListMergeModifier{}
}

// Description returns a human-readable description of the modifier.
func (m rrsetListMergeModifier) Description(_ context.Context) string {
	return "Keeps the plan stable for Optional+Computed RRSet lists. Backend-injected default RRsets " +
		"are excluded during read/apply by the merge helpers, so this modifier does not pin prior " +
		"state and lets backend-side additions, deletions, and value changes surface as drift."
}

// MarkdownDescription returns a Markdown description of the modifier.
func (m rrsetListMergeModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

// PlanModifyList performs only the null/unknown bookkeeping an Optional+Computed
// RRSet list requires.
//
// Why: backend-injected default RRsets are excluded from state by the Read/apply
// merge helpers (utils.MergeRRSetListForRead / MergeRRSetListForApply), so this
// modifier no longer needs to pin prior state to hide them. Pinning prior state
// would mask genuine drift (backend additions/deletions), so it is deliberately
// avoided here.
//
// How:
//   - config unknown: leave the framework plan untouched.
//   - config null + state null (or non-computed `rr_set_group` list): plan null.
//   - config null + state present (for Optional+Computed `default_rr_set_group`):
//     plan unknown, forcing a re-read so a removed list (and any backend defaults)
//     reconcile through the merge helpers.
//   - config present: keep the framework-computed plan (falling back to config),
//     so backend-side additions/deletions/edits surface as normal drift.
func (m rrsetListMergeModifier) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	tflog.Debug(ctx, "RRSetListMergeModifier: ENTER",
		map[string]interface{}{
			"config_null":    req.ConfigValue.IsNull(),
			"config_unknown": req.ConfigValue.IsUnknown(),
			"state_null":     req.StateValue.IsNull(),
			"state_unknown":  req.StateValue.IsUnknown(),
			"plan_null":      req.PlanValue.IsNull(),
			"plan_unknown":   req.PlanValue.IsUnknown(),
		})

	if req.ConfigValue.IsUnknown() {
		return
	}

	elementType := determineElementType(ctx, req)

	if req.ConfigValue.IsNull() {
		if req.StateValue.IsNull() || isRRSetGroupElementType(elementType) {
			resp.PlanValue = types.ListNull(elementType)
			return
		}

		resp.PlanValue = types.ListUnknown(elementType)
		return
	}

	resp.PlanValue = plannedValueOrConfig(req)
}

// plannedValueOrConfig returns the framework-computed plan value when available.
//
// Why: Nested Optional+Computed RRSet fields such as `ttl` may receive defaults
// or child-level plan modifications before this list modifier runs.
//
// How: Prefer the incoming `req.PlanValue` unless it is still null or unknown,
// then fall back to the raw config value.
//
// Parameters:
//   - req: the list plan-modifier request containing config and framework plan values.
//
// Returns:
//   - types.List: the best available known planned value for the RRSet list.
//
// Example:
//
//	planned := plannedValueOrConfig(req)
//	resp.PlanValue = planned
func plannedValueOrConfig(req planmodifier.ListRequest) types.List {
	if !req.PlanValue.IsNull() && !req.PlanValue.IsUnknown() {
		return req.PlanValue
	}

	return req.ConfigValue
}

// determineElementType returns the most specific available list element type.
func determineElementType(ctx context.Context, req planmodifier.ListRequest) attr.Type {
	if elemType := req.ConfigValue.ElementType(ctx); elemType != nil {
		return elemType
	}
	if elemType := req.StateValue.ElementType(ctx); elemType != nil {
		return elemType
	}
	return req.PlanValue.ElementType(ctx)
}

// isRRSetGroupElementType checks whether the given list element type represents a named
// RRSetGroup object (`rr_set_group`).
//
// Why: In `f5xc_dns_zone`, `rr_set_group` is defined with `Optional: true, Computed: false`,
// whereas `default_rr_set_group` is `Optional: true, Computed: true`. When a user removes
// `rr_set_group` from HCL (`req.ConfigValue.IsNull()`), setting `resp.PlanValue` to
// `types.ListUnknown` causes Terraform to fail with an invalid unknown plan value on a
// non-computed attribute. Detecting `rr_set_group` allows planning `types.ListNull` instead.
//
// How: It inspects whether `elemType` is a `types.ObjectType` that defines both `"metadata"`
// and `"rr_set"` attributes.
//
// Parameters:
//   - elemType: the list element type extracted from the plan modifier request.
//
// Returns:
//   - bool: `true` if `elemType` represents an `rr_set_group` object element, `false` otherwise.
//
// Example:
//
//	if isRRSetGroupElementType(elementType) {
//	    resp.PlanValue = types.ListNull(elementType)
//	}
func isRRSetGroupElementType(elemType attr.Type) bool {
	if elemType == nil {
		return false
	}
	objType, ok := elemType.(types.ObjectType)
	if !ok {
		return false
	}
	_, hasMeta := objType.AttrTypes["metadata"]
	_, hasRRSet := objType.AttrTypes["rr_set"]
	return hasMeta && hasRRSet
}
