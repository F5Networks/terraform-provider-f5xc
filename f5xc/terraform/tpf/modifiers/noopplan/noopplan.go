//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

package noopplan

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const fingerprintKey = "noopplan_config_fingerprint"

// PrivateReader is satisfied by the framework's resource private state data.
type PrivateReader interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
}

// PrivateWriter is satisfied by the framework's resource private state data.
type PrivateWriter interface {
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
}

// Apply turns an update plan into a no-op when its only differences from prior
// state are unknown values at paths the user did not configure.
//
// Why: unknown-marking of Computed attributes is triggered by any proposed change,
// including spurious ones from Terraform core, so per-attribute pinning cannot keep
// no-op plans empty for every Computed attribute.
//
// How: the plan is left untouched unless a config fingerprint was recorded at the
// last apply (see RecordConfig) and the current config matches it; a mismatch means
// the user changed something (including removing an Optional+Computed block), and a
// missing fingerprint (resource created by an older build or imported) gives no proof
// either way. Otherwise each unknown plan value whose config value is null (or sits under a null config
// ancestor) is replaced with the prior state value at the same path. If the result
// equals prior state, the plan is set to it and RequiresReplace is cleared. Any
// lookup failure keeps the unknown value, so ambiguous cases remain visible.
//
// Parameters:
//   - ctx: request context
//   - req: the resource ModifyPlan request (config, prior state, private state, proposed plan)
//   - resp: the ModifyPlan response whose Plan is inspected and possibly replaced
//
// Example:
//
//	func (r *ResourceX) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
//		noopplan.Apply(ctx, req, resp)
//	}
func Apply(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	apply(ctx, req, resp, req.Private)
}

// RecordConfig stores a fingerprint of the applied config in private state so a
// later Apply can tell user config changes apart from spurious proposed changes.
//
// How: write-only values are nulled, the config is canonically encoded and hashed with
// SHA-256. When the config cannot be fingerprinted the key is removed, which makes
// Apply keep the next plan as is. Failures are logged, never returned, so a
// successful apply is not turned into an error.
//
// Parameters:
//   - ctx: request context
//   - config: the config used by Create or Update
//   - private: the Create/Update response private state
//
// Example:
//
//	defer noopplan.RecordConfig(ctx, request.Config, response.Private)
func RecordConfig(ctx context.Context, config tfsdk.Config, private PrivateWriter) {
	if private == nil {
		return
	}
	var value []byte
	if fp, ok := fingerprint(ctx, config); ok {
		value, _ = json.Marshal(fp)
	}
	if diags := private.SetKey(ctx, fingerprintKey, value); diags.HasError() {
		tflog.Debug(ctx, "noopplan: unable to record config fingerprint", map[string]any{"diagnostics": diags})
	}
}

// apply implements Apply with an injectable private state reader.
//
// Parameters:
//   - ctx: request context
//   - req: the resource ModifyPlan request
//   - resp: the ModifyPlan response
//   - private: source of the recorded config fingerprint
func apply(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse, private PrivateReader) {
	if req.State.Raw.IsNull() || resp.Plan.Raw.IsNull() || resp.Plan.Raw.IsFullyKnown() {
		return
	}

	stored := storedFingerprint(ctx, private)
	if stored == "" {
		tflog.Debug(ctx, "noopplan: no config fingerprint recorded (older build or import), keeping plan")
		return
	}
	if current, ok := fingerprint(ctx, req.Config); !ok || current != stored {
		tflog.Debug(ctx, "noopplan: config changed since last apply, keeping plan", map[string]any{"fingerprinted": ok})
		return
	}

	candidate, err := tftypes.Transform(resp.Plan.Raw, func(p *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
		if v.IsKnown() || !isNullInConfig(req.Config.Raw, p) {
			return v, nil
		}
		if prior, ok := valueAt(req.State.Raw, p); ok {
			return prior, nil
		}
		return v, nil
	})
	if err != nil {
		tflog.Debug(ctx, "noopplan: transform failed, keeping plan", map[string]any{"error": err.Error()})
		return
	}
	if !candidate.Equal(req.State.Raw) {
		tflog.Debug(ctx, "noopplan: plan differs from prior state, keeping plan", map[string]any{"paths": diffPaths(candidate, req.State.Raw)})
		return
	}

	resp.Plan.Raw = candidate
	resp.RequiresReplace = nil
}

// diffPaths lists up to ten attribute paths where a and b differ, for debug logging.
//
// Parameters:
//   - a, b: values of the same type
//
// Returns:
//   - []string: differing paths, or the Diff error message
func diffPaths(a, b tftypes.Value) []string {
	diffs, err := a.Diff(b)
	if err != nil {
		return []string{err.Error()}
	}
	paths := make([]string, 0, min(len(diffs), 10))
	for _, d := range diffs[:min(len(diffs), 10)] {
		paths = append(paths, d.Path.String())
	}
	return paths
}

// isNullInConfig reports whether the config value at p is null, either directly or
// because an ancestor is null.
//
// Parameters:
//   - config: the raw resource config
//   - p: the attribute path to resolve
//
// Returns:
//   - bool: true when null; false when the value is set, unknown, or cannot be resolved
func isNullInConfig(config tftypes.Value, p *tftypes.AttributePath) bool {
	cur := config
	for _, step := range p.Steps() {
		if cur.IsNull() {
			return true
		}
		if !cur.IsKnown() {
			return false
		}
		next, ok := applyStep(cur, step)
		if !ok {
			return false
		}
		cur = next
	}
	return cur.IsNull()
}

// valueAt resolves the value at p within v.
//
// Parameters:
//   - v: the raw value to walk (e.g. prior state)
//   - p: the attribute path to resolve
//
// Returns:
//   - tftypes.Value: the value at p
//   - bool: false when any step cannot be resolved
func valueAt(v tftypes.Value, p *tftypes.AttributePath) (tftypes.Value, bool) {
	cur := v
	for _, step := range p.Steps() {
		next, ok := applyStep(cur, step)
		if !ok {
			return tftypes.Value{}, false
		}
		cur = next
	}
	return cur, true
}

// applyStep applies a single attribute path step to v.
//
// Parameters:
//   - v: the value to step into
//   - step: attribute name, element index, or element key
//
// Returns:
//   - tftypes.Value: the child value
//   - bool: false when v is null/unknown or the step does not exist
func applyStep(v tftypes.Value, step tftypes.AttributePathStep) (tftypes.Value, bool) {
	if v.IsNull() || !v.IsKnown() {
		return tftypes.Value{}, false
	}
	next, err := v.ApplyTerraform5AttributePathStep(step)
	if err != nil {
		return tftypes.Value{}, false
	}
	child, ok := next.(tftypes.Value)
	return child, ok
}

// storedFingerprint reads the config fingerprint recorded by RecordConfig.
//
// Parameters:
//   - ctx: request context
//   - private: resource private state (may wrap a nil pointer)
//
// Returns:
//   - string: the fingerprint, or "" when absent or unreadable
func storedFingerprint(ctx context.Context, private PrivateReader) string {
	if private == nil {
		return ""
	}
	raw, diags := private.GetKey(ctx, fingerprintKey)
	if diags.HasError() || len(raw) == 0 {
		return ""
	}
	var fp string
	if err := json.Unmarshal(raw, &fp); err != nil {
		return ""
	}
	return fp
}

// fingerprint hashes the config with write-only values removed.
//
// Why: write-only values must never be persisted, not even as a hash, and changes
// to them never produce plan diffs anyway.
//
// Parameters:
//   - ctx: request context
//   - config: the resource config
//
// Returns:
//   - string: hex-encoded SHA-256 of the canonically encoded config
//   - bool: false when the config is null, not fully known, or cannot be encoded
func fingerprint(ctx context.Context, config tfsdk.Config) (string, bool) {
	redacted, ok := redactedConfig(ctx, config)
	if !ok {
		return "", false
	}
	return hashValue(redacted)
}

// redactedConfig returns the fully known config with write-only values nulled.
//
// Parameters:
//   - ctx: request context
//   - config: the resource config
//
// Returns:
//   - tftypes.Value: the redacted config
//   - bool: false when the config is null, not fully known, or cannot be transformed
func redactedConfig(ctx context.Context, config tfsdk.Config) (tftypes.Value, bool) {
	if config.Raw.IsNull() || !config.Raw.IsFullyKnown() {
		return tftypes.Value{}, false
	}
	redacted, err := tftypes.Transform(config.Raw, func(p *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
		if v.IsNull() || config.Schema == nil || len(p.Steps()) == 0 {
			return v, nil
		}
		if attr, err := config.Schema.AttributeAtTerraformPath(ctx, p); err == nil && attr.IsWriteOnly() {
			return tftypes.NewValue(v.Type(), nil), nil
		}
		return v, nil
	})
	if err != nil {
		return tftypes.Value{}, false
	}
	return redacted, true
}

// hashValue returns the hex SHA-256 of a canonical encoding of v.
//
// Why: tftypes msgpack encoding iterates Go maps unsorted, so it is not stable for
// configs that contain maps with more than one key (e.g. labels).
//
// Parameters:
//   - v: value to hash
//
// Returns:
//   - string: hex digest
//   - bool: false when v cannot be encoded
func hashValue(v tftypes.Value) (string, bool) {
	var buf bytes.Buffer
	if err := writeCanonical(&buf, v); err != nil {
		return "", false
	}
	sum := sha256.Sum256(buf.Bytes())
	return hex.EncodeToString(sum[:]), true
}

// writeCanonical appends a deterministic, length-prefixed encoding of v to buf.
// Map and object keys are sorted and set elements are ordered by their encoding.
//
// Parameters:
//   - buf: destination buffer
//   - v: value to encode
//
// Returns:
//   - error: when a value cannot be converted or has an unsupported type
func writeCanonical(buf *bytes.Buffer, v tftypes.Value) error {
	if !v.IsKnown() {
		buf.WriteByte('u')
		return nil
	}
	if v.IsNull() {
		buf.WriteByte('n')
		return nil
	}
	typ := v.Type()
	switch {
	case typ.Is(tftypes.String):
		var s string
		if err := v.As(&s); err != nil {
			return err
		}
		writeToken(buf, 's', s)
	case typ.Is(tftypes.Number):
		n := new(big.Float)
		if err := v.As(&n); err != nil {
			return err
		}
		writeToken(buf, 'd', n.Text('g', -1))
	case typ.Is(tftypes.Bool):
		var b bool
		if err := v.As(&b); err != nil {
			return err
		}
		writeToken(buf, 'b', strconv.FormatBool(b))
	case typ.Is(tftypes.List{}) || typ.Is(tftypes.Tuple{}):
		var elems []tftypes.Value
		if err := v.As(&elems); err != nil {
			return err
		}
		writeToken(buf, 'l', strconv.Itoa(len(elems)))
		for _, e := range elems {
			if err := writeCanonical(buf, e); err != nil {
				return err
			}
		}
	case typ.Is(tftypes.Set{}):
		var elems []tftypes.Value
		if err := v.As(&elems); err != nil {
			return err
		}
		encoded := make([]string, len(elems))
		for i, e := range elems {
			var eb bytes.Buffer
			if err := writeCanonical(&eb, e); err != nil {
				return err
			}
			encoded[i] = eb.String()
		}
		sort.Strings(encoded)
		writeToken(buf, 'S', strconv.Itoa(len(encoded)))
		for _, e := range encoded {
			writeToken(buf, 'e', e)
		}
	case typ.Is(tftypes.Map{}) || typ.Is(tftypes.Object{}):
		var attrs map[string]tftypes.Value
		if err := v.As(&attrs); err != nil {
			return err
		}
		keys := make([]string, 0, len(attrs))
		for k := range attrs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		writeToken(buf, 'm', strconv.Itoa(len(keys)))
		for _, k := range keys {
			writeToken(buf, 'k', k)
			if err := writeCanonical(buf, attrs[k]); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported type %s", typ)
	}
	return nil
}

// writeToken appends tag, the byte length of s, a colon and s, so tokens cannot run together.
//
// Parameters:
//   - buf: destination buffer
//   - tag: single-byte kind marker
//   - s: payload
func writeToken(buf *bytes.Buffer, tag byte, s string) {
	buf.WriteByte(tag)
	buf.WriteString(strconv.Itoa(len(s)))
	buf.WriteByte(':')
	buf.WriteString(s)
}
