//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

package noopplan_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/modifiers/noopplan"
)

var (
	choiceType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{"flag": tftypes.Bool}}
	rootType   = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"name":   tftypes.String,
		"status": tftypes.String,
		"choice": choiceType,
	}}
	unknownString = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
	nullString    = tftypes.NewValue(tftypes.String, nil)
	nullChoice    = tftypes.NewValue(choiceType, nil)
)

func TestApply_CascadeOnlyPlanBecomesNoOp(t *testing.T) {
	state := root(str("a"), str("ready"), choice(true))
	config := root(str("a"), nullString, nullChoice)
	plan := root(str("a"), unknownString, choice(true))

	resp := run(t, config, state, plan)

	assert.True(t, resp.Plan.Raw.Equal(state))
	assert.Nil(t, resp.RequiresReplace)
}

func TestApply_UnknownComputedObjectRestoredFromState(t *testing.T) {
	state := root(str("a"), str("ready"), choice(true))
	config := root(str("a"), nullString, nullChoice)
	plan := root(str("a"), unknownString, tftypes.NewValue(choiceType, tftypes.UnknownValue))

	resp := run(t, config, state, plan)

	assert.True(t, resp.Plan.Raw.Equal(state))
}

func TestApply_RealChangeKeepsUnknowns(t *testing.T) {
	state := root(str("a"), str("ready"), choice(true))
	config := root(str("b"), nullString, nullChoice)
	plan := root(str("b"), unknownString, choice(true))

	resp := run(t, config, state, plan)

	assert.True(t, resp.Plan.Raw.Equal(plan))
	assert.NotNil(t, resp.RequiresReplace)
}

func TestApply_UnknownConfigValueIsNotTreatedAsNoOp(t *testing.T) {
	state := root(str("a"), str("ready"), choice(true))
	config := root(unknownString, nullString, nullChoice)
	plan := root(unknownString, unknownString, choice(true))

	resp := run(t, config, state, plan)

	assert.True(t, resp.Plan.Raw.Equal(plan))
}

func TestApply_CreateIsUntouched(t *testing.T) {
	state := tftypes.NewValue(rootType, nil)
	config := root(str("a"), nullString, nullChoice)
	plan := root(str("a"), unknownString, tftypes.NewValue(choiceType, tftypes.UnknownValue))

	resp := run(t, config, state, plan)

	assert.True(t, resp.Plan.Raw.Equal(plan))
}

func run(t *testing.T, config, state, plan tftypes.Value) *resource.ModifyPlanResponse {
	t.Helper()
	s := testSchema()
	req := resource.ModifyPlanRequest{
		Config: tfsdk.Config{Raw: config, Schema: s},
		State:  tfsdk.State{Raw: state, Schema: s},
		Plan:   tfsdk.Plan{Raw: plan, Schema: s},
	}
	resp := &resource.ModifyPlanResponse{
		Plan:            req.Plan,
		RequiresReplace: path.Paths{path.Root("name")},
	}
	private := fakePrivate{}
	noopplan.RecordConfig(context.Background(), req.Config, private)
	noopplan.ApplyWithPrivate(context.Background(), req, resp, private)
	return resp
}

type fakePrivate map[string][]byte

func (f fakePrivate) GetKey(_ context.Context, key string) ([]byte, diag.Diagnostics) {
	return f[key], nil
}

func (f fakePrivate) SetKey(_ context.Context, key string, value []byte) diag.Diagnostics {
	f[key] = value
	return nil
}

func testSchema() schema.Schema {
	return schema.Schema{Attributes: map[string]schema.Attribute{
		"name":   schema.StringAttribute{Required: true},
		"status": schema.StringAttribute{Computed: true},
		"choice": schema.SingleNestedAttribute{
			Optional: true,
			Computed: true,
			Attributes: map[string]schema.Attribute{
				"flag": schema.BoolAttribute{Optional: true},
			},
		},
	}}
}

func root(name, status, choice tftypes.Value) tftypes.Value {
	return tftypes.NewValue(rootType, map[string]tftypes.Value{
		"name":   name,
		"status": status,
		"choice": choice,
	})
}

func choice(flag bool) tftypes.Value {
	return tftypes.NewValue(choiceType, map[string]tftypes.Value{"flag": tftypes.NewValue(tftypes.Bool, flag)})
}

func str(s string) tftypes.Value {
	return tftypes.NewValue(tftypes.String, s)
}
