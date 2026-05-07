//
// Copyright (c) 2025 F5 Inc. All rights reserved.
//

package defaultvaluemodifier

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ planmodifier.Int32 = DefaultValueModifier{}  // Ensure DefaultValueModifier implements the planmodifier.Int32 interface
var _ planmodifier.Int64 = DefaultValueModifier{}  // Ensure DefaultValueModifier implements the planmodifier.Int32 interface
var _ planmodifier.String = DefaultValueModifier{} // Ensure DefaultValueModifier implements the planmodifier.String interface

// DefaultValueModifier struct
type DefaultValueModifier struct {
	// List of sibling attribute names that, if set, suppress the default.
	Suppressors []string
}

// NewDefaultValueModifierInt32 Constructor function
func NewDefaultValueModifierInt32(suppressors ...string) planmodifier.Int32 {
	return DefaultValueModifier{Suppressors: suppressors}
}

// NewDefaultValueModifierInt64 Constructor function
func NewDefaultValueModifierInt64(suppressors ...string) planmodifier.Int64 {
	return DefaultValueModifier{Suppressors: suppressors}
}

// NewDefaultValueModifierString Constructor function
func NewDefaultValueModifierString(suppressors ...string) planmodifier.String {
	return DefaultValueModifier{Suppressors: suppressors}
}

// Description returns a description of the modifier
func (m DefaultValueModifier) Description(_ context.Context) string {
	return "Suppresses the default value if any specified sibling attribute is set."
}

// MarkdownDescription returns a Markdown description of the modifier
func (m DefaultValueModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

// PlanModifyInt32 Core Logic
func (m DefaultValueModifier) PlanModifyInt32(ctx context.Context, req planmodifier.Int32Request, resp *planmodifier.Int32Response) {
	// If the user explicitly set the current attribute, we respect that.
	if !req.ConfigValue.IsNull() {
		return
	}

	// Iterate over all suppressor attribute names
	for _, suppressorName := range m.Suppressors {
		// Get the path to the sibling attribute (same block, different name)
		suppressorPath := req.Path.ParentPath().AtName(suppressorName)

		var suppressorValue types.Bool // Assuming all your suppressors are Bool types

		// Look up the planned value of the suppressor attribute
		req.Plan.GetAttribute(ctx, suppressorPath, &suppressorValue)

		// Check for lookup errors
		if resp.Diagnostics.HasError() {
			return
		}

		// Logic Check: is the suppressor explicitly set (not Null, not Unknown) AND true?
		isSuppressed := !suppressorValue.IsNull() && !suppressorValue.IsUnknown() && suppressorValue.ValueBool()

		if isSuppressed {
			// If *any* suppressor is active, set the plan to Null and exit.
			// This prevents the Default from being injected.
			resp.PlanValue = types.Int32Null()
			return
		}
	}

	// If the loop finishes without finding an active suppressor,
	// resp.PlanValue remains the default (which is correct in this case).
}

// PlanModifyInt64 Core Logic
func (m DefaultValueModifier) PlanModifyInt64(ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	// If the user explicitly set the current attribute, we respect that.
	if !req.ConfigValue.IsNull() {
		return
	}

	// Iterate over all suppressor attribute names
	for _, suppressorName := range m.Suppressors {
		// Get the path to the sibling attribute (same block, different name)
		suppressorPath := req.Path.ParentPath().AtName(suppressorName)

		var suppressorValue types.Bool // Assuming all your suppressors are Bool types

		// Look up the planned value of the suppressor attribute
		req.Plan.GetAttribute(ctx, suppressorPath, &suppressorValue)

		// Check for lookup errors
		if resp.Diagnostics.HasError() {
			return
		}

		// Logic Check: is the suppressor explicitly set (not Null, not Unknown) AND true?
		isSuppressed := !suppressorValue.IsNull() && !suppressorValue.IsUnknown() && suppressorValue.ValueBool()

		if isSuppressed {
			// If *any* suppressor is active, set the plan to Null and exit.
			// This prevents the Default from being injected.
			resp.PlanValue = types.Int64Null()
			return
		}
	}

	// If the loop finishes without finding an active suppressor,
	// resp.PlanValue remains the default (which is correct in this case).
}

// PlanModifyString Core Logic
func (m DefaultValueModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If the user explicitly set the current attribute, we respect that.
	if !req.ConfigValue.IsNull() {
		return
	}

	// Iterate over all suppressor attribute names
	for _, suppressorName := range m.Suppressors {
		// Get the path to the sibling attribute (same block, different name)
		suppressorPath := req.Path.ParentPath().AtName(suppressorName)

		var suppressorValue types.Bool // Assuming all your suppressors are Bool types

		// Look up the planned value of the suppressor attribute
		req.Plan.GetAttribute(ctx, suppressorPath, &suppressorValue)

		// Check for lookup errors
		if resp.Diagnostics.HasError() {
			return
		}

		// Logic Check: is the suppressor explicitly set (not Null, not Unknown) AND true?
		isSuppressed := !suppressorValue.IsNull() && !suppressorValue.IsUnknown() && suppressorValue.ValueBool()

		if isSuppressed {
			// If *any* suppressor is active, set the plan to Null and exit.
			// This prevents the Default from being injected.
			resp.PlanValue = types.StringNull()
			return
		}
	}

	// If the loop finishes without finding an active suppressor,
	// resp.PlanValue remains the default (which is correct in this case).
}
