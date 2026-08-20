//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

// Package immutableplanmodifier provides a Terraform plan modifier for list attributes
// that must not be changed after the resource is created.
//
// # Background
//
// Some F5 XC API fields (e.g. layer2_intf_ref on subnet) cannot be changed via a
// Replace (PUT) request after the resource has been created. The API returns a 404
// with the message "Invalid Layer2 interface change" when such a change is attempted.
//
// Unlike RequiresReplace, this modifier does NOT trigger a destroy+create cycle.
// Instead it surfaces a plan-time error that tells the user the field is immutable,
// preventing the apply from reaching the API at all.
//
// # Usage in config.json
//
//	"field-plan-modifiers": {
//	  "subnet:ves.io.schema.subnet.GlobalSpecType.layer2_intf_ref": ["immutable"]
//	}
//
// # Generated schema output
//
//	PlanModifiers: []planmodifier.List{
//	    immutableplanmodifier.ImmutableList(),
//	},
package immutableplanmodifier

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

var _ planmodifier.List = immutableListModifier{} // Ensure immutableListModifier implements planmodifier.List

// immutableListModifier is a plan modifier that prevents any change to a list
// attribute after the resource has been created.
type immutableListModifier struct{}

// ImmutableList returns a new immutableListModifier.
func ImmutableList() planmodifier.List {
	return immutableListModifier{}
}

// Description returns a human-readable description of the modifier.
func (m immutableListModifier) Description(_ context.Context) string {
	return "Prevents any change to this attribute after the resource is created. " +
		"The F5 XC API does not allow this field to be modified via a Replace request. " +
		"To change this field you must destroy and recreate the resource."
}

// MarkdownDescription returns a Markdown description of the modifier.
func (m immutableListModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

// PlanModifyList surfaces a plan-time error when the user attempts to change the
// value of an immutable list attribute after the resource has been created.
//
// Decision table:
//
//	state=null (create)  → no-op (any value is allowed on first create)
//	state=known, plan==state → no-op (no change, nothing to block)
//	state=known, plan!=state → error (change is not allowed after create)
func (m immutableListModifier) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	// Allow any value on first create (state is null or unknown).
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}

	// Allow unknown plan values (e.g. during interpolation).
	if req.PlanValue.IsUnknown() {
		return
	}

	// No change — nothing to block.
	if req.PlanValue.Equal(req.StateValue) {
		return
	}

	resp.Diagnostics.AddError(
		"Immutable Attribute Change",
		"The F5 XC API does not allow this attribute to be changed after the resource is created. "+
			"To change this field you must destroy and recreate the resource.\n\n"+
			"Prior value: "+req.StateValue.String()+"\n"+
			"Planned value: "+req.PlanValue.String(),
	)
}
