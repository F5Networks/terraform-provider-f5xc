//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

package immutableplanmodifier

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

var _ planmodifier.Object = immutableObjectModifier{}

// immutableObjectModifier is a plan modifier that prevents any change to an object
// attribute after the resource has been created.
type immutableObjectModifier struct{}

// ImmutableObject returns a new immutableObjectModifier.
func ImmutableObject() planmodifier.Object {
	return immutableObjectModifier{}
}

// Description returns a human-readable description of the modifier.
func (m immutableObjectModifier) Description(_ context.Context) string {
	return "Prevents any change to this attribute after the resource is created. " +
		"The F5 XC API does not allow this field to be modified via a Replace request. " +
		"To change this field you must destroy and recreate the resource."
}

// MarkdownDescription returns a Markdown description of the modifier.
func (m immutableObjectModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

// PlanModifyObject surfaces a plan-time error when the user attempts to change the
// value of an immutable object attribute after the resource has been created.
func (m immutableObjectModifier) PlanModifyObject(ctx context.Context, req planmodifier.ObjectRequest, resp *planmodifier.ObjectResponse) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}

	if req.PlanValue.IsUnknown() {
		return
	}

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
