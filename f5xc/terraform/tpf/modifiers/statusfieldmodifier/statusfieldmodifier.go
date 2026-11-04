//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

package statusfieldmodifier

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ planmodifier.String = statusFieldStringModifier{}
	_ planmodifier.List   = statusFieldListModifier{}
	_ planmodifier.Object = statusFieldObjectModifier{}
)

// statusFieldStringModifier implements planmodifier.String for computed status fields.
type statusFieldStringModifier struct {
	dependencyPath path.Path
}

// statusFieldListModifier implements planmodifier.List for computed list status fields.
type statusFieldListModifier struct {
	dependencyPath path.Path
}

// statusFieldObjectModifier implements planmodifier.Object for computed object status fields.
type statusFieldObjectModifier struct {
	dependencyPath path.Path
}

// UseStateUnlessDependencyChangesString returns a new planmodifier.String that manages
// computed string status fields whose population depends on another configuration attribute.
//
// Why: Standard UseStateForUnknown causes "Provider produced inconsistent result after apply"
// when an asynchronous status field transitions from null to a populated value or when a
// resource is de-advertised. This modifier preserves state when the dependency is unchanged,
// but leaves the plan value as Unknown when the prior state is null or when the dependency changes.
//
// How: It inspects req.StateValue, req.State, and req.Plan. If prior state is null/unknown or
// the dependency attribute at dependencyPath has changed between state and plan, the plan value
// remains Unknown. Otherwise, req.StateValue is copied into resp.PlanValue.
//
// Parameters:
//   - dependencyPath: path.Path to the controlling configuration attribute (e.g. path.Root("advertise_choice"))
//
// Returns:
//   - planmodifier.String: configured modifier instance
//
// Example:
//
//	statusfieldmodifier.UseStateUnlessDependencyChangesString(path.Root("advertise_choice"))
func UseStateUnlessDependencyChangesString(dependencyPath path.Path) planmodifier.String {
	return statusFieldStringModifier{
		dependencyPath: dependencyPath,
	}
}

// UseStateUnlessDependencyChangesList returns a new planmodifier.List that manages
// computed list status fields whose population depends on another configuration attribute.
//
// Why: Provides the same three lifecycle use cases as UseStateUnlessDependencyChangesString
// for List attributes (e.g. dns_info on load balancers).
//
// How: Checks nullability of req.StateValue and evaluates dependency changes via isDependencyChanged.
//
// Parameters:
//   - dependencyPath: path.Path to the controlling configuration attribute (e.g. path.Root("advertise_choice"))
//
// Returns:
//   - planmodifier.List: configured modifier instance
//
// Example:
//
//	statusfieldmodifier.UseStateUnlessDependencyChangesList(path.Root("advertise_choice"))
func UseStateUnlessDependencyChangesList(dependencyPath path.Path) planmodifier.List {
	return statusFieldListModifier{
		dependencyPath: dependencyPath,
	}
}

// UseStateUnlessDependencyChangesObject returns a new planmodifier.Object that manages computed
// object status fields whose population depends on another configuration attribute.
//
// Why: Standard UseStateForUnknown can produce inconsistent-result errors when an asynchronous
// status object transitions or populates child attributes during apply. This modifier preserves
// state when the controlling dependency is unchanged, but leaves the plan unknown when the prior
// state is null or the dependency changes.
//
// How: It checks whether the resource is being created and whether the dependency at
// dependencyPath changed between state and plan. If neither applies, req.StateValue is copied
// into resp.PlanValue.
//
// Parameters:
//   - dependencyPath: path.Path to the controlling configuration attribute (for example,
//     path.Root("loadbalancer_type_choice"))
//
// Returns:
//   - planmodifier.Object: configured modifier instance
//
// Example:
//
//	statusfieldmodifier.UseStateUnlessDependencyChangesObject(path.Root("loadbalancer_type_choice"))
func UseStateUnlessDependencyChangesObject(dependencyPath path.Path) planmodifier.Object {
	return statusFieldObjectModifier{
		dependencyPath: dependencyPath,
	}
}

// Description returns a human-readable description of the modifier.
func (m statusFieldStringModifier) Description(_ context.Context) string {
	return fmt.Sprintf("Preserves the prior string state value unless the dependency attribute at %s "+
		"has changed between state and plan.", m.dependencyPath)
}

// MarkdownDescription returns a Markdown description of the modifier.
func (m statusFieldStringModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

// PlanModifyString implements the string plan modifier logic.
//
// Lifecycle use cases handled:
//  1. Unchanged dependency: resp.PlanValue = req.StateValue (preserves state, including null, avoiding plan drift).
//  2. Dependency changed: resp.PlanValue = Unknown (allows backend to allocate new value or clear to null).
func (m statusFieldStringModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Do nothing if resource is being created.
	if req.State.Raw.IsNull() {
		return
	}

	// Do nothing if there is already a known planned value.
	if !req.PlanValue.IsUnknown() {
		return
	}

	// Do nothing if configuration value is unknown (interpolation guard).
	if req.ConfigValue.IsUnknown() {
		return
	}

	// Use Case 2: Dependency changed.
	// If the controlling attribute changed, keep plan as Unknown so backend can allocate or clear.
	if isDependencyChanged(ctx, req.State, req.Plan, m.dependencyPath) {
		if !resp.PlanValue.IsUnknown() {
			resp.PlanValue = types.StringUnknown()
		}
		return
	}

	// Use Case 1: Unchanged dependency -> preserve prior state value (even if null).
	resp.PlanValue = req.StateValue
}

// Description returns a human-readable description of the modifier.
func (m statusFieldListModifier) Description(_ context.Context) string {
	return fmt.Sprintf("Preserves the prior list state value unless the dependency attribute at %s "+
		"has changed between state and plan.", m.dependencyPath)
}

// MarkdownDescription returns a Markdown description of the modifier.
func (m statusFieldListModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

// PlanModifyList implements the list plan modifier logic.
//
// Lifecycle use cases handled:
//  1. Unchanged dependency: resp.PlanValue = req.StateValue (preserves state, including null, avoiding plan drift).
//  2. Dependency changed: resp.PlanValue = Unknown (allows backend to allocate new items or clear to null).
func (m statusFieldListModifier) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	// Do nothing if resource is being created.
	if req.State.Raw.IsNull() {
		return
	}

	// Do nothing if there is already a known planned value.
	if !req.PlanValue.IsUnknown() {
		return
	}

	// Do nothing if configuration value is unknown (interpolation guard).
	if req.ConfigValue.IsUnknown() {
		return
	}

	// Use Case 2: Dependency changed.
	// If the controlling attribute changed, keep plan as Unknown.
	if isDependencyChanged(ctx, req.State, req.Plan, m.dependencyPath) {
		if !resp.PlanValue.IsUnknown() {
			elemType := req.PlanValue.ElementType(ctx)
			if elemType == nil && !req.StateValue.IsNull() && !req.StateValue.IsUnknown() {
				elemType = req.StateValue.ElementType(ctx)
			}
			if elemType != nil {
				resp.PlanValue = types.ListUnknown(elemType)
			}
		}
		return
	}

	// Use Case 1: Unchanged dependency -> preserve prior state value (even if null).
	resp.PlanValue = req.StateValue
}

// Description returns a human-readable description of the object modifier.
func (m statusFieldObjectModifier) Description(_ context.Context) string {
	return fmt.Sprintf("Preserves the prior object state value unless the dependency attribute at %s "+
		"has changed between state and plan.", m.dependencyPath)
}

// MarkdownDescription returns a Markdown description of the object modifier.
func (m statusFieldObjectModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

// PlanModifyObject preserves the prior object unless its dependency changes.
func (m statusFieldObjectModifier) PlanModifyObject(ctx context.Context, req planmodifier.ObjectRequest, resp *planmodifier.ObjectResponse) {
	if req.State.Raw.IsNull() || !req.PlanValue.IsUnknown() || req.ConfigValue.IsUnknown() {
		return
	}

	if isDependencyChanged(ctx, req.State, req.Plan, m.dependencyPath) {
		return
	}

	resp.PlanValue = req.StateValue
}

// isDependencyChanged determines whether the attribute at depPath has changed between state and plan.
//
// Why: Status fields like host_name or dns_info depend on the configured advertise choice.
// When that choice changes, the backend will allocate or clear the status values.
//
// How: It queries the attribute value at depPath from state and plan. If depPath is empty,
// it returns false (no dependency configured). If either value is unknown or if the values
// differ, it returns true.
//
// Parameters:
//   - ctx: execution context
//   - state: prior resource state
//   - plan: planned resource state
//   - depPath: path to the controlling attribute
//
// Returns:
//   - bool: true if dependency has changed or is unknown; false if identical
func isDependencyChanged(ctx context.Context, state tfsdk.State, plan tfsdk.Plan, depPath path.Path) bool {
	if depPath.Equal(path.Empty()) {
		return false
	}

	var stateVal attr.Value
	var planVal attr.Value

	stateDiags := state.GetAttribute(ctx, depPath, &stateVal)
	planDiags := plan.GetAttribute(ctx, depPath, &planVal)

	if stateDiags.HasError() || planDiags.HasError() {
		tflog.Debug(ctx, "statusfieldmodifier: failed to read dependency path, treating as changed", map[string]interface{}{
			"path": depPath.String(),
		})
		return true
	}

	// If plan dependency is unknown, assume it is changing.
	if planVal != nil && planVal.IsUnknown() {
		return true
	}

	// If state dependency was unknown, treat as changed.
	if stateVal != nil && stateVal.IsUnknown() {
		return true
	}

	// Check nullability differences.
	stateIsNull := stateVal == nil || stateVal.IsNull()
	planIsNull := planVal == nil || planVal.IsNull()

	if stateIsNull && planIsNull {
		return false
	}
	if stateIsNull != planIsNull {
		return true
	}

	// Compare values.
	changed := !planVal.Equal(stateVal)
	tflog.Debug(ctx, "statusfieldmodifier: evaluated dependency", map[string]interface{}{
		"path":    depPath.String(),
		"changed": changed,
	})
	return changed
}
