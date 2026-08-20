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
//     {keyA:valueA, ves.io/provider:azure} → specToModel stores only config value →
//     state={keyA:valueA} ✓ (backend keys picked up on next Read)
//
//     Update (state already contains merged user+backend labels):
//     Config={keyA:valueB} (user changed a value), State={keyA:valueA, ves.io/provider:azure}
//     Plan = prior state = {keyA:valueA, ves.io/provider:azure}.
//     Update handler sends {keyA:valueB} to API (reads Config for user-intent labels).
//     API returns {keyA:valueB, ves.io/provider:azure}.
//     specToModel overwrites data.Labels with API response, but template restores
//     data.Labels = planLabels (= prior state) before State.Set → state={keyA:valueA, ves.io/provider:azure} ✓
//     apply_result == plan == prior state → Terraform Core happy ✓
//     Next Read: stores full API response → state={keyA:valueB, ves.io/provider:azure} ✓
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
// | Config  | State        | Plan result              | Behavior                                                       |
// |---------|--------------|--------------------------|----------------------------------------------------------------|
// | Unknown | any          | unchanged                | Interpolation guard — do not touch plan                        |
// | null    | null         | null                     | Re-apply, no labels at all — keep plan null, no diff           |
// | null    | null (first) | Unknown                  | First create, no labels — defer to allow backend injection     |
// | null    | known        | Unknown                  | Update, user removed all labels                                |
// | known   | null/unknown | config                   | First create; Terraform Core requires plan=config              |
// | known   | known        | prior state              | Update: plan=state; Update handler reads Config for API req    |
// | Config  | State        | Plan result  | Behavior                                                                   |

package mapmergemodifier

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
		"(e.g. 'ves.io/config-propagation-destination'). When config is null, marks the plan as Unknown to " +
		"allow backend-injected keys to be stored in state. When config is known and state is known " +
		"(update), plan=prior state so Terraform Core's constraint is satisfied and backend-injected keys " +
		"are preserved. The Update handler reads user-intent labels from Config (not Plan) to send to the " +
		"API, and restores the plan value before storing state so apply_result == plan."
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
//	config=null,  state=null    → plan=null     (re-apply with no labels — no diff)
//	config=null,  state=unknown → plan=Unknown  (first create, no labels — allow backend injection)
//	config=null,  state=known   → plan=Unknown  (update, user removed all labels)
//	config=known, state=null    → plan=config   (first create with user labels)
//	config=known, state=known   → plan=config ∪ ves.io/* backend keys from prior state
//	                             (update: reflects user changes + preserves backend-injected keys)
//	config=known, state=known   → plan=prior state
//	                             (update: Terraform Core requires plan = config or prior state;
//	                              plan=prior state always satisfies this and preserves backend-injected keys.
//	                              Update handler reads Config for API request.
//	                              Template restores plan value before State.Set. After apply,
//	                              Read captures full API response in state.)
func (m MapMergeModifier) PlanModifyMap(ctx context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	tflog.Debug(ctx, "MapMergeModifier: ENTER",
		map[string]interface{}{
			"config_null":    req.ConfigValue.IsNull(),
			"config_unknown": req.ConfigValue.IsUnknown(),
			"config_value":   fmt.Sprintf("%v", req.ConfigValue),
			"state_null":     req.StateValue.IsNull(),
			"state_unknown":  req.StateValue.IsUnknown(),
			"state_value":    fmt.Sprintf("%v", req.StateValue),
			"plan_null":      req.PlanValue.IsNull(),
			"plan_unknown":   req.PlanValue.IsUnknown(),
			"plan_value":     fmt.Sprintf("%v", req.PlanValue),
		})
	// Guard: if the config value is Unknown (e.g. an interpolation that has not been
	// resolved yet), do not modify the plan. Changing the plan during interpolation
	// can cause incorrect diffs. This mirrors the behavior of UseStateForUnknown.
	if req.ConfigValue.IsUnknown() {
		return
	}

	// Case: config is null (user has not configured any labels).
	if req.ConfigValue.IsNull() {
		// Subcase: state is also null — neither the user nor the backend has set any
		// labels. This happens on a re-apply after a previous apply where the API
		// returned no labels. Keep the plan as null so Terraform sees no diff and
		// does not show "+ labels = (known after apply)".
		if req.StateValue.IsNull() {
			resp.PlanValue = types.MapNull(types.StringType)
			return
		}

		// Subcase: state is unknown (first create) or state is known (update where
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
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		resp.PlanValue = req.ConfigValue
		return
	}

	// Case: config is known, state is known (update with user labels).
	//
	// Terraform Core enforces: plan MUST equal config OR prior state. No hybrid.
	// We use a conditional strategy:
	//   - If user-intent keys (non-ves.io/) in config match prior state exactly
	//     → plan = prior state (preserves backend-injected keys, no diff)
	//   - If user changed/added/removed any key
	//     → plan = config (satisfies TF Core constraint)
	//
	// When plan = config and user changed labels, the plan output may show
	// backend-injected keys as "will be removed". This is cosmetically imperfect
	// but harmless — after Apply+Read, the keys reappear in state.

	stateElems := req.StateValue.Elements()
	configElems := req.ConfigValue.Elements()

	// Check if user-intent keys in config match prior state exactly.
	// We ignore backend-injected keys (ves.io/ prefix) in prior state.
	configMatchesPrior := true

	// 1. Every non-backend key in prior state must exist in config with same value.
	for k, v := range stateElems {
		if strings.HasPrefix(k, "ves.io/") {
			continue // skip backend-injected keys
		}
		cv, exists := configElems[k]
		if !exists || !cv.Equal(v) {
			configMatchesPrior = false
			break
		}
	}

	// 2. Config must have no extra keys beyond prior's user keys.
	if configMatchesPrior {
		for k := range configElems {
			if _, exists := stateElems[k]; !exists {
				configMatchesPrior = false
				break
			}
		}
	}

	if configMatchesPrior {
		// No user change → keep prior state (includes backend-injected keys).
		// plan == prior state → TF Core accepts.
		tflog.Debug(ctx, "MapMergeModifier: user keys unchanged, plan = prior state")
		resp.PlanValue = req.StateValue
		return
	}

	// User changed labels → plan must equal config exactly.
	// Backend-injected keys will be re-added by Read handler after Apply.
	tflog.Debug(ctx, "MapMergeModifier: user keys changed, plan = config")
	resp.PlanValue = req.ConfigValue
}
