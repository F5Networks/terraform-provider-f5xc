//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

package noopplan

import (
	"bytes"
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

// Apply leaves the plan unchanged on create, when prior state is null.
func TestApply_CreateKeepsPlan(t *testing.T) {
	plan := fpRoot("a", tftypes.UnknownValue, nil, nil)
	s := fpSchema()
	req := resource.ModifyPlanRequest{
		Config: tfsdk.Config{Raw: fpRoot("a", nil, nil, nil), Schema: s},
		State:  tfsdk.State{Raw: tftypes.NewValue(fpRootType, nil), Schema: s},
		Plan:   tfsdk.Plan{Raw: plan, Schema: s},
	}
	resp := &resource.ModifyPlanResponse{Plan: req.Plan}

	Apply(context.Background(), req, resp)

	assert.True(t, resp.Plan.Raw.Equal(plan))
}

// An unknown value under a config-null ancestor stays unknown when prior state
// cannot resolve its path.
func TestApply_UnresolvablePriorKeepsUnknown(t *testing.T) {
	private := fakePrivate{}
	config := fpRoot("a", nil, nil, nil)
	RecordConfig(context.Background(), fpConfig(config), private)

	state := fpRoot("a", "ready", nil, nil)
	plan := fpRoot("a", "ready", nil, tftypes.NewValue(fpChoiceType, map[string]tftypes.Value{
		"flag": tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue),
	}))

	resp := fpApply(config, state, plan, private)

	assert.True(t, resp.Plan.Raw.Equal(plan))
}

// A prior state value of a different type makes the transform fail, so the plan is kept.
func TestApply_TransformErrorKeepsPlan(t *testing.T) {
	private := fakePrivate{}
	config := fpRoot("a", nil, nil, nil)
	RecordConfig(context.Background(), fpConfig(config), private)

	stateType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"name":   tftypes.String,
		"status": tftypes.Number,
		"secret": tftypes.String,
		"choice": fpChoiceType,
	}}
	state := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"name":   tftypes.NewValue(tftypes.String, "a"),
		"status": tftypes.NewValue(tftypes.Number, 1),
		"secret": tftypes.NewValue(tftypes.String, nil),
		"choice": tftypes.NewValue(fpChoiceType, nil),
	})
	plan := fpRoot("a", tftypes.UnknownValue, nil, nil)

	resp := fpApply(config, state, plan, private)

	assert.True(t, resp.Plan.Raw.Equal(plan))
}

type failingWriter struct{}

func (failingWriter) SetKey(context.Context, string, []byte) diag.Diagnostics {
	var d diag.Diagnostics
	d.AddError("set", "failed")
	return d
}

// RecordConfig never panics or returns when the private state write fails.
func TestRecordConfig_WriteErrorIsIgnored(t *testing.T) {
	assert.NotPanics(t, func() {
		RecordConfig(context.Background(), fpConfig(fpRoot("a", nil, nil, nil)), failingWriter{})
	})
}

// diffPaths returns the Diff error message when the values have different types.
func TestDiffPaths_TypeMismatch(t *testing.T) {
	got := diffPaths(tftypes.NewValue(tftypes.String, "a"), tftypes.NewValue(tftypes.Number, 1))
	require.Len(t, got, 1)
	assert.NotEmpty(t, got[0])
}

// isNullInConfig is true under a null ancestor and false for unknown or unresolvable paths.
func TestIsNullInConfig(t *testing.T) {
	flag := tftypes.NewAttributePath().WithAttributeName("choice").WithAttributeName("flag")

	assert.True(t, isNullInConfig(fpRoot("a", nil, nil, nil), flag), "null ancestor")
	assert.False(t, isNullInConfig(fpRoot("a", nil, nil, tftypes.UnknownValue), flag), "unknown ancestor")
	assert.False(t, isNullInConfig(fpRoot("a", nil, nil, nil), tftypes.NewAttributePath().WithAttributeName("missing").WithAttributeName("x")), "missing attribute")
}

// valueAt and applyStep fail for null values and missing attributes.
func TestValueAtAndApplyStep_Failures(t *testing.T) {
	_, ok := valueAt(fpRoot("a", nil, nil, nil), tftypes.NewAttributePath().WithAttributeName("missing"))
	assert.False(t, ok)

	_, ok = applyStep(tftypes.NewValue(fpRootType, nil), tftypes.AttributeName("name"))
	assert.False(t, ok)
}

type errorReader struct {
	raw   []byte
	diags diag.Diagnostics
}

func (r errorReader) GetKey(context.Context, string) ([]byte, diag.Diagnostics) {
	return r.raw, r.diags
}

// storedFingerprint returns "" for a nil reader, read errors, and malformed JSON.
func TestStoredFingerprint_Invalid(t *testing.T) {
	ctx := context.Background()
	var readErr diag.Diagnostics
	readErr.AddError("get", "failed")

	assert.Empty(t, storedFingerprint(ctx, nil))
	assert.Empty(t, storedFingerprint(ctx, errorReader{raw: []byte(`"x"`), diags: readErr}))
	assert.Empty(t, storedFingerprint(ctx, errorReader{raw: []byte(`not-json`)}))
	assert.Equal(t, "x", storedFingerprint(ctx, errorReader{raw: []byte(`"x"`)}))
}

// hashValue fails for a value whose type has no canonical encoding.
func TestHashValue_UnsupportedType(t *testing.T) {
	_, ok := hashValue(tftypes.NewValue(tftypes.DynamicPseudoType, "x"))
	assert.False(t, ok)
}

// writeCanonical encodes unknowns, lists, and tuples, and propagates nested encoding errors.
func TestWriteCanonical_Collections(t *testing.T) {
	dyn := tftypes.NewValue(tftypes.DynamicPseudoType, "x")

	var buf bytes.Buffer
	require.NoError(t, writeCanonical(&buf, tftypes.NewValue(tftypes.String, tftypes.UnknownValue)))
	assert.Equal(t, "u", buf.String())

	buf.Reset()
	list := tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "a")})
	require.NoError(t, writeCanonical(&buf, list))
	assert.Equal(t, "l1:1s1:a", buf.String())

	tupleType := tftypes.Tuple{ElementTypes: []tftypes.Type{tftypes.DynamicPseudoType}}
	setType := tftypes.Set{ElementType: tftypes.DynamicPseudoType}
	mapType := tftypes.Map{ElementType: tftypes.DynamicPseudoType}

	invalid := map[string]tftypes.Value{
		"tuple": tftypes.NewValue(tupleType, []tftypes.Value{dyn}),
		"set":   tftypes.NewValue(setType, []tftypes.Value{dyn}),
		"map":   tftypes.NewValue(mapType, map[string]tftypes.Value{"k": dyn}),
	}
	for name, v := range invalid {
		t.Run(name, func(t *testing.T) {
			assert.Error(t, writeCanonical(&bytes.Buffer{}, v))
		})
	}
}
