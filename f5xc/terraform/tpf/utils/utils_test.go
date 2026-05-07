//
// Copyright (c) 2025 F5 Inc. All rights reserved.
//

package utils

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFormatToAbsolutePath(t *testing.T) {
	// The stdlib TLS library strips "file:///" (8 chars) from the URI before calling
	// os.ReadFile. To ensure the remainder is a valid absolute path (starting with "/"),
	// FormatToAbsolutePath must produce a URI with four slashes: "file:////absolute/path".
	// After stripping "file:///" (8 chars): "file:////etc/hosts"[8:] == "/etc/hosts" ✓
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string returns empty string",
			input: "",
			want:  "",
		},
		{
			name:  "plain absolute path gets file:/// prefix (four slashes total)",
			input: "/etc/hosts",
			want:  "file:////etc/hosts",
		},
		{
			name:  "plain relative path gets leading slash and file:/// prefix",
			input: "etc/hosts",
			want:  "file:////etc/hosts",
		},
		{
			name:  "existing file:/// URI is normalized to four-slash form",
			input: "file:///etc/hosts",
			want:  "file:////etc/hosts",
		},
		{
			name:  "existing file://// URI is idempotent",
			input: "file:////etc/hosts",
			want:  "file:////etc/hosts",
		},
		{
			name:  "file:// URI without leading slash on path gets slash added",
			input: "file://etc/hosts",
			want:  "file:////etc/hosts",
		},
		{
			name:  "absolute path with subdirectories",
			input: "/Users/foo/bar/baz.txt",
			want:  "file:////Users/foo/bar/baz.txt",
		},
		{
			name:  "file URI with absolute path and subdirectories",
			input: "file:///Users/foo/bar/baz.txt",
			want:  "file:////Users/foo/bar/baz.txt",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatToAbsolutePath(tc.input)
			if got != tc.want {
				t.Errorf("FormatToAbsolutePath(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestGetListStringDefaultValue(t *testing.T) {
	val := GetListStringDefaultValue("foo")
	if val.IsNull() {
		t.Errorf("Expected non-null ListValue")
	}
	if val.Elements()[0].(types.String).ValueString() != "foo" {
		t.Errorf("Expected value 'foo', got %v", val.Elements()[0])
	}
}

func TestParseTime(t *testing.T) {
	ts, err := ParseTime("2023-01-01T12:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ts == nil {
		t.Fatalf("expected non-nil timestamp")
	}
}

func TestParseTime_InvalidInput(t *testing.T) {
	ts, err := ParseTime("not-a-time")
	if err == nil {
		t.Fatalf("expected error for invalid time string, got nil")
	}
	if ts != nil {
		t.Fatalf("expected nil timestamp on error, got %v", ts)
	}
}

func TestUint32MapToTerraformMap(t *testing.T) {
	in := map[uint32]string{1: "a", 2: "b"}
	m, diags := Uint32MapToTerraformMap(in)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if m.IsNull() {
		t.Fatalf("expected non-null map")
	}
	if len(m.Elements()) != 2 {
		t.Errorf("expected 2 elements, got %d", len(m.Elements()))
	}
}

func TestUint32MapToTerraformMap_Nil(t *testing.T) {
	m, diags := Uint32MapToTerraformMap(nil)
	if diags != nil && diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !m.IsNull() {
		t.Fatalf("expected null map for nil input")
	}
}

func TestStringMapToTerraformMap(t *testing.T) {
	in := map[string]string{"foo": "bar"}
	m, diags := StringMapToTerraformMap(in)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if m.IsNull() {
		t.Fatalf("expected non-null map")
	}
	if m.Elements()["foo"].(types.String).ValueString() != "bar" {
		t.Errorf("expected 'bar', got %v", m.Elements()["foo"])
	}
}

func TestStringMapToTerraformMap_Nil(t *testing.T) {
	m, diags := StringMapToTerraformMap(nil)
	if diags != nil && diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !m.IsNull() {
		t.Fatalf("expected null map for nil input")
	}
}

func TestStringMapToTerraformMap_Empty(t *testing.T) {
	m, diags := StringMapToTerraformMap(map[string]string{})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if m.IsNull() {
		t.Fatalf("expected non-null map for empty input")
	}
	if len(m.Elements()) != 0 {
		t.Errorf("expected 0 elements, got %d", len(m.Elements()))
	}
}

func TestGoMapToObjectMapValue(t *testing.T) {
	type testStruct struct {
		A string `tfsdk:"a"`
	}
	attrMap := map[string]attr.Type{"a": types.StringType}
	goMap := map[string]testStruct{"x": {A: "y"}}
	mv, diags := GoMapToObjectMapValue(context.Background(), attrMap, goMap)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if mv.IsNull() {
		t.Fatalf("expected non-null map value")
	}
}

func TestGoMapToObjectMapValue_Empty(t *testing.T) {
	type testStruct struct {
		A string `tfsdk:"a"`
	}
	attrMap := map[string]attr.Type{"a": types.StringType}
	mv, diags := GoMapToObjectMapValue(context.Background(), attrMap, map[string]testStruct{})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !mv.IsNull() {
		t.Fatalf("expected null map value for empty input")
	}
}

func TestGoStructToObjectValue(t *testing.T) {
	type testStruct struct {
		A string `tfsdk:"a"`
	}
	attrMap := map[string]attr.Type{"a": types.StringType}
	obj, diags := GoStructToObjectValue(context.Background(), attrMap, testStruct{A: "foo"})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if obj.IsNull() {
		t.Fatalf("expected non-null object value")
	}
}

func TestGoStructSliceToListValue(t *testing.T) {
	type testStruct struct {
		A string `tfsdk:"a"`
	}
	attrMap := map[string]attr.Type{"a": types.StringType}
	slice := []testStruct{{A: "foo"}, {A: "bar"}}
	lv, diags := GoStructSliceToListValue(context.Background(), attrMap, slice)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if lv.IsNull() {
		t.Fatalf("expected non-null list value")
	}
	if len(lv.Elements()) != 2 {
		t.Errorf("expected 2 elements, got %d", len(lv.Elements()))
	}
}

func TestGoStructSliceToListValue_Empty(t *testing.T) {
	type testStruct struct {
		A string `tfsdk:"a"`
	}
	attrMap := map[string]attr.Type{"a": types.StringType}
	lv, diags := GoStructSliceToListValue(context.Background(), attrMap, []testStruct{})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if lv.IsNull() {
		t.Fatalf("expected non-null list value for empty slice")
	}
	if len(lv.Elements()) != 0 {
		t.Errorf("expected 0 elements, got %d", len(lv.Elements()))
	}
}

// For WaitForPendingInitializers, we use a mock resource and getter

type mockPendingResource struct {
	calls   int
	pending [][]string
}

func (m *mockPendingResource) GetObjPendingInitializers() []string {
	if m.calls < len(m.pending) {
		res := m.pending[m.calls]
		m.calls++
		return res
	}
	// Once all pre-configured responses are exhausted, keep returning the last entry
	// so that a timeout test can simulate a resource that never finishes initializing.
	if len(m.pending) > 0 {
		return m.pending[len(m.pending)-1]
	}
	return nil
}

// testWaitConfig returns a fast configuration for use in tests: zero-duration sleeps
// and a small retry count so tests complete instantly.
func testWaitConfig(maxRetries int) waitForPendingInitializersConfig {
	return waitForPendingInitializersConfig{
		maxRetries:     maxRetries,
		maxBackoff:     0,
		initialBackoff: 0,
		sleep:          func(time.Duration) {},
	}
}

func TestWaitForPendingInitializers(t *testing.T) {
	ctx := context.Background()
	resource := &mockPendingResource{
		pending: [][]string{{"init1"}, {"init2"}, {}},
	}
	getter := func(ctx context.Context, ns, name string) (*mockPendingResource, error) {
		return resource, nil
	}
	err := waitForPendingInitializersWithConfig(ctx, "n", "ns", "uid", getter, testWaitConfig(5))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestWaitForPendingInitializers_404(t *testing.T) {
	ctx := context.Background()
	getter := func(ctx context.Context, ns, name string) (*mockPendingResource, error) {
		return nil, fmt.Errorf("status code 404")
	}
	err := waitForPendingInitializersWithConfig(ctx, "n", "ns", "uid", getter, testWaitConfig(5))
	if err == nil || !strings.Contains(err.Error(), "no longer exists") {
		t.Fatalf("expected 404 error, got %v", err)
	}
}

func TestWaitForPendingInitializers_GenericError(t *testing.T) {
	ctx := context.Background()
	getter := func(ctx context.Context, ns, name string) (*mockPendingResource, error) {
		return nil, fmt.Errorf("internal server error")
	}
	err := waitForPendingInitializersWithConfig(ctx, "n", "ns", "uid", getter, testWaitConfig(5))
	if err == nil || !strings.Contains(err.Error(), "error finding resource") {
		t.Fatalf("expected generic error, got %v", err)
	}
}

func TestWaitForPendingInitializers_Timeout(t *testing.T) {
	ctx := context.Background()
	// The mock always returns pending initializers (last entry is non-empty),
	// so the function must exhaust all retries and return a timeout error.
	resource := &mockPendingResource{
		pending: [][]string{{"init1"}, {"init2"}, {"init3"}},
	}
	getter := func(ctx context.Context, ns, name string) (*mockPendingResource, error) {
		return resource, nil
	}
	err := waitForPendingInitializersWithConfig(ctx, "n", "ns", "uid", getter, testWaitConfig(5))
	if err == nil || !strings.Contains(err.Error(), "still has pending initializers") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}
