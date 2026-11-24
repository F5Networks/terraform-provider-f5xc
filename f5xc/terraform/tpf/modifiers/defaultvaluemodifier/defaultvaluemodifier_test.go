// Copyright (c) 2025 F5 Inc. All rights reserved.

package defaultvaluemodifier

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

// TestDefaultValueModifier_Description verifies the Description method returns the expected string.
func TestDefaultValueModifier_Description(t *testing.T) {
	m := DefaultValueModifier{Suppressors: []string{"foo"}}
	desc := m.Description(context.Background())
	assert.Contains(t, desc, "Suppresses the default value")
}

// TestDefaultValueModifier_MarkdownDescription verifies MarkdownDescription delegates to Description.
func TestDefaultValueModifier_MarkdownDescription(t *testing.T) {
	m := DefaultValueModifier{Suppressors: []string{"foo"}}
	desc := m.MarkdownDescription(context.Background())
	assert.Equal(t, m.Description(context.Background()), desc)
}

// TestNewDefaultValueModifierInt32 verifies the constructor for Int32 plan modifier.
func TestNewDefaultValueModifierInt32(t *testing.T) {
	m := NewDefaultValueModifierInt32("foo", "bar")
	assert.Implements(t, (*planmodifier.Int32)(nil), m)
	assert.Equal(t, []string{"foo", "bar"}, m.(DefaultValueModifier).Suppressors)
}

// TestNewDefaultValueModifierString verifies the constructor for String plan modifier.
func TestNewDefaultValueModifierString(t *testing.T) {
	m := NewDefaultValueModifierString("foo", "bar")
	assert.Implements(t, (*planmodifier.String)(nil), m)
	assert.Equal(t, []string{"foo", "bar"}, m.(DefaultValueModifier).Suppressors)
}

var suppressorObjectType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"off": tftypes.Bool,
	"on":  tftypes.Bool,
}}

// suppressorPlan returns a plan where "off" is false and "on" holds onValue.
func suppressorPlan(onValue any) tfsdk.Plan {
	return tfsdk.Plan{
		Schema: schema.Schema{Attributes: map[string]schema.Attribute{
			"off": schema.BoolAttribute{Optional: true},
			"on":  schema.BoolAttribute{Optional: true},
		}},
		Raw: tftypes.NewValue(suppressorObjectType, map[string]tftypes.Value{
			"off": tftypes.NewValue(tftypes.Bool, false),
			"on":  tftypes.NewValue(tftypes.Bool, onValue),
		}),
	}
}

type planCase struct {
	name         string
	config       bool // true when the attribute is set in config
	onValue      any
	seedError    bool
	wantNullPlan bool
}

var planCases = []planCase{
	{name: "config set keeps plan", config: true, onValue: true},
	{name: "active suppressor nulls plan", onValue: true, wantNullPlan: true},
	{name: "inactive suppressors keep plan", onValue: false},
	{name: "unknown suppressor keeps plan", onValue: tftypes.UnknownValue},
	{name: "prior error stops evaluation", onValue: true, seedError: true},
}

func seededDiags(seed bool) diag.Diagnostics {
	var d diag.Diagnostics
	if seed {
		d.AddError("seed", "seed")
	}
	return d
}

// PlanModifyInt32 nulls the plan only when config is null and a suppressor is true.
func TestPlanModifyInt32(t *testing.T) {
	m := DefaultValueModifier{Suppressors: []string{"off", "on"}}
	for _, tc := range planCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := types.Int32Null()
			if tc.config {
				cfg = types.Int32Value(1)
			}
			req := planmodifier.Int32Request{Path: path.Root("val"), ConfigValue: cfg, Plan: suppressorPlan(tc.onValue)}
			resp := &planmodifier.Int32Response{PlanValue: types.Int32Value(5), Diagnostics: seededDiags(tc.seedError)}
			m.PlanModifyInt32(context.Background(), req, resp)
			assert.Equal(t, tc.wantNullPlan, resp.PlanValue.IsNull())
		})
	}
}

// PlanModifyInt64 nulls the plan only when config is null and a suppressor is true.
func TestPlanModifyInt64(t *testing.T) {
	m := NewDefaultValueModifierInt64("off", "on").(DefaultValueModifier)
	for _, tc := range planCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := types.Int64Null()
			if tc.config {
				cfg = types.Int64Value(1)
			}
			req := planmodifier.Int64Request{Path: path.Root("val"), ConfigValue: cfg, Plan: suppressorPlan(tc.onValue)}
			resp := &planmodifier.Int64Response{PlanValue: types.Int64Value(5), Diagnostics: seededDiags(tc.seedError)}
			m.PlanModifyInt64(context.Background(), req, resp)
			assert.Equal(t, tc.wantNullPlan, resp.PlanValue.IsNull())
		})
	}
}

// PlanModifyString nulls the plan only when config is null and a suppressor is true.
func TestPlanModifyString(t *testing.T) {
	m := DefaultValueModifier{Suppressors: []string{"off", "on"}}
	for _, tc := range planCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := types.StringNull()
			if tc.config {
				cfg = types.StringValue("x")
			}
			req := planmodifier.StringRequest{Path: path.Root("val"), ConfigValue: cfg, Plan: suppressorPlan(tc.onValue)}
			resp := &planmodifier.StringResponse{PlanValue: types.StringValue("d"), Diagnostics: seededDiags(tc.seedError)}
			m.PlanModifyString(context.Background(), req, resp)
			assert.Equal(t, tc.wantNullPlan, resp.PlanValue.IsNull())
		})
	}
}
