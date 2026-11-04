//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

package noopplan

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	fpChoiceType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{"flag": tftypes.Bool}}
	fpRootType   = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"name":   tftypes.String,
		"status": tftypes.String,
		"secret": tftypes.String,
		"choice": fpChoiceType,
	}}
)

func TestApply_MatchingFingerprintCollapsesCascade(t *testing.T) {
	private := fakePrivate{}
	config := fpRoot("a", nil, "s1", nil)
	RecordConfig(context.Background(), fpConfig(config), private)

	state := fpRoot("a", "ready", nil, fpChoice(true))
	plan := fpRoot("a", tftypes.UnknownValue, nil, fpChoice(true))

	resp := fpApply(config, state, plan, private)

	assert.True(t, resp.Plan.Raw.Equal(state))
}

func TestApply_RemovedBlockKeepsUnknown(t *testing.T) {
	private := fakePrivate{}
	RecordConfig(context.Background(), fpConfig(fpRoot("a", nil, nil, fpChoice(true))), private)

	config := fpRoot("a", nil, nil, nil)
	state := fpRoot("a", "ready", nil, fpChoice(true))
	plan := fpRoot("a", tftypes.UnknownValue, nil, tftypes.UnknownValue)

	resp := fpApply(config, state, plan, private)

	assert.True(t, resp.Plan.Raw.Equal(plan))
}

func TestApply_NoFingerprintKeepsPlan(t *testing.T) {
	config := fpRoot("a", nil, nil, nil)
	state := fpRoot("a", "ready", nil, fpChoice(true))
	plan := fpRoot("a", tftypes.UnknownValue, nil, tftypes.UnknownValue)

	resp := fpApply(config, state, plan, fakePrivate{})

	assert.True(t, resp.Plan.Raw.Equal(plan))
}

func TestRecordConfig_NilPrivateIsNoOp(t *testing.T) {
	assert.NotPanics(t, func() {
		RecordConfig(context.Background(), fpConfig(fpRoot("a", nil, nil, nil)), nil)
	})
}

func TestFingerprint_IgnoresWriteOnlyValues(t *testing.T) {
	first, ok := fingerprint(context.Background(), fpConfig(fpRoot("a", nil, "s1", nil)))
	require.True(t, ok)
	second, ok := fingerprint(context.Background(), fpConfig(fpRoot("a", nil, "s2", nil)))
	require.True(t, ok)

	assert.Equal(t, first, second)
}

func TestRecordConfig_UnknownConfigClearsFingerprint(t *testing.T) {
	private := fakePrivate{fingerprintKey: []byte(`"stale"`)}

	RecordConfig(context.Background(), fpConfig(fpRoot(tftypes.UnknownValue, nil, nil, nil)), private)

	assert.NotContains(t, private, fingerprintKey)
}

func TestHashValue_StableForMapsAndSets(t *testing.T) {
	entries := map[string]tftypes.Value{}
	for _, k := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		entries[k] = tftypes.NewValue(tftypes.String, k)
	}
	labels := tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, entries)
	setType := tftypes.Set{ElementType: tftypes.Number}
	setA := tftypes.NewValue(setType, []tftypes.Value{tftypes.NewValue(tftypes.Number, 1), tftypes.NewValue(tftypes.Number, 2)})
	setB := tftypes.NewValue(setType, []tftypes.Value{tftypes.NewValue(tftypes.Number, 2), tftypes.NewValue(tftypes.Number, 1)})

	want, ok := hashValue(labels)
	require.True(t, ok)
	for range 50 {
		got, _ := hashValue(labels)
		require.Equal(t, want, got)
	}

	a, ok := hashValue(setA)
	require.True(t, ok)
	b, _ := hashValue(setB)
	assert.Equal(t, a, b)
}

type fakePrivate map[string][]byte

func (f fakePrivate) GetKey(_ context.Context, key string) ([]byte, diag.Diagnostics) {
	return f[key], nil
}

func (f fakePrivate) SetKey(_ context.Context, key string, value []byte) diag.Diagnostics {
	if len(value) == 0 {
		delete(f, key)
		return nil
	}
	f[key] = value
	return nil
}

func fpApply(config, state, plan tftypes.Value, private PrivateReader) *resource.ModifyPlanResponse {
	s := fpSchema()
	req := resource.ModifyPlanRequest{
		Config: tfsdk.Config{Raw: config, Schema: s},
		State:  tfsdk.State{Raw: state, Schema: s},
		Plan:   tfsdk.Plan{Raw: plan, Schema: s},
	}
	resp := &resource.ModifyPlanResponse{Plan: req.Plan}
	apply(context.Background(), req, resp, private)
	return resp
}

func fpConfig(raw tftypes.Value) tfsdk.Config {
	return tfsdk.Config{Raw: raw, Schema: fpSchema()}
}

func fpSchema() schema.Schema {
	return schema.Schema{Attributes: map[string]schema.Attribute{
		"name":   schema.StringAttribute{Required: true},
		"status": schema.StringAttribute{Computed: true},
		"secret": schema.StringAttribute{Optional: true, WriteOnly: true},
		"choice": schema.SingleNestedAttribute{
			Optional: true,
			Computed: true,
			Attributes: map[string]schema.Attribute{
				"flag": schema.BoolAttribute{Optional: true},
			},
		},
	}}
}

// fpRoot builds a root value; each argument is a raw tftypes value (string, nil,
// tftypes.UnknownValue) except choice, which is a ready tftypes.Value, nil or unknown.
func fpRoot(name, status, secret, choice any) tftypes.Value {
	choiceValue, ok := choice.(tftypes.Value)
	if !ok {
		choiceValue = tftypes.NewValue(fpChoiceType, choice)
	}
	return tftypes.NewValue(fpRootType, map[string]tftypes.Value{
		"name":   tftypes.NewValue(tftypes.String, name),
		"status": tftypes.NewValue(tftypes.String, status),
		"secret": tftypes.NewValue(tftypes.String, secret),
		"choice": choiceValue,
	})
}

func fpChoice(flag bool) tftypes.Value {
	return tftypes.NewValue(fpChoiceType, map[string]tftypes.Value{"flag": tftypes.NewValue(tftypes.Bool, flag)})
}
