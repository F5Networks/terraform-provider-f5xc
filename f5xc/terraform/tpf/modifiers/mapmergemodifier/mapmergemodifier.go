//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

// Package mapmergemodifier provides a Terraform plan modifier for map attributes
// where the backend may inject additional key/value pairs that were not present
// in the user's configuration (e.g. "ves.io/provider" labels injected by the F5 XC API).
//
// # Background
//
// The labels attribute is Optional+Computed. The F5 XC API may inject extra
// labels (e.g. "ves.io/provider") regardless of what the user configured.
//
// # The four scenarios
//
//  1. Empty (no user labels, no backend labels):
//     First apply:  Config=null, State=null(unknown) → plan=Unknown → apply: API returns nil → state=null ✓
//     Re-apply:     Config=null, State=null           → plan=null   → no diff shown ✓
//
//  2. User labels only (no backend injection):
//     Config={keyA:valueA} → plan={keyA:valueA} → apply: API returns {keyA:valueA}
//     → state={keyA:valueA} ✓
//
//  3. Backend labels only (user has no labels configured):
//     Config=null → plan=Unknown → apply: API returns {ves.io/provider:azure}
//     → state={ves.io/provider:azure} ✓
//
//  4. User labels + backend labels (the complex case):
//     First create:
//     Config={keyA:valueA} → plan={keyA:valueA} (Terraform Core enforces this for
//     Optional+Computed when config is known) → apply: API returns
//     {keyA:valueA, ves.io/provider:azure} → specToModel merges plan+API →
//     state={keyA:valueA, ves.io/provider:azure} ✓
//     BUT: plan was {keyA:valueA} and result is {keyA:valueA, ves.io/provider:azure}
//     → "Provider produced inconsistent result after apply" ✗
//
//     This is the fundamental Terraform constraint: for Optional+Computed attributes,
//     Terraform Core does NOT allow the plan to be Unknown when config has a known
//     value. Therefore, on first create with user labels, we cannot defer the
//     consistency check — the apply result MUST match the plan exactly.
//
//     Solution for first create (state=null, config=known):
//     Plan = config value. specToModel returns exactly the config value (ignores
//     backend-injected keys). State = config value only. Backend keys are lost.
//
//     Solution for updates (state=known, config=known):
//     Plan = prior state value (which already contains the merged user+backend labels
//     from the previous apply). specToModel returns the merged API response.
//     State = merged user+backend labels. ✓
//     This works because Terraform Core allows the plan to differ from config for
//     Computed attributes — the plan just cannot be Unknown when config is known.
//
// # Label ownership scenarios
//
// | Scenario            | State result                              | Terraform behavior                                                                                                                |
// |---------------------|-------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------- |
// | User labels only    | `{"user": "val"}`                         | Standard behavior.                                                                                                                |
// | Backend labels only | `{"sys": "val"}`                          | Because the attribute is `Computed: true`, the backend value is simply stored in state. No diff, since no user config is present. |
// | User + Backend      | `{"user": "val", "sys": "val"}`           | Critical: without the plan modifier, Terraform would attempt to delete the backend-injected key.                                  |
//
// # Summary of plan modifier logic
//
// | Config  | State        | Plan result                | Behavior                                                       |
// |---------|--------------|----------------------------|----------------------------------------------------------------|
// | Unknown | any          | unchanged                  | Interpolation guard — do not touch plan                        |
// | null    | null         | null                       | Re-apply, no labels at all — keep plan null, no diff           |
// | null    | null (first) | Unknown                    | First create, no labels — defer to allow backend injection     |
// | null    | known        | Unknown                    | Update, user removed all labels                                |
// | known   | null/unknown | config                     | First create; Terraform Core requires plan=config              |
// | known   | known        | config ∪ state["ves.io/*"] | Update: user key deletions respected, backend keys preserved   |
//

package mapmergemodifier

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ planmodifier.Map = MapMergeModifier{} // Ensure MapMergeModifier implements the planmodifier.Map interface

// MapMergeModifier is a plan modifier for Optional+Computed map attributes where
// the backend may inject additional key/value pairs beyond what the user configured.
type MapMergeModifier struct{}

// New returns a new MapMergeModifier.
func New() planmodifier.Map {
	return MapMergeModifier{}
}

// Description returns a human-readable description of the modifier.
func (m MapMergeModifier) Description(_ context.Context) string {
	return "Handles Optional+Computed map attributes where the backend may inject extra key/value pairs " +
		"(e.g. 'ves.io/provider'). When config is null, marks the plan as Unknown to allow backend-injected " +
		"keys to be stored in state. When config is known and prior state is known (update), uses the prior " +
		"state as the plan value so that merged user+backend labels are preserved across updates. " +
		"On first create with user labels, the plan equals the config value and backend-injected keys " +
		"are not stored in state until the next update."
}

// MarkdownDescription returns a Markdown description of the modifier.
func (m MapMergeModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

// PlanModifyMap implements the plan modifier logic for all label scenarios.
//
// Decision table:
//
//	config=unknown              → plan=unchanged (interpolation guard)
//	config=null,  state=null   → plan=null     (re-apply with no labels — no diff)
//	config=null,  state=unknown → plan=Unknown  (first create, no labels — allow backend injection)
//	config=null,  state=known  → plan=Unknown  (update, user removed all labels)
//	config=known, state=null   → plan=config   (first create with user labels)
//	config=known, state=known  → plan=config ∪ state["ves.io/*"]
//	                             (update: user key deletions are respected,
//	                              backend-injected keys are preserved)
func (m MapMergeModifier) PlanModifyMap(ctx context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	// Guard: if the config value is Unknown (e.g. an interpolation that has not been
	// resolved yet), do not modify the plan. Changing the plan during interpolation
	// can cause incorrect diffs. This mirrors the behavior of UseStateForUnknown.
	if req.ConfigValue.IsUnknown() {
		return
	}

	// Case: config is null (user has not configured any labels).
	if req.ConfigValue.IsNull() {
		// Sub-case: state is also null — neither the user nor the backend has set any
		// labels. This happens on a re-apply after a previous apply where the API
		// returned no labels. Keep the plan as null so Terraform sees no diff and
		// does not show "+ labels = (known after apply)".
		if req.StateValue.IsNull() {
			resp.PlanValue = types.MapNull(types.StringType)
			return
		}

		// Sub-case: state is unknown (first create) or state is known (update where
		// the user removed all labels). Mark the plan as Unknown so Terraform defers
		// the element-level consistency check until after apply. This allows
		// backend-injected labels to be stored in state without causing
		// "Provider produced inconsistent result after apply".
		resp.PlanValue = types.MapUnknown(types.StringType)
		return
	}

	// From here on: config is a known, non-null map value.

	// Case: config is known, state is null or unknown (first create with user labels).
	// Terraform Core does not allow Unknown in the plan when config has a known
	// value for Optional+Computed attributes — it would cause "Provider produced
	// invalid plan: planned value does not match config value".
	// The framework may have already set the plan to Unknown (its default for
	// Computed attributes), so we must explicitly set the plan to the config value.
	// The specToModel logic will return exactly the config value so the apply
	// result matches the plan. Backend-injected keys are not stored in state on
	// first create; they will be picked up by the Read handler on the next refresh.
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		resp.PlanValue = req.ConfigValue
		return
	}

	// Case: config is known, state is known (update with user labels).
	//
	// Build the merged plan:
	//   1. Start with all keys from the current config (= current user intent).
	//      This correctly drops any user key that was removed from the HCL.
	//   2. Add all keys from the prior state that have the "ves.io/" prefix.
	//      These are backend-injected labels that must be preserved across updates.
	//
	// Result: user key deletions are visible in the plan, backend keys are retained.
	merged := make(map[string]attr.Value, len(req.ConfigValue.Elements())+len(req.StateValue.Elements()))

	// Step 1: all user-configured keys (current config is authoritative for user keys).
	for k, v := range req.ConfigValue.Elements() {
		merged[k] = v
	}

	// Step 2: backend-injected keys from prior state (identified by "ves.io/" prefix).
	for k, v := range req.StateValue.Elements() {
		if strings.HasPrefix(k, "ves.io/") {
			merged[k] = v
		}
	}

	mergedMap, diags := types.MapValue(types.StringType, merged)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	resp.PlanValue = mergedMap
}
