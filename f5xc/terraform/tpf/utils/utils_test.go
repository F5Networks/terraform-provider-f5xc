//
// Copyright (c) 2025 F5 Inc. All rights reserved.
//

package utils

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	googleProtobuf "github.com/gogo/protobuf/types"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestResolveEnumValue_DirectMatch(t *testing.T) {
	enumMap := map[string]int32{"SERVICE_HEALTH_LOW": 5}

	v, ok := ResolveEnumValue("SERVICE_HEALTH_LOW", enumMap, false)
	if !ok {
		t.Fatal("expected direct enum match")
	}
	if v != 5 {
		t.Fatalf("ResolveEnumValue returned %d, want 5", v)
	}
}

func TestResolveEnumValue_DisplayNameMatchEnabled(t *testing.T) {
	enumMap := map[string]int32{"SERVICE_HEALTH_LOW": 5}

	v, ok := ResolveEnumValue("ServiceHealthLow", enumMap, true)
	if !ok {
		t.Fatal("expected display-name enum match")
	}
	if v != 5 {
		t.Fatalf("ResolveEnumValue returned %d, want 5", v)
	}
}

func TestResolveEnumValue_DisplayNameMatchDisabled(t *testing.T) {
	enumMap := map[string]int32{"SERVICE_HEALTH_LOW": 5}

	_, ok := ResolveEnumValue("ServiceHealthLow", enumMap, false)
	if ok {
		t.Fatal("expected display-name enum lookup to fail when fallback is disabled")
	}
}

func TestResolveEnumValue_AcronymDisplayNameMatchEnabled(t *testing.T) {
	enumMap := map[string]int32{"L7DDOS": 31}

	v, ok := ResolveEnumValue("L7DDoS", enumMap, true)
	if !ok {
		t.Fatal("expected acronym display-name enum match")
	}
	if v != 31 {
		t.Fatalf("ResolveEnumValue returned %d, want 31", v)
	}
}

func TestResolveEnumValue_NoMatch(t *testing.T) {
	enumMap := map[string]int32{"SERVICE_HEALTH_LOW": 5}

	_, ok := ResolveEnumValue("NotARealAlert", enumMap, true)
	if ok {
		t.Fatal("expected enum lookup to fail for unknown value")
	}
}

func TestResolveEnumStateValue_DisplayNameDisabled(t *testing.T) {
	got := ResolveEnumStateValue("SERVICE_HEALTH_LOW", false)
	if got != "SERVICE_HEALTH_LOW" {
		t.Fatalf("ResolveEnumStateValue returned %q, want SERVICE_HEALTH_LOW", got)
	}
}

func TestResolveEnumStateValue_DisplayNameEnabled(t *testing.T) {
	got := ResolveEnumStateValue("SERVICE_HEALTH_LOW", true)
	if got != "ServiceHealthLow" {
		t.Fatalf("ResolveEnumStateValue returned %q, want ServiceHealthLow", got)
	}
}

func TestResolveEnumStateValue_Acronym(t *testing.T) {
	got := ResolveEnumStateValue("L7_DDOS", true)
	if got != "L7DDoS" {
		t.Fatalf("ResolveEnumStateValue returned %q, want L7DDoS", got)
	}
}

func TestResolveEnumStateValue_AlreadyDisplayName(t *testing.T) {
	got := ResolveEnumStateValue("ServiceHealthLow", true)
	if got != "ServiceHealthLow" {
		t.Fatalf("ResolveEnumStateValue returned %q, want ServiceHealthLow", got)
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

// --- WaitForPendingFinalizers tests ---

// mockFinalizerResource is a mock that implements PendingFinalizersResource.
type mockFinalizerResource struct{}

func (m *mockFinalizerResource) GetObjFinalizers() []string {
	return nil
}

// testFinalizerWaitConfig returns a fast configuration for finalizer tests.
func testFinalizerWaitConfig(maxRetries int) waitForPendingFinalizersConfig {
	return waitForPendingFinalizersConfig{
		maxRetries:     maxRetries,
		maxBackoff:     0,
		initialBackoff: 0,
		sleep:          func(time.Duration) {},
	}
}

// TestWaitForPendingFinalizers_404 verifies that a 404 error from the getter is treated as
// successful completion (resource is gone, finalizers are done).
func TestWaitForPendingFinalizers_404(t *testing.T) {
	ctx := context.Background()
	getter := func(ctx context.Context, ns, name string) (*mockFinalizerResource, error) {
		return nil, fmt.Errorf("status code 404")
	}
	err := waitForPendingFinalizersWithConfig(ctx, "n", "ns", "uid", getter, testFinalizerWaitConfig(5))
	if err != nil {
		t.Fatalf("expected nil error for 404 (resource gone), got %v", err)
	}
}

// TestWaitForPendingFinalizers_GenericError verifies that a non-404 error is propagated.
func TestWaitForPendingFinalizers_GenericError(t *testing.T) {
	ctx := context.Background()
	getter := func(ctx context.Context, ns, name string) (*mockFinalizerResource, error) {
		return nil, fmt.Errorf("internal server error")
	}
	err := waitForPendingFinalizersWithConfig(ctx, "n", "ns", "uid", getter, testFinalizerWaitConfig(5))
	if err == nil || !strings.Contains(err.Error(), "error polling resource") {
		t.Fatalf("expected polling error, got %v", err)
	}
}

// TestWaitForPendingFinalizers_Timeout verifies that an error is returned when the resource
// still exists after all retries (finalizers never complete).
func TestWaitForPendingFinalizers_Timeout(t *testing.T) {
	ctx := context.Background()
	resource := &mockFinalizerResource{}
	getter := func(ctx context.Context, ns, name string) (*mockFinalizerResource, error) {
		// Resource always exists — finalizers never complete.
		return resource, nil
	}
	err := waitForPendingFinalizersWithConfig(ctx, "n", "ns", "uid", getter, testFinalizerWaitConfig(3))
	if err == nil || !strings.Contains(err.Error(), "still exists after") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

// TestWaitForPendingFinalizers_EventuallyGone verifies that the function succeeds when the
// resource exists for a few attempts and then returns 404.
func TestWaitForPendingFinalizers_EventuallyGone(t *testing.T) {
	ctx := context.Background()
	callCount := 0
	resource := &mockFinalizerResource{}
	getter := func(ctx context.Context, ns, name string) (*mockFinalizerResource, error) {
		callCount++
		if callCount >= 3 {
			return nil, fmt.Errorf("status code 404")
		}
		return resource, nil
	}
	err := waitForPendingFinalizersWithConfig(ctx, "n", "ns", "uid", getter, testFinalizerWaitConfig(5))
	if err != nil {
		t.Fatalf("expected nil error when resource eventually disappears, got %v", err)
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "status 404", err: fmt.Errorf("status code 404"), want: true},
		{name: "mixed case notfound", err: fmt.Errorf("rpc error: NotFound"), want: true},
		{name: "other error", err: fmt.Errorf("status code 500"), want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsNotFoundError(tc.err)
			if got != tc.want {
				t.Fatalf("IsNotFoundError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestIsTransientDeleteError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "pending change", err: fmt.Errorf("Previous DNS zone change is pending"), want: true},
		{name: "deployment state", err: fmt.Errorf("Failed to get zone deployment state"), want: true},
		{name: "status 500", err: fmt.Errorf("request failed with status code 500"), want: true},
		{name: "status 503", err: fmt.Errorf("request failed with status code 503"), want: true},
		{name: "status 404", err: fmt.Errorf("status code 404"), want: false},
		{name: "generic", err: fmt.Errorf("invalid argument"), want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsTransientDeleteError(tc.err)
			if got != tc.want {
				t.Fatalf("IsTransientDeleteError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestDeleteWithTransientRetry_NilDeleteFn(t *testing.T) {
	err := deleteWithTransientRetryWithConfig(
		context.Background(),
		"dns_zone",
		nil,
		transientDeleteRetryConfig{maxAttempts: 3, retryInterval: 0, sleep: func(time.Duration) {}},
	)
	if err == nil || !strings.Contains(err.Error(), "delete operation is nil") {
		t.Fatalf("expected nil delete operation error, got %v", err)
	}
}

func TestDeleteWithTransientRetry_SuccessFirstAttempt(t *testing.T) {
	calls := 0
	err := deleteWithTransientRetryWithConfig(
		context.Background(),
		"dns_zone",
		func() error {
			calls++
			return nil
		},
		transientDeleteRetryConfig{maxAttempts: 3, retryInterval: 0, sleep: func(time.Duration) {}},
	)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestDeleteWithTransientRetry_NotFoundTreatedAsSuccess(t *testing.T) {
	calls := 0
	err := deleteWithTransientRetryWithConfig(
		context.Background(),
		"dns_zone",
		func() error {
			calls++
			return fmt.Errorf("status code 404")
		},
		transientDeleteRetryConfig{maxAttempts: 3, retryInterval: 0, sleep: func(time.Duration) {}},
	)
	if err != nil {
		t.Fatalf("expected nil error for not found, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestDeleteWithTransientRetry_TransientThenSuccess(t *testing.T) {
	calls := 0
	err := deleteWithTransientRetryWithConfig(
		context.Background(),
		"dns_zone",
		func() error {
			calls++
			if calls == 1 {
				return fmt.Errorf("Previous DNS zone change is pending")
			}
			return nil
		},
		transientDeleteRetryConfig{maxAttempts: 3, retryInterval: 0, sleep: func(time.Duration) {}},
	)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestDeleteWithTransientRetry_HTTP409Conflict(t *testing.T) {
	calls := 0
	err := deleteWithTransientRetryWithConfig(
		context.Background(),
		"service_policy",
		func() error {
			calls++
			if calls == 1 {
				return fmt.Errorf("Unsuccessful DELETE at URL ... status code 409, body: service_policy is still being referred by 1 objects")
			}
			return nil
		},
		transientDeleteRetryConfig{maxAttempts: 3, retryInterval: 0, sleep: func(time.Duration) {}},
	)
	if err != nil {
		t.Fatalf("expected nil error after retry on 409, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestDeleteWithTransientRetry_TransientExhausted(t *testing.T) {
	calls := 0
	err := deleteWithTransientRetryWithConfig(
		context.Background(),
		"dns_zone",
		func() error {
			calls++
			return fmt.Errorf("status code 503")
		},
		transientDeleteRetryConfig{maxAttempts: 3, retryInterval: 0, sleep: func(time.Duration) {}},
	)
	if err == nil || !strings.Contains(err.Error(), "status code 503") {
		t.Fatalf("expected exhausted transient retry error, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestDeleteWithTransientRetry_NonTransientFailsImmediately(t *testing.T) {
	calls := 0
	err := deleteWithTransientRetryWithConfig(
		context.Background(),
		"dns_zone",
		func() error {
			calls++
			return fmt.Errorf("permission denied")
		},
		transientDeleteRetryConfig{maxAttempts: 3, retryInterval: 0, sleep: func(time.Duration) {}},
	)
	if err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("expected non-transient error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestIsTransientCreateError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "pending change", err: fmt.Errorf("Previous DNS zone change is pending"), want: true},
		{name: "deployment state", err: fmt.Errorf("Failed to get zone deployment state"), want: true},
		{name: "status 500", err: fmt.Errorf("Unsuccessful custom API POST ... status code 500"), want: true},
		{name: "status 503", err: fmt.Errorf("request failed with status code 503"), want: true},
		{name: "status 404", err: fmt.Errorf("status code 404"), want: false},
		{name: "generic", err: fmt.Errorf("invalid argument"), want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsTransientCreateError(tc.err)
			if got != tc.want {
				t.Fatalf("IsTransientCreateError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestCreateWithTransientRetry_NilCreateFn(t *testing.T) {
	_, err := createWithTransientRetryWithConfig(
		context.Background(),
		"dns_zone_record",
		nil,
		transientCreateRetryConfig{maxAttempts: 3, retryInterval: 0, sleep: func(time.Duration) {}},
	)
	if err == nil || !strings.Contains(err.Error(), "create operation is nil") {
		t.Fatalf("expected nil create operation error, got %v", err)
	}
}

func TestCreateWithTransientRetry_SuccessFirstAttempt(t *testing.T) {
	calls := 0
	want := &googleProtobuf.Empty{}
	resp, err := createWithTransientRetryWithConfig(
		context.Background(),
		"dns_zone_record",
		func() (proto.Message, error) {
			calls++
			return want, nil
		},
		transientCreateRetryConfig{maxAttempts: 3, retryInterval: 0, sleep: func(time.Duration) {}},
	)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resp != want {
		t.Fatalf("expected create response to be returned, got %v", resp)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestCreateWithTransientRetry_TransientThenSuccess(t *testing.T) {
	calls := 0
	want := &googleProtobuf.Empty{}
	resp, err := createWithTransientRetryWithConfig(
		context.Background(),
		"dns_zone_record",
		func() (proto.Message, error) {
			calls++
			if calls == 1 {
				return nil, fmt.Errorf("Failed to get zone deployment state")
			}
			return want, nil
		},
		transientCreateRetryConfig{maxAttempts: 3, retryInterval: 0, sleep: func(time.Duration) {}},
	)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resp != want {
		t.Fatalf("expected create response after retry, got %v", resp)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestCreateWithTransientRetry_TransientExhausted(t *testing.T) {
	calls := 0
	_, err := createWithTransientRetryWithConfig(
		context.Background(),
		"dns_zone_record",
		func() (proto.Message, error) {
			calls++
			return nil, fmt.Errorf("status code 500")
		},
		transientCreateRetryConfig{maxAttempts: 3, retryInterval: 0, sleep: func(time.Duration) {}},
	)
	if err == nil || !strings.Contains(err.Error(), "status code 500") {
		t.Fatalf("expected exhausted transient retry error, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestCreateWithTransientRetry_NonTransientFailsImmediately(t *testing.T) {
	calls := 0
	_, err := createWithTransientRetryWithConfig(
		context.Background(),
		"dns_zone_record",
		func() (proto.Message, error) {
			calls++
			return nil, fmt.Errorf("permission denied")
		},
		transientCreateRetryConfig{maxAttempts: 3, retryInterval: 0, sleep: func(time.Duration) {}},
	)
	if err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("expected non-transient error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

// --- Additional edge-case tests for existing functions ---

// TestGetListStringDefaultValue_EmptyString verifies that an empty string is stored correctly.
func TestGetListStringDefaultValue_EmptyString(t *testing.T) {
	val := GetListStringDefaultValue("")
	if val.IsNull() {
		t.Errorf("expected non-null ListValue for empty string default")
	}
	if val.Elements()[0].(types.String).ValueString() != "" {
		t.Errorf("expected empty string value, got %v", val.Elements()[0])
	}
}

// TestGetListStringDefaultValue_SingleElement verifies that the list always contains exactly one element.
func TestGetListStringDefaultValue_SingleElement(t *testing.T) {
	val := GetListStringDefaultValue("test-value")
	if len(val.Elements()) != 1 {
		t.Errorf("expected exactly 1 element, got %d", len(val.Elements()))
	}
}

// TestParseTime_EmptyString verifies that an empty string returns an error.
func TestParseTime_EmptyString(t *testing.T) {
	ts, err := ParseTime("")
	if err == nil {
		t.Fatalf("expected error for empty time string, got nil")
	}
	if ts != nil {
		t.Fatalf("expected nil timestamp on error, got %v", ts)
	}
}

// TestParseTime_WithTimezone verifies that a RFC3339 timestamp with a non-UTC timezone is parsed correctly.
func TestParseTime_WithTimezone(t *testing.T) {
	ts, err := ParseTime("2023-06-15T10:30:00+02:00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ts == nil {
		t.Fatalf("expected non-nil timestamp")
	}
	// 10:30 +02:00 == 08:30 UTC → Unix timestamp 1686817800
	if ts.Seconds != 1686817800 {
		t.Errorf("unexpected seconds value: %d", ts.Seconds)
	}
}

// TestUint32MapToTerraformMap_Empty verifies that an empty (non-nil) map produces a non-null, empty Terraform map.
func TestUint32MapToTerraformMap_Empty(t *testing.T) {
	m, diags := Uint32MapToTerraformMap(map[uint32]string{})
	if diags != nil && diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if m.IsNull() {
		t.Fatalf("expected non-null map for empty (non-nil) input")
	}
	if len(m.Elements()) != 0 {
		t.Errorf("expected 0 elements, got %d", len(m.Elements()))
	}
}

// TestUint32MapToTerraformMap_KeyFormat verifies that uint32 keys are correctly stringified.
func TestUint32MapToTerraformMap_KeyFormat(t *testing.T) {
	in := map[uint32]string{0: "zero", 4294967295: "max"}
	m, diags := Uint32MapToTerraformMap(in)
	if diags != nil && diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	elems := m.Elements()
	if _, ok := elems["0"]; !ok {
		t.Errorf("expected key '0' in map, got %v", elems)
	}
	if _, ok := elems["4294967295"]; !ok {
		t.Errorf("expected key '4294967295' in map, got %v", elems)
	}
}

// TestGoStructToObjectValue_MismatchedAttributes verifies that a mismatched attribute map
// causes diagnostics errors to be returned rather than panicking.
func TestGoStructToObjectValue_MismatchedAttributes(t *testing.T) {
	type testStruct struct {
		A string `tfsdk:"a"`
	}
	// Provide an attribute map that does not match the struct fields.
	mismatchedAttrMap := map[string]attr.Type{"b": types.StringType}
	_, diags := GoStructToObjectValue(context.Background(), mismatchedAttrMap, testStruct{A: "foo"})
	if !diags.HasError() {
		t.Fatalf("expected diagnostics error for mismatched attribute map, got none")
	}
}

// TestGoStructSliceToListValue_Nil verifies that a nil slice produces a non-null, empty ListValue.
func TestGoStructSliceToListValue_Nil(t *testing.T) {
	type testStruct struct {
		A string `tfsdk:"a"`
	}
	attrMap := map[string]attr.Type{"a": types.StringType}
	lv, diags := GoStructSliceToListValue(context.Background(), attrMap, []testStruct(nil))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if lv.IsNull() {
		t.Fatalf("expected non-null list value for nil slice")
	}
	if len(lv.Elements()) != 0 {
		t.Errorf("expected 0 elements for nil slice, got %d", len(lv.Elements()))
	}
}

// TestGoMapToObjectMapValue_MismatchedAttributes verifies that a mismatched attribute map
// causes diagnostics errors to be returned.
func TestGoMapToObjectMapValue_MismatchedAttributes(t *testing.T) {
	type testStruct struct {
		A string `tfsdk:"a"`
	}
	// Provide an attribute map that does not match the struct fields.
	mismatchedAttrMap := map[string]attr.Type{"b": types.StringType}
	goMap := map[string]testStruct{"x": {A: "y"}}
	_, diags := GoMapToObjectMapValue(context.Background(), mismatchedAttrMap, goMap)
	if !diags.HasError() {
		t.Fatalf("expected diagnostics error for mismatched attribute map, got none")
	}
}

// TestTerraformMapToUint32Map verifies round-tripping of a well-formed uint32-keyed map.
func TestTerraformMapToUint32Map(t *testing.T) {
	src := map[uint32]string{1: "a", 200: "b", 4000000000: "c"}
	tfMap, diags := Uint32MapToTerraformMap(src)
	if diags.HasError() {
		t.Fatalf("Uint32MapToTerraformMap: unexpected diagnostics: %v", diags)
	}
	got, diags := TerraformMapToUint32Map(context.Background(), tfMap)
	if diags.HasError() {
		t.Fatalf("TerraformMapToUint32Map: unexpected diagnostics: %v", diags)
	}
	if len(got) != len(src) {
		t.Fatalf("size mismatch: got %d, want %d", len(got), len(src))
	}
	for k, v := range src {
		if got[k] != v {
			t.Errorf("key %d: got %q, want %q", k, got[k], v)
		}
	}
}

// TestTerraformMapToUint32Map_Null verifies that a null Terraform map returns (nil, no diagnostics).
func TestTerraformMapToUint32Map_Null(t *testing.T) {
	got, diags := TerraformMapToUint32Map(context.Background(), types.MapNull(types.StringType))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got != nil {
		t.Fatalf("expected nil for null input, got %v", got)
	}
}

// TestTerraformMapToUint32Map_Unknown verifies that an unknown Terraform map returns (nil, no diagnostics).
func TestTerraformMapToUint32Map_Unknown(t *testing.T) {
	got, diags := TerraformMapToUint32Map(context.Background(), types.MapUnknown(types.StringType))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got != nil {
		t.Fatalf("expected nil for unknown input, got %v", got)
	}
}

// TestTerraformMapToUint32Map_Empty verifies that an empty (non-null) Terraform map produces an empty result.
func TestTerraformMapToUint32Map_Empty(t *testing.T) {
	tfMap, diags := types.MapValue(types.StringType, map[string]attr.Value{})
	if diags.HasError() {
		t.Fatalf("MapValue: unexpected diagnostics: %v", diags)
	}
	got, diags := TerraformMapToUint32Map(context.Background(), tfMap)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %d elements", len(got))
	}
}

// TestTerraformMapToUint32Map_InvalidKey verifies that a non-numeric key produces a diagnostic error.
func TestTerraformMapToUint32Map_InvalidKey(t *testing.T) {
	tfMap, diags := types.MapValue(types.StringType, map[string]attr.Value{
		"not-a-number": types.StringValue("v"),
	})
	if diags.HasError() {
		t.Fatalf("MapValue: unexpected diagnostics: %v", diags)
	}
	_, diags = TerraformMapToUint32Map(context.Background(), tfMap)
	if !diags.HasError() {
		t.Fatalf("expected diagnostic error for non-numeric key")
	}
}

// TestTerraformMapToUint32Map_KeyOverflow verifies that a key exceeding uint32 range produces a diagnostic error.
func TestTerraformMapToUint32Map_KeyOverflow(t *testing.T) {
	tfMap, diags := types.MapValue(types.StringType, map[string]attr.Value{
		"4294967296": types.StringValue("v"), // math.MaxUint32 + 1
	})
	if diags.HasError() {
		t.Fatalf("MapValue: unexpected diagnostics: %v", diags)
	}
	_, diags = TerraformMapToUint32Map(context.Background(), tfMap)
	if !diags.HasError() {
		t.Fatalf("expected diagnostic error for out-of-range uint32 key")
	}
}

// TestValueContainsUnknown_TopLevel verifies that a top-level unknown value is detected.
func TestValueContainsUnknown_TopLevel(t *testing.T) {
	if !ValueContainsUnknown(types.StringUnknown()) {
		t.Fatal("expected top-level unknown string to be detected")
	}
}

// TestValueContainsUnknown_NestedObject verifies that nested unknown children are detected.
func TestValueContainsUnknown_NestedObject(t *testing.T) {
	child, childDiags := types.ObjectValue(
		map[string]attr.Type{"description": types.StringType},
		map[string]attr.Value{"description": types.StringUnknown()},
	)
	if childDiags.HasError() {
		t.Fatalf("ObjectValue child: unexpected diagnostics: %v", childDiags)
	}

	parent, parentDiags := types.ListValue(
		types.ObjectType{AttrTypes: map[string]attr.Type{"description": types.StringType}},
		[]attr.Value{child},
	)
	if parentDiags.HasError() {
		t.Fatalf("ListValue parent: unexpected diagnostics: %v", parentDiags)
	}

	if !ValueContainsUnknown(parent) {
		t.Fatal("expected nested unknown child to be detected")
	}
}

// TestValueContainsUnknown_KnownValue verifies that known and null children are treated as safe.
func TestValueContainsUnknown_KnownValue(t *testing.T) {
	obj, diags := types.ObjectValue(
		map[string]attr.Type{
			"description": types.StringType,
			"ttl":         types.Int64Type,
		},
		map[string]attr.Value{
			"description": types.StringNull(),
			"ttl":         types.Int64Value(3600),
		},
	)
	if diags.HasError() {
		t.Fatalf("ObjectValue: unexpected diagnostics: %v", diags)
	}

	if ValueContainsUnknown(obj) {
		t.Fatal("did not expect known/null object to contain unknown values")
	}
}

// TestResolveUnknowns_FastPathKnown verifies that if the plan value contains no
// unknown attributes, ResolveUnknowns returns it unmodified immediately.
func TestResolveUnknowns_FastPathKnown(t *testing.T) {
	planObj, diags := types.ObjectValue(
		map[string]attr.Type{
			"secret": types.StringType,
			"tenant": types.StringType,
		},
		map[string]attr.Value{
			"secret": types.StringValue("super-secret"),
			"tenant": types.StringValue("ves-io-known"),
		},
	)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics creating planObj: %v", diags)
	}

	apiObj, diags := types.ObjectValue(
		map[string]attr.Type{
			"secret": types.StringType,
			"tenant": types.StringType,
		},
		map[string]attr.Value{
			"secret": types.StringValue("redacted"),
			"tenant": types.StringValue("ves-io-api"),
		},
	)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics creating apiObj: %v", diags)
	}

	result := ResolveUnknowns(planObj, apiObj)
	if !result.Equal(planObj) {
		t.Fatalf("expected plan value to be preserved exactly in fast path, got %v", result)
	}
}

// TestResolveUnknowns_ObjectWithSensitiveAndUnknown verifies that an object with a
// known sensitive field and an unknown tenant field preserves the sensitive field
// while resolving the tenant from the API response.
func TestResolveUnknowns_ObjectWithSensitiveAndUnknown(t *testing.T) {
	attrTypes := map[string]attr.Type{
		"secret": types.StringType,
		"tenant": types.StringType,
		"count":  types.Int64Type,
	}

	planObj, diags := types.ObjectValue(
		attrTypes,
		map[string]attr.Value{
			"secret": types.StringValue("blindfold-or-md5-secret"),
			"tenant": types.StringUnknown(),
			"count":  types.Int64Value(42),
		},
	)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics creating planObj: %v", diags)
	}

	apiObj, diags := types.ObjectValue(
		attrTypes,
		map[string]attr.Value{
			"secret": types.StringNull(), // Backend redacts or omits secret in GET
			"tenant": types.StringValue("ves-io-tenant-resolved"),
			"count":  types.Int64Value(42),
		},
	)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics creating apiObj: %v", diags)
	}

	result := ResolveUnknowns(planObj, apiObj)
	if ValueContainsUnknown(result) {
		t.Fatalf("resolved object should not contain any unknown attributes, got %v", result)
	}

	attrs := result.Attributes()
	if !attrs["secret"].Equal(types.StringValue("blindfold-or-md5-secret")) {
		t.Errorf("expected secret to be preserved from plan, got %v", attrs["secret"])
	}
	if !attrs["tenant"].Equal(types.StringValue("ves-io-tenant-resolved")) {
		t.Errorf("expected tenant to be resolved from API, got %v", attrs["tenant"])
	}
	if !attrs["count"].Equal(types.Int64Value(42)) {
		t.Errorf("expected count to be preserved, got %v", attrs["count"])
	}
}

// TestResolveUnknowns_ListOfObjects verifies that a list of objects containing
// newly-added items with unknown fields has its unknown attributes resolved from the API.
func TestResolveUnknowns_ListOfObjects(t *testing.T) {
	elemType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":   types.StringType,
			"secret": types.StringType,
			"tenant": types.StringType,
		},
	}

	// Peer 0 is already known; Peer 1 is newly added during update with tenant unknown.
	peer0Plan, _ := types.ObjectValue(elemType.AttrTypes, map[string]attr.Value{
		"name":   types.StringValue("peer-0"),
		"secret": types.StringValue("secret-0"),
		"tenant": types.StringValue("ves-io-tenant-0"),
	})
	peer1Plan, _ := types.ObjectValue(elemType.AttrTypes, map[string]attr.Value{
		"name":   types.StringValue("peer-1"),
		"secret": types.StringValue("secret-1"),
		"tenant": types.StringUnknown(),
	})

	planList, diags := types.ListValue(elemType, []attr.Value{peer0Plan, peer1Plan})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics creating planList: %v", diags)
	}

	peer0API, _ := types.ObjectValue(elemType.AttrTypes, map[string]attr.Value{
		"name":   types.StringValue("peer-0"),
		"secret": types.StringNull(),
		"tenant": types.StringValue("ves-io-tenant-0"),
	})
	peer1API, _ := types.ObjectValue(elemType.AttrTypes, map[string]attr.Value{
		"name":   types.StringValue("peer-1"),
		"secret": types.StringNull(),
		"tenant": types.StringValue("ves-io-tenant-resolved"),
	})

	apiList, diags := types.ListValue(elemType, []attr.Value{peer0API, peer1API})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics creating apiList: %v", diags)
	}

	result := ResolveUnknowns(planList, apiList)
	if ValueContainsUnknown(result) {
		t.Fatalf("resolved list should not contain any unknowns, got %v", result)
	}

	elems := result.Elements()
	if len(elems) != 2 {
		t.Fatalf("expected 2 elements in resolved list, got %d", len(elems))
	}

	peer1Resolved := elems[1].(types.Object).Attributes()
	if !peer1Resolved["secret"].Equal(types.StringValue("secret-1")) {
		t.Errorf("expected peer-1 secret to be preserved from plan, got %v", peer1Resolved["secret"])
	}
	if !peer1Resolved["tenant"].Equal(types.StringValue("ves-io-tenant-resolved")) {
		t.Errorf("expected peer-1 tenant to be resolved from API, got %v", peer1Resolved["tenant"])
	}
}

// TestResolveUnknowns_MissingAPIElement verifies that when the API returns fewer
// elements than planned, unknown fields in extra elements fallback safely to typed null.
func TestResolveUnknowns_MissingAPIElement(t *testing.T) {
	elemType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":   types.StringType,
			"tenant": types.StringType,
		},
	}

	peerPlan, _ := types.ObjectValue(elemType.AttrTypes, map[string]attr.Value{
		"name":   types.StringValue("peer-orphan"),
		"tenant": types.StringUnknown(),
	})

	planList, _ := types.ListValue(elemType, []attr.Value{peerPlan})
	// API returned an empty list
	apiList, _ := types.ListValue(elemType, []attr.Value{})

	result := ResolveUnknowns(planList, apiList)
	if ValueContainsUnknown(result) {
		t.Fatalf("resolved list should not contain unknowns even if API omitted element, got %v", result)
	}

	elems := result.Elements()
	if len(elems) != 1 {
		t.Fatalf("expected 1 element, got %d", len(elems))
	}
	resolvedAttrs := elems[0].(types.Object).Attributes()
	if !resolvedAttrs["tenant"].Equal(types.StringNull()) {
		t.Errorf("expected tenant to fallback to StringNull(), got %v", resolvedAttrs["tenant"])
	}
}

// TestMergeRRSetListForApply_PreservesPlanShape verifies that backend-only API
// RRsets are excluded from the immediate post-apply state while matching planned
// RRsets still receive API-derived values.
func TestMergeRRSetListForApply_PreservesPlanShape(t *testing.T) {
	planned := testRRSetListValue(t,
		testARecordRRSet(t, "host.example.com", "192.168.1.10", types.StringUnknown()),
	)
	api := testRRSetListValue(t,
		testARecordRRSet(t, "host.example.com", "192.168.1.10", types.StringNull()),
		testNSRecordRRSet(t, "", "ns1crt.f5clouddns-test.com", types.StringNull()),
	)

	merged, diags := MergeRRSetListForApply(context.Background(), planned, api)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	want := testRRSetListValue(t,
		testARecordRRSet(t, "host.example.com", "192.168.1.10", types.StringNull()),
	)
	if !want.Equal(merged) {
		t.Fatalf("merged rrset list mismatch\nwant: %v\ngot:  %v", want, merged)
	}
}

// TestMergeRRSetListForApply_KnownPlanWins verifies that a fully-known planned
// RRSet list is returned unchanged.
func TestMergeRRSetListForApply_KnownPlanWins(t *testing.T) {
	planned := testRRSetListValue(t,
		testARecordRRSet(t, "host.example.com", "192.168.1.10", types.StringNull()),
	)
	api := testRRSetListValue(t,
		testARecordRRSet(t, "host.example.com", "192.168.1.10", types.StringNull()),
		testNSRecordRRSet(t, "", "ns1crt.f5clouddns-test.com", types.StringNull()),
	)

	merged, diags := MergeRRSetListForApply(context.Background(), planned, api)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !planned.Equal(merged) {
		t.Fatalf("expected known plan to win\nwant: %v\ngot:  %v", planned, merged)
	}
}

// TestMergeRRSetListForApply_NullPlanWins verifies that an unconfigured RRSet list
// stays null even when the API returns backend-managed RRsets.
func TestMergeRRSetListForApply_NullPlanWins(t *testing.T) {
	planned := types.ListNull(types.ObjectType{AttrTypes: testRRSetAttrTypes()})
	api := testRRSetListValue(t,
		testNSRecordRRSet(t, "", "ns1crt.f5clouddns-test.com", types.StringNull()),
	)

	merged, diags := MergeRRSetListForApply(context.Background(), planned, api)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !planned.Equal(merged) {
		t.Fatalf("expected null plan to win\nwant: %v\ngot:  %v", planned, merged)
	}
}

// TestMergeRRSetListForRead_PreservesPriorShape verifies that refresh keeps only
// prior-state RRSet keys while still using matching API values.
func TestMergeRRSetListForRead_PreservesPriorShape(t *testing.T) {
	prior := testRRSetListValue(t,
		testARecordRRSet(t, "host.example.com", "192.168.1.10", types.StringNull()),
	)
	api := testRRSetListValue(t,
		testARecordRRSet(t, "host.example.com", "192.168.1.10", types.StringNull()),
		testNSRecordRRSet(t, "", "ns1crt.f5clouddns-test.com", types.StringNull()),
	)

	merged, diags := MergeRRSetListForRead(context.Background(), prior, api)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !prior.Equal(merged) {
		t.Fatalf("expected prior-state shape to win\nwant: %v\ngot:  %v", prior, merged)
	}
}

// TestMergeRRSetListForRead_NullPriorWins verifies that an unconfigured prior state
// remains null during refresh even if the API returns backend-managed RRsets.
func TestMergeRRSetListForRead_NullPriorWins(t *testing.T) {
	prior := types.ListNull(types.ObjectType{AttrTypes: testRRSetAttrTypes()})
	api := testRRSetListValue(t,
		testNSRecordRRSet(t, "", "ns1crt.f5clouddns-test.com", types.StringNull()),
	)

	merged, diags := MergeRRSetListForRead(context.Background(), prior, api)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !prior.Equal(merged) {
		t.Fatalf("expected null prior state to win\nwant: %v\ngot:  %v", prior, merged)
	}
}

// TestMergeRRSetListForApply_UnknownPlanExcludesDefault verifies that when the
// planned list is top-level unknown, the API list is returned with backend
// default RRsets removed.
func TestMergeRRSetListForApply_UnknownPlanExcludesDefault(t *testing.T) {
	planned := types.ListUnknown(types.ObjectType{AttrTypes: testRRSetAttrTypes()})
	api := testRRSetListValue(t,
		testARecordRRSet(t, "host.example.com", "192.168.1.10", types.StringNull()),
		testNSRecordRRSet(t, "", "ns1crt.f5clouddns-test.com", types.StringNull()),
	)

	merged, diags := MergeRRSetListForApply(context.Background(), planned, api)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	want := testRRSetListValue(t,
		testARecordRRSet(t, "host.example.com", "192.168.1.10", types.StringNull()),
	)
	if !want.Equal(merged) {
		t.Fatalf("expected default NS excluded from unknown-plan apply\nwant: %v\ngot:  %v", want, merged)
	}
}

// TestMergeRRSetListForApply_MultipleDefaultNSExcludesDuplicates verifies that when
// the API response contains multiple default NS records (which share the same empty name
// and thus produce identical index keys), MergeRRSetListForApply filters them out before
// indexing so that duplicate-key collisions do not abort apply-time reconciliation.
func TestMergeRRSetListForApply_MultipleDefaultNSExcludesDuplicates(t *testing.T) {
	planned := testRRSetListValue(t,
		testARecordRRSet(t, "host.example.com", "192.168.1.10", types.StringUnknown()),
	)
	api := testRRSetListValue(t,
		testARecordRRSet(t, "host.example.com", "192.168.1.10", types.StringNull()),
		testNSRecordRRSet(t, "", "ns1crt.f5clouddns-test.com", types.StringNull()),
		testNSRecordRRSet(t, "", "ns2crt.f5clouddns-test.com", types.StringNull()),
	)

	merged, diags := MergeRRSetListForApply(context.Background(), planned, api)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	want := testRRSetListValue(t,
		testARecordRRSet(t, "host.example.com", "192.168.1.10", types.StringNull()),
	)
	if !want.Equal(merged) {
		t.Fatalf("merged rrset list mismatch\nwant: %v\ngot:  %v", want, merged)
	}
}

// TestMergeRRSetListForRead_FirstReadExcludesDefault verifies the default NS RRSet
// is excluded when there is no prior state.
func TestMergeRRSetListForRead_FirstReadExcludesDefault(t *testing.T) {
	prior := types.ListNull(types.ObjectType{AttrTypes: testRRSetAttrTypes()})
	api := testRRSetListValue(t,
		testARecordRRSet(t, "host.example.com", "192.168.1.10", types.StringNull()),
		testNSRecordRRSet(t, "", "ns1crt.f5clouddns-test.com", types.StringNull()),
	)

	merged, diags := MergeRRSetListForRead(context.Background(), prior, api)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	want := testRRSetListValue(t,
		testARecordRRSet(t, "host.example.com", "192.168.1.10", types.StringNull()),
	)
	if !want.Equal(merged) {
		t.Fatalf("expected default NS excluded on first read\nwant: %v\ngot:  %v", want, merged)
	}
}

// TestMergeRRSetListForRead_MultiValueDefaultExcluded verifies a multi-value NS
// default (matched order-insensitively) is excluded.
func TestMergeRRSetListForRead_MultiValueDefaultExcluded(t *testing.T) {
	prior := types.ListNull(types.ObjectType{AttrTypes: testRRSetAttrTypes()})
	api := testRRSetListValue(t,
		testARecordRRSet(t, "host.example.com", "192.168.1.10", types.StringNull()),
		testNSRecordRRSetMulti(t, "", types.StringNull(), "ns2.f5clouddns.com", "ns1.f5clouddns.com"),
	)

	merged, diags := MergeRRSetListForRead(context.Background(), prior, api)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	want := testRRSetListValue(t,
		testARecordRRSet(t, "host.example.com", "192.168.1.10", types.StringNull()),
	)
	if !want.Equal(merged) {
		t.Fatalf("expected multi-value default NS excluded\nwant: %v\ngot:  %v", want, merged)
	}
}

// TestMergeRRSetListForRead_DeletionDetected verifies a prior record that no
// longer exists in the API is dropped from state so deletion surfaces as drift.
func TestMergeRRSetListForRead_DeletionDetected(t *testing.T) {
	prior := testRRSetListValue(t,
		testARecordRRSet(t, "host.example.com", "192.168.1.10", types.StringNull()),
	)
	api := testRRSetListValue(t,
		testNSRecordRRSet(t, "", "ns1crt.f5clouddns-test.com", types.StringNull()),
	)

	merged, diags := MergeRRSetListForRead(context.Background(), prior, api)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	want := types.ListNull(types.ObjectType{AttrTypes: testRRSetAttrTypes()})
	if !want.Equal(merged) {
		t.Fatalf("expected deleted record dropped (null list)\nwant: %v\ngot:  %v", want, merged)
	}
}

// TestMergeRRSetListForRead_NonDefaultAdditionAppended verifies an unexpected
// non-default backend record is surfaced in state as drift.
func TestMergeRRSetListForRead_NonDefaultAdditionAppended(t *testing.T) {
	prior := testRRSetListValue(t,
		testARecordRRSet(t, "host.example.com", "192.168.1.10", types.StringNull()),
	)
	api := testRRSetListValue(t,
		testARecordRRSet(t, "host.example.com", "192.168.1.10", types.StringNull()),
		testARecordRRSet(t, "extra.example.com", "10.0.0.1", types.StringNull()),
		testNSRecordRRSet(t, "", "ns1crt.f5clouddns-test.com", types.StringNull()),
	)

	merged, diags := MergeRRSetListForRead(context.Background(), prior, api)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	want := testRRSetListValue(t,
		testARecordRRSet(t, "host.example.com", "192.168.1.10", types.StringNull()),
		testARecordRRSet(t, "extra.example.com", "10.0.0.1", types.StringNull()),
	)
	if !want.Equal(merged) {
		t.Fatalf("expected non-default addition appended\nwant: %v\ngot:  %v", want, merged)
	}
}

// TestMergeRRSetListForRead_ValueChangeUsesAPI verifies an out-of-band value
// change on a declared record is reflected in state (so it shows as drift).
func TestMergeRRSetListForRead_ValueChangeUsesAPI(t *testing.T) {
	prior := testRRSetListValue(t,
		testARecordRRSet(t, "host.example.com", "192.168.1.10", types.StringNull()),
	)
	api := testRRSetListValue(t,
		testARecordRRSet(t, "host.example.com", "10.20.30.40", types.StringNull()),
		testNSRecordRRSet(t, "", "ns1crt.f5clouddns-test.com", types.StringNull()),
	)

	merged, diags := MergeRRSetListForRead(context.Background(), prior, api)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	want := testRRSetListValue(t,
		testARecordRRSet(t, "host.example.com", "10.20.30.40", types.StringNull()),
	)
	if !want.Equal(merged) {
		t.Fatalf("expected API value change reflected\nwant: %v\ngot:  %v", want, merged)
	}
}

// TestIsBackendDefaultRRSet verifies default recognition matches only the exact
// backend default NS records.
func TestIsBackendDefaultRRSet(t *testing.T) {
	if !isBackendDefaultRRSet(testNSRecordRRSet(t, "", "ns1crt.f5clouddns-test.com", types.StringNull())) {
		t.Fatalf("expected known default NS to be recognised")
	}
	if !isBackendDefaultRRSet(testNSRecordRRSetMulti(t, "", types.StringValue("ignored"), "ns2.f5clouddns.com", "ns1.f5clouddns.com")) {
		t.Fatalf("expected multi-value default NS to be recognised (description ignored)")
	}
	if isBackendDefaultRRSet(testARecordRRSet(t, "host.example.com", "192.168.1.10", types.StringNull())) {
		t.Fatalf("A record must not be treated as a default")
	}
	if isBackendDefaultRRSet(testNSRecordRRSet(t, "", "ns-not-a-default.example.com", types.StringNull())) {
		t.Fatalf("NS with non-default values must not be treated as a default")
	}
	if isBackendDefaultRRSet(testNSRecordRRSet(t, "sub", "ns1crt.f5clouddns-test.com", types.StringNull())) {
		t.Fatalf("NS with a non-apex name must not be treated as a default")
	}
}

// testRRSetListValue creates a typed RRSet list used by apply-time merge tests.
func testRRSetListValue(t *testing.T, rrsets ...types.Object) types.List {
	t.Helper()

	elements := make([]attr.Value, 0, len(rrsets))
	for _, rrset := range rrsets {
		elements = append(elements, rrset)
	}

	listValue, diags := types.ListValue(types.ObjectType{AttrTypes: testRRSetAttrTypes()}, elements)
	if diags.HasError() {
		t.Fatalf("ListValue: unexpected diagnostics: %v", diags)
	}

	return listValue
}

// testARecordRRSet creates a minimal A record RRSet object for merge tests.
func testARecordRRSet(t *testing.T, name, value string, description types.String) types.Object {
	t.Helper()

	return testRRSetObject(t, map[string]attr.Value{
		"description": description,
		"ttl":         types.Int64Value(3600),
		"type_record_set_choice": testRRSetTypeChoiceObject(t, map[string]attr.Value{
			"a_record": testRRSetBranchObject(t, testARecordAttrTypes(), map[string]attr.Value{
				"name":   types.StringValue(name),
				"values": testStringListValue(t, value),
			}),
			"ns_record": types.ObjectNull(testNSRecordAttrTypes()),
		}),
	})
}

// testNSRecordRRSet creates a minimal NS record RRSet object for merge tests.
func testNSRecordRRSet(t *testing.T, name, value string, description types.String) types.Object {
	t.Helper()

	return testRRSetObject(t, map[string]attr.Value{
		"description": description,
		"ttl":         types.Int64Value(86400),
		"type_record_set_choice": testRRSetTypeChoiceObject(t, map[string]attr.Value{
			"a_record": types.ObjectNull(testARecordAttrTypes()),
			"ns_record": testRRSetBranchObject(t, testNSRecordAttrTypes(), map[string]attr.Value{
				"name":   types.StringValue(name),
				"values": testStringListValue(t, value),
			}),
		}),
	})
}

// testNSRecordRRSetMulti creates an NS record RRSet object with multiple values.
func testNSRecordRRSetMulti(t *testing.T, name string, description types.String, values ...string) types.Object {
	t.Helper()

	return testRRSetObject(t, map[string]attr.Value{
		"description": description,
		"ttl":         types.Int64Value(86400),
		"type_record_set_choice": testRRSetTypeChoiceObject(t, map[string]attr.Value{
			"a_record": types.ObjectNull(testARecordAttrTypes()),
			"ns_record": testRRSetBranchObject(t, testNSRecordAttrTypes(), map[string]attr.Value{
				"name":   types.StringValue(name),
				"values": testStringListValue(t, values...),
			}),
		}),
	})
}

// testRRSetObject creates an RRSet object value for merge tests.
func testRRSetObject(t *testing.T, attrs map[string]attr.Value) types.Object {
	t.Helper()

	obj, diags := types.ObjectValue(testRRSetAttrTypes(), attrs)
	if diags.HasError() {
		t.Fatalf("ObjectValue rrset: unexpected diagnostics: %v", diags)
	}

	return obj
}

// testRRSetTypeChoiceObject creates a minimal type_record_set_choice object.
func testRRSetTypeChoiceObject(t *testing.T, attrs map[string]attr.Value) types.Object {
	t.Helper()

	obj, diags := types.ObjectValue(testRRSetTypeChoiceAttrTypes(), attrs)
	if diags.HasError() {
		t.Fatalf("ObjectValue type choice: unexpected diagnostics: %v", diags)
	}

	return obj
}

// testRRSetBranchObject creates a typed branch object for the RRSet oneof.
func testRRSetBranchObject(t *testing.T, attrTypes map[string]attr.Type, attrs map[string]attr.Value) types.Object {
	t.Helper()

	obj, diags := types.ObjectValue(attrTypes, attrs)
	if diags.HasError() {
		t.Fatalf("ObjectValue branch: unexpected diagnostics: %v", diags)
	}

	return obj
}

// testStringListValue creates a typed string list for RRSet branch values.
func testStringListValue(t *testing.T, values ...string) types.List {
	t.Helper()

	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.StringValue(value))
	}

	listValue, diags := types.ListValue(types.StringType, elements)
	if diags.HasError() {
		t.Fatalf("ListValue string: unexpected diagnostics: %v", diags)
	}

	return listValue
}

// testRRSetAttrTypes returns the RRSet object attribute map used in merge tests.
func testRRSetAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"description":            types.StringType,
		"ttl":                    types.Int64Type,
		"type_record_set_choice": types.ObjectType{AttrTypes: testRRSetTypeChoiceAttrTypes()},
	}
}

// testRRSetTypeChoiceAttrTypes returns the oneof object attribute map used in merge tests.
func testRRSetTypeChoiceAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"a_record":  types.ObjectType{AttrTypes: testARecordAttrTypes()},
		"ns_record": types.ObjectType{AttrTypes: testNSRecordAttrTypes()},
	}
}

// testARecordAttrTypes returns the A-record branch attribute map used in merge tests.
func testARecordAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":   types.StringType,
		"values": types.ListType{ElemType: types.StringType},
	}
}

// testNSRecordAttrTypes returns the NS-record branch attribute map used in merge tests.
func testNSRecordAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":   types.StringType,
		"values": types.ListType{ElemType: types.StringType},
	}
}

func TestSanitizeAPIError(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		action     string
		resource   string
		err        error
		wantSubstr []string
		notSubstr  []string
	}{
		{
			name:     "5xx drops internal details and returns generic message",
			action:   "create",
			resource: "securemesh_site_v2",
			err: errors.New(
				`Creating object: Unsuccessful POST at URL https://x.volterra.us/api/public/namespaces/system/securemesh_site_v2s, status code 500, ` +
					`body {"code":13,"details":[{"code":"UNKNOWN","details":"Filling oneof default choice: oneof container field list _protoutil.NilField is not _protoutil.PtrListField","timestamp":"2025-07-06T20:20:59Z"}],"message":"There was a server error in processing request"}`,
			),
			wantSubstr: []string{"Unable to create securemesh_site_v2", "internal server error", "HTTP 500"},
			notSubstr:  []string{"_protoutil", "NilField", "PtrListField", "There was a server error"},
		},
		{
			name:     "4xx surfaces message and code but drops details[].details",
			action:   "update",
			resource: "http_loadbalancer",
			err: errors.New(
				`Updating object: Unsuccessful PUT at URL https://x.volterra.us/api/config/foo, status code 400, ` +
					`body {"code":3,"details":[{"code":"INVALID_ARGUMENT","details":"internal.pkg.Foo something"}],"message":"invalid domain: not FQDN"}`,
			),
			wantSubstr: []string{"Unable to update http_loadbalancer", "HTTP 400", "code 3", "invalid domain: not FQDN"},
			notSubstr:  []string{"internal.pkg.Foo", "INVALID_ARGUMENT"},
		},
		{
			name:       "unparsable error falls back to raw text",
			action:     "read",
			resource:   "foo",
			err:        errors.New("connection refused"),
			wantSubstr: []string{"Unable to read foo", "connection refused"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizeAPIError(ctx, tc.action, tc.resource, tc.err)
			for _, s := range tc.wantSubstr {
				if !strings.Contains(got, s) {
					t.Errorf("expected substring %q in %q", s, got)
				}
			}
			for _, s := range tc.notSubstr {
				if strings.Contains(got, s) {
					t.Errorf("did NOT expect substring %q in %q", s, got)
				}
			}
		})
	}
}

// testRRSetGroupMetaAttrTypes returns the attribute type map for an RRSetGroup's metadata.
//
// Why: Provides typed schema definitions for group metadata (name, description) in tests.
//
// How: Constructs a map with name and description StringTypes.
//
// Returns:
//   - map[string]attr.Type: schema attribute types.
func testRRSetGroupMetaAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":        types.StringType,
		"description": types.StringType,
	}
}

// testLBRecordValueAttrTypes returns the attribute type map for an lb_record's value ref.
//
// Why: Models the ObjectRefType structure (name, namespace, tenant) in tests.
//
// How: Constructs a map with name, namespace, and tenant StringTypes.
//
// Returns:
//   - map[string]attr.Type: schema attribute types.
func testLBRecordValueAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":      types.StringType,
		"namespace": types.StringType,
		"tenant":    types.StringType,
	}
}

// testLBRecordAttrTypes returns the attribute type map for an lb_record choice branch.
//
// Why: Models the lb_record object structure containing name and value.
//
// How: Constructs a map with name StringType and value ObjectType.
//
// Returns:
//   - map[string]attr.Type: schema attribute types.
func testLBRecordAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":  types.StringType,
		"value": types.ObjectType{AttrTypes: testLBRecordValueAttrTypes()},
	}
}

// testRRSetTypeChoiceWithLBAttrTypes returns the oneof type choice attribute map including lb_record.
//
// Why: Enables testing lb_record resolution alongside a_record and ns_record.
//
// How: Constructs a map with a_record, ns_record, and lb_record ObjectTypes.
//
// Returns:
//   - map[string]attr.Type: schema attribute types.
func testRRSetTypeChoiceWithLBAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"a_record":  types.ObjectType{AttrTypes: testARecordAttrTypes()},
		"ns_record": types.ObjectType{AttrTypes: testNSRecordAttrTypes()},
		"lb_record": types.ObjectType{AttrTypes: testLBRecordAttrTypes()},
	}
}

// testRRSetWithLBAttrTypes returns the RRSet object attribute map including lb_record in choice.
//
// Why: Defines full RRSet schema supporting lb_record branches in tests.
//
// How: Constructs a map with description, ttl, and type_record_set_choice ObjectType.
//
// Returns:
//   - map[string]attr.Type: schema attribute types.
func testRRSetWithLBAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"description":            types.StringType,
		"ttl":                    types.Int64Type,
		"type_record_set_choice": types.ObjectType{AttrTypes: testRRSetTypeChoiceWithLBAttrTypes()},
	}
}

// testRRSetGroupWithLBAttrTypes returns the RRSetGroup object attribute map.
//
// Why: Defines the nested RRSetGroup schema containing metadata and child rr_set list.
//
// How: Constructs a map with metadata ObjectType and rr_set ListType.
//
// Returns:
//   - map[string]attr.Type: schema attribute types.
func testRRSetGroupWithLBAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"metadata": types.ObjectType{AttrTypes: testRRSetGroupMetaAttrTypes()},
		"rr_set":   types.ListType{ElemType: types.ObjectType{AttrTypes: testRRSetWithLBAttrTypes()}},
	}
}

// testLBRecordRRSet creates an RRSet object containing an lb_record choice branch.
//
// Why: Synthesizes test fixtures for f5xc_dns_zone rr_set_group lb_record tests.
//
// How: Builds nested ObjectValues for lb_record value, choice, and enclosing RRSet.
//
// Parameters:
//   - t: testing context.
//   - recordName: record name inside lb_record.
//   - lbName: referenced load balancer name.
//   - lbNamespace: referenced load balancer namespace.
//   - tenant: candidate tenant string (may be unknown or concrete).
//   - description: record description string (may be unknown, null, or concrete).
//
// Returns:
//   - types.Object: synthesized RRSet object value.
func testLBRecordRRSet(t *testing.T, recordName, lbName, lbNamespace string, tenant, description types.String) types.Object {
	t.Helper()

	choiceAttrs := map[string]attr.Value{
		"a_record":  types.ObjectNull(testARecordAttrTypes()),
		"ns_record": types.ObjectNull(testNSRecordAttrTypes()),
		"lb_record": testRRSetBranchObject(t, testLBRecordAttrTypes(), map[string]attr.Value{
			"name": types.StringValue(recordName),
			"value": testRRSetBranchObject(t, testLBRecordValueAttrTypes(), map[string]attr.Value{
				"name":      types.StringValue(lbName),
				"namespace": types.StringValue(lbNamespace),
				"tenant":    tenant,
			}),
		}),
	}

	choiceObj, diags := types.ObjectValue(testRRSetTypeChoiceWithLBAttrTypes(), choiceAttrs)
	require.False(t, diags.HasError())

	rrsetAttrs := map[string]attr.Value{
		"description":            description,
		"ttl":                    types.Int64Value(63),
		"type_record_set_choice": choiceObj,
	}

	obj, diags := types.ObjectValue(testRRSetWithLBAttrTypes(), rrsetAttrs)
	require.False(t, diags.HasError())
	return obj
}

// testRRSetGroupObject creates an RRSetGroup object containing metadata and child record sets.
//
// Why: Synthesizes test fixtures representing named record groups in f5xc_dns_zone.
//
// How: Builds group metadata object and typed list of RRSet objects.
//
// Parameters:
//   - t: testing context.
//   - groupName: unique name for metadata.name.
//   - rrsets: child RRSet objects to embed in the group.
//
// Returns:
//   - types.Object: synthesized RRSetGroup object value.
func testRRSetGroupObject(t *testing.T, groupName string, rrsets ...types.Object) types.Object {
	t.Helper()

	metaObj, diags := types.ObjectValue(testRRSetGroupMetaAttrTypes(), map[string]attr.Value{
		"name":        types.StringValue(groupName),
		"description": types.StringNull(),
	})
	require.False(t, diags.HasError())

	rrsetElemType := types.ObjectType{AttrTypes: testRRSetWithLBAttrTypes()}
	rrsetElements := make([]attr.Value, len(rrsets))
	for i, r := range rrsets {
		rrsetElements[i] = r
	}
	rrsetList, diags := types.ListValue(rrsetElemType, rrsetElements)
	require.False(t, diags.HasError())

	groupObj, diags := types.ObjectValue(testRRSetGroupWithLBAttrTypes(), map[string]attr.Value{
		"metadata": metaObj,
		"rr_set":   rrsetList,
	})
	require.False(t, diags.HasError())
	return groupObj
}

// TestMergeRRSetListForApply_RRSetGroup_ResolvesNestedUnknowns verifies that when
// planned rr_set_group elements contain nested unknown values (such as computed
// description and lb_record.value.tenant), MergeRRSetListForApply populates concrete
// values from matching API groups and guarantees zero unknown values remain.
func TestMergeRRSetListForApply_RRSetGroup_ResolvesNestedUnknowns(t *testing.T) {
	ctx := context.Background()

	// 1. Build planned group where description and lb_record.value.tenant are unknown
	plannedRecord := testLBRecordRRSet(t, "grouped", "repro-lb", "system", types.StringUnknown(), types.StringUnknown())
	plannedGroup := testRRSetGroupObject(t, "group1", plannedRecord)
	elemType := types.ObjectType{AttrTypes: testRRSetGroupWithLBAttrTypes()}
	plannedList, diags := types.ListValue(elemType, []attr.Value{plannedGroup})
	require.False(t, diags.HasError())
	require.True(t, ValueContainsUnknown(plannedList))

	// 2. Build API group where description is null and tenant is resolved
	apiRecord := testLBRecordRRSet(t, "grouped", "repro-lb", "system", types.StringValue("automation-rllmbbuf"), types.StringNull())
	apiGroup := testRRSetGroupObject(t, "group1", apiRecord)
	apiList, diags := types.ListValue(elemType, []attr.Value{apiGroup})
	require.False(t, diags.HasError())
	require.False(t, ValueContainsUnknown(apiList))

	// 3. Execute apply merge
	result, diags := MergeRRSetListForApply(ctx, plannedList, apiList)
	require.False(t, diags.HasError())

	// 4. Invariant: zero unknowns in result
	assert.False(t, ValueContainsUnknown(result), "Result list must not contain any unknown values")

	// 5. Verify concrete field values
	require.Len(t, result.Elements(), 1)
	resultGroup := result.Elements()[0].(types.Object)
	groupAttrs := resultGroup.Attributes()

	rrSetList := groupAttrs["rr_set"].(types.List)
	require.Len(t, rrSetList.Elements(), 1)
	resultRecord := rrSetList.Elements()[0].(types.Object)
	recAttrs := resultRecord.Attributes()

	// description must be null
	assert.True(t, recAttrs["description"].IsNull(), "Omitted description must resolve to null")

	// tenant must be "automation-rllmbbuf"
	choiceObj := recAttrs["type_record_set_choice"].(types.Object)
	lbObj := choiceObj.Attributes()["lb_record"].(types.Object)
	valObj := lbObj.Attributes()["value"].(types.Object)
	tenantVal := valObj.Attributes()["tenant"].(types.String)
	assert.Equal(t, "automation-rllmbbuf", tenantVal.ValueString(), "Tenant must be populated from API")
}

// TestMergeRRSetListForApply_UnkeyableFallbackResolvesUnknowns verifies the defensive
// invariant that if a planned list containing unknowns cannot be indexed by RRSet or
// RRSetGroup keying, MergeRRSetListForApply falls back to ResolveUnknowns rather than
// returning raw planned values with unknowns.
func TestMergeRRSetListForApply_UnkeyableFallbackResolvesUnknowns(t *testing.T) {
	ctx := context.Background()

	dummyAttrTypes := map[string]attr.Type{
		"custom_field": types.StringType,
	}
	dummyElemType := types.ObjectType{AttrTypes: dummyAttrTypes}

	plannedObj, diags := types.ObjectValue(dummyAttrTypes, map[string]attr.Value{
		"custom_field": types.StringUnknown(),
	})
	require.False(t, diags.HasError())
	plannedList, diags := types.ListValue(dummyElemType, []attr.Value{plannedObj})
	require.False(t, diags.HasError())
	require.True(t, ValueContainsUnknown(plannedList))

	apiObj, diags := types.ObjectValue(dummyAttrTypes, map[string]attr.Value{
		"custom_field": types.StringValue("resolved-value"),
	})
	require.False(t, diags.HasError())
	apiList, diags := types.ListValue(dummyElemType, []attr.Value{apiObj})
	require.False(t, diags.HasError())

	result, diags := MergeRRSetListForApply(ctx, plannedList, apiList)
	require.False(t, diags.HasError())
	assert.False(t, ValueContainsUnknown(result), "Fallback must guarantee zero unknown values")

	resObj := result.Elements()[0].(types.Object)
	assert.Equal(t, "resolved-value", resObj.Attributes()["custom_field"].(types.String).ValueString())
}

// TestMergeRRSetListForRead_RRSetGroup_ReconcilesDrift verifies that MergeRRSetListForRead
// preserves prior group ordering, surfaces out-of-band group additions, drops deleted
// groups, and updates child record values.
func TestMergeRRSetListForRead_RRSetGroup_ReconcilesDrift(t *testing.T) {
	ctx := context.Background()
	elemType := types.ObjectType{AttrTypes: testRRSetGroupWithLBAttrTypes()}

	// Prior: group1 (rec1 with tenant="t1"), group2 (rec2)
	priorGroup1 := testRRSetGroupObject(t, "group1", testLBRecordRRSet(t, "rec1", "lb1", "sys", types.StringValue("t1"), types.StringNull()))
	priorGroup2 := testRRSetGroupObject(t, "group2", testLBRecordRRSet(t, "rec2", "lb2", "sys", types.StringValue("t2"), types.StringNull()))
	priorList, diags := types.ListValue(elemType, []attr.Value{priorGroup1, priorGroup2})
	require.False(t, diags.HasError())

	// API: group1 (rec1 modified with tenant="t1-updated"), group2 deleted, group3 added
	apiGroup1 := testRRSetGroupObject(t, "group1", testLBRecordRRSet(t, "rec1", "lb1", "sys", types.StringValue("t1-updated"), types.StringNull()))
	apiGroup3 := testRRSetGroupObject(t, "group3", testLBRecordRRSet(t, "rec3", "lb3", "sys", types.StringValue("t3"), types.StringNull()))
	apiList, diags := types.ListValue(elemType, []attr.Value{apiGroup1, apiGroup3})
	require.False(t, diags.HasError())

	result, diags := MergeRRSetListForRead(ctx, priorList, apiList)
	require.False(t, diags.HasError())

	// Result must contain group1 and group3 (group2 dropped)
	require.Len(t, result.Elements(), 2)

	// group1 is first (prior order preserved), with updated tenant
	resGroup1 := result.Elements()[0].(types.Object)
	group1Name, ok := rrsetGroupAttrValueKey(resGroup1)
	require.True(t, ok)
	assert.Equal(t, "group1", group1Name)

	rrsetList1 := resGroup1.Attributes()["rr_set"].(types.List)
	rec1 := rrsetList1.Elements()[0].(types.Object)
	choice1 := rec1.Attributes()["type_record_set_choice"].(types.Object)
	lb1 := choice1.Attributes()["lb_record"].(types.Object)
	val1 := lb1.Attributes()["value"].(types.Object)
	assert.Equal(t, "t1-updated", val1.Attributes()["tenant"].(types.String).ValueString())

	// group3 is appended
	resGroup3 := result.Elements()[1].(types.Object)
	group3Name, ok := rrsetGroupAttrValueKey(resGroup3)
	require.True(t, ok)
	assert.Equal(t, "group3", group3Name)
}

// testUnkeyableRRSetGroupObject creates an RRSetGroup object with a null metadata object,
// making it unkeyable by rrsetGroupAttrValueKey.
func testUnkeyableRRSetGroupObject(t *testing.T) types.Object {
	t.Helper()
	metaObj := types.ObjectNull(testRRSetGroupMetaAttrTypes())
	rrsetElemType := types.ObjectType{AttrTypes: testRRSetWithLBAttrTypes()}
	rrsetList, diags := types.ListValue(rrsetElemType, []attr.Value{})
	require.False(t, diags.HasError())

	groupObj, diags := types.ObjectValue(testRRSetGroupWithLBAttrTypes(), map[string]attr.Value{
		"metadata": metaObj,
		"rr_set":   rrsetList,
	})
	require.False(t, diags.HasError())
	return groupObj
}

// TestMergeRRSetListForRead_RRSetGroup_NullPriorNormalizesEmptyList verifies that when
// prior state is null or unknown, an empty API list is normalized to types.ListNull
// rather than an empty list, keeping behavior consistent with flat RRSet reconciliation.
func TestMergeRRSetListForRead_RRSetGroup_NullPriorNormalizesEmptyList(t *testing.T) {
	ctx := context.Background()
	elemType := types.ObjectType{AttrTypes: testRRSetGroupWithLBAttrTypes()}

	// Case 1: Prior is null, API is empty -> result is ListNull
	priorNull := types.ListNull(elemType)
	apiEmpty, diags := types.ListValue(elemType, []attr.Value{})
	require.False(t, diags.HasError())

	result, diags := MergeRRSetListForRead(ctx, priorNull, apiEmpty)
	require.False(t, diags.HasError())
	assert.True(t, result.IsNull(), "Empty API list with null prior must normalize to null list")

	// Case 2: Prior is unknown, API is empty -> result is ListNull
	priorUnknown := types.ListUnknown(elemType)
	result, diags = MergeRRSetListForRead(ctx, priorUnknown, apiEmpty)
	require.False(t, diags.HasError())
	assert.True(t, result.IsNull(), "Empty API list with unknown prior must normalize to null list")

	// Case 3: Prior is null, API has elements -> result is non-null with elements
	apiGroup := testRRSetGroupObject(t, "group1", testLBRecordRRSet(t, "rec1", "lb1", "sys", types.StringValue("t1"), types.StringNull()))
	apiWithElems, diags := types.ListValue(elemType, []attr.Value{apiGroup})
	require.False(t, diags.HasError())

	result, diags = MergeRRSetListForRead(ctx, priorNull, apiWithElems)
	require.False(t, diags.HasError())
	assert.False(t, result.IsNull(), "Non-empty API list must not be null")
	require.Len(t, result.Elements(), 1)
}

// TestMergeRRSetListForRead_RRSetGroup_UnkeyableOrDuplicateFallback verifies that if API
// elements are unkeyable or contain duplicate keys, or if prior elements are unkeyable,
// MergeRRSetListForRead safely falls back to returning the API list rather than dropping
// elements or crashing.
func TestMergeRRSetListForRead_RRSetGroup_UnkeyableOrDuplicateFallback(t *testing.T) {
	ctx := context.Background()
	elemType := types.ObjectType{AttrTypes: testRRSetGroupWithLBAttrTypes()}

	validGroup1 := testRRSetGroupObject(t, "group1", testLBRecordRRSet(t, "rec1", "lb1", "sys", types.StringValue("t1"), types.StringNull()))
	validGroup2 := testRRSetGroupObject(t, "group2", testLBRecordRRSet(t, "rec2", "lb2", "sys", types.StringValue("t2"), types.StringNull()))
	unkeyableGroup := testUnkeyableRRSetGroupObject(t)

	priorList, diags := types.ListValue(elemType, []attr.Value{validGroup1})
	require.False(t, diags.HasError())

	// Subtest 1: Duplicate group names in API -> safe fallback to API list
	t.Run("duplicate-api-keys", func(t *testing.T) {
		dupGroup1 := testRRSetGroupObject(t, "group1", testLBRecordRRSet(t, "rec1-dup", "lb1", "sys", types.StringValue("t1-dup"), types.StringNull()))
		apiDupList, diags := types.ListValue(elemType, []attr.Value{validGroup1, dupGroup1})
		require.False(t, diags.HasError())

		result, diags := MergeRRSetListForRead(ctx, priorList, apiDupList)
		require.False(t, diags.HasError())
		require.Len(t, result.Elements(), 2, "Fallback must preserve all API elements without dropping duplicates")
	})

	// Subtest 2: Unkeyable element in API -> safe fallback to API list
	t.Run("unkeyable-api-element", func(t *testing.T) {
		apiUnkeyableList, diags := types.ListValue(elemType, []attr.Value{validGroup1, unkeyableGroup})
		require.False(t, diags.HasError())

		result, diags := MergeRRSetListForRead(ctx, priorList, apiUnkeyableList)
		require.False(t, diags.HasError())
		require.Len(t, result.Elements(), 2, "Fallback must preserve unkeyable API element")
	})

	// Subtest 3: Unkeyable element in prior -> safe fallback to API list
	t.Run("unkeyable-prior-element", func(t *testing.T) {
		priorUnkeyableList, diags := types.ListValue(elemType, []attr.Value{unkeyableGroup})
		require.False(t, diags.HasError())
		apiList, diags := types.ListValue(elemType, []attr.Value{validGroup1, validGroup2})
		require.False(t, diags.HasError())

		result, diags := MergeRRSetListForRead(ctx, priorUnkeyableList, apiList)
		require.False(t, diags.HasError())
		require.Len(t, result.Elements(), 2, "Fallback must return API list when prior is unkeyable")
	})
}

// testCustomNSRecordRRSet creates an NS record RRSet object with custom TTL, nullable name,
// description, and multiple nameserver values for testing apex NS reconciliation.
//
// Why: Models user-configured apex NS records (where name may be omitted/null or empty ""
// and TTL differs from the 86400 backend standalone default) alongside API responses that
// append backend default nameservers.
//
// How: Constructs a typed RRSet ObjectValue with the provided ttl, description, name,
// and values list under the ns_record branch.
//
// Parameters:
//   - t: testing context.
//   - ttl: record time-to-live in seconds.
//   - name: record name attribute (may be null or concrete string).
//   - description: record description attribute (may be unknown, null, or concrete string).
//   - values: nameserver hostnames for ns_record.values.
//
// Return values:
//   - types.Object: the constructed RRSet object value.
//
// Example:
//
//	rrset := testCustomNSRecordRRSet(t, 360, types.StringNull(), types.StringUnknown(), "ns1.vestest.secondary.com")
func testCustomNSRecordRRSet(t *testing.T, ttl int64, name, description types.String, values ...string) types.Object {
	t.Helper()

	return testRRSetObject(t, map[string]attr.Value{
		"description": description,
		"ttl":         types.Int64Value(ttl),
		"type_record_set_choice": testRRSetTypeChoiceObject(t, map[string]attr.Value{
			"a_record": types.ObjectNull(testARecordAttrTypes()),
			"ns_record": testRRSetBranchObject(t, testNSRecordAttrTypes(), map[string]attr.Value{
				"name":   name,
				"values": testStringListValue(t, values...),
			}),
		}),
	})
}

// TestMergeRRSetListForApply_ApexNSRecordBackendInjectedValue verifies that when a user
// configures an apex NS record (name omitted/null or empty "", ttl=360) and the backend
// appends its default nameserver into the same ns_record.values slice, MergeRRSetListForApply
// strips the unconfigured backend default nameserver, resolves unknown description to null,
// and preserves the planned name representation.
//
// Why: Prevents Terraform core from aborting post-apply with "Provider produced inconsistent
// result after apply: ...ns_record.values: new element 1 has appeared" (XC-76508 / XC-6129).
//
// How: Exercises subtests for omitted name (null), explicit empty name (""), and user-configured
// default NS value retention.
//
// Parameters:
//   - t: testing context.
//
// Return values:
//   - None.
//
// Example:
//
//	go test -run TestMergeRRSetListForApply_ApexNSRecordBackendInjectedValue ./f5xc/terraform/tpf/utils/...
func TestMergeRRSetListForApply_ApexNSRecordBackendInjectedValue(t *testing.T) {
	ctx := context.Background()

	t.Run("omitted-null-name-strips-backend-injected-ns", func(t *testing.T) {
		planned := testRRSetListValue(t,
			testCustomNSRecordRRSet(t, 360, types.StringNull(), types.StringUnknown(), "ns1.vestest.secondary.com"),
		)
		api := testRRSetListValue(t,
			testCustomNSRecordRRSet(t, 360, types.StringNull(), types.StringNull(), "ns1.vestest.secondary.com", "ns1crt.f5clouddns-test.com"),
		)

		merged, diags := MergeRRSetListForApply(ctx, planned, api)
		require.False(t, diags.HasError())
		assert.False(t, ValueContainsUnknown(merged))

		want := testRRSetListValue(t,
			testCustomNSRecordRRSet(t, 360, types.StringNull(), types.StringNull(), "ns1.vestest.secondary.com"),
		)
		assert.Equal(t, want, merged)
	})

	t.Run("explicit-empty-name-preserved-and-strips-backend-injected-ns", func(t *testing.T) {
		planned := testRRSetListValue(t,
			testCustomNSRecordRRSet(t, 360, types.StringValue(""), types.StringUnknown(), "ns1.vestest.secondary.com"),
		)
		api := testRRSetListValue(t,
			testCustomNSRecordRRSet(t, 360, types.StringNull(), types.StringNull(), "ns1.vestest.secondary.com", "ns1crt.f5clouddns-test.com"),
		)

		merged, diags := MergeRRSetListForApply(ctx, planned, api)
		require.False(t, diags.HasError())
		assert.False(t, ValueContainsUnknown(merged))

		want := testRRSetListValue(t,
			testCustomNSRecordRRSet(t, 360, types.StringValue(""), types.StringNull(), "ns1.vestest.secondary.com"),
		)
		assert.Equal(t, want, merged)
	})

	t.Run("user-configured-default-ns-value-is-retained", func(t *testing.T) {
		planned := testRRSetListValue(t,
			testCustomNSRecordRRSet(t, 360, types.StringNull(), types.StringUnknown(), "ns1.vestest.secondary.com", "ns1crt.f5clouddns-test.com"),
		)
		api := testRRSetListValue(t,
			testCustomNSRecordRRSet(t, 360, types.StringNull(), types.StringNull(), "ns1.vestest.secondary.com", "ns1crt.f5clouddns-test.com"),
		)

		merged, diags := MergeRRSetListForApply(ctx, planned, api)
		require.False(t, diags.HasError())

		want := testRRSetListValue(t,
			testCustomNSRecordRRSet(t, 360, types.StringNull(), types.StringNull(), "ns1.vestest.secondary.com", "ns1crt.f5clouddns-test.com"),
		)
		assert.Equal(t, want, merged)
	})
}

// TestMergeRRSetListForRead_ApexNSRecordBackendInjectedValue verifies that during Read
// (both steady-state refresh and first-read/ImportState), backend-injected default NS
// values are stripped from an apex NS record while genuine out-of-band drift is preserved.
//
// Why: Ensures subsequent `terraform plan` and `ImportStateVerify` runs do not report
// perpetual drift on the backend-appended default nameserver while still catching real
// out-of-band modifications.
//
// How: Tests steady-state refresh with no drift, out-of-band addition of a non-default
// nameserver, and first-read (null prior state) sanitization.
//
// Parameters:
//   - t: testing context.
//
// Return values:
//   - None.
//
// Example:
//
//	go test -run TestMergeRRSetListForRead_ApexNSRecordBackendInjectedValue ./f5xc/terraform/tpf/utils/...
func TestMergeRRSetListForRead_ApexNSRecordBackendInjectedValue(t *testing.T) {
	ctx := context.Background()

	t.Run("steady-state-read-strips-backend-injected-ns", func(t *testing.T) {
		prior := testRRSetListValue(t,
			testCustomNSRecordRRSet(t, 360, types.StringNull(), types.StringNull(), "ns1.vestest.secondary.com"),
		)
		api := testRRSetListValue(t,
			testCustomNSRecordRRSet(t, 360, types.StringNull(), types.StringNull(), "ns1.vestest.secondary.com", "ns1crt.f5clouddns-test.com"),
		)

		merged, diags := MergeRRSetListForRead(ctx, prior, api)
		require.False(t, diags.HasError())
		assert.Equal(t, prior, merged)
	})

	t.Run("out-of-band-non-default-ns-surfaces-as-drift", func(t *testing.T) {
		prior := testRRSetListValue(t,
			testCustomNSRecordRRSet(t, 360, types.StringNull(), types.StringNull(), "ns1.vestest.secondary.com"),
		)
		api := testRRSetListValue(t,
			testCustomNSRecordRRSet(t, 360, types.StringNull(), types.StringNull(), "ns1.vestest.secondary.com", "ns2.custom.example.com", "ns1crt.f5clouddns-test.com"),
		)

		merged, diags := MergeRRSetListForRead(ctx, prior, api)
		require.False(t, diags.HasError())

		want := testRRSetListValue(t,
			testCustomNSRecordRRSet(t, 360, types.StringNull(), types.StringNull(), "ns1.vestest.secondary.com", "ns2.custom.example.com"),
		)
		assert.Equal(t, want, merged)
	})

	t.Run("first-read-import-strips-backend-injected-ns-from-apex-record", func(t *testing.T) {
		prior := types.ListNull(types.ObjectType{AttrTypes: testRRSetAttrTypes()})
		api := testRRSetListValue(t,
			testCustomNSRecordRRSet(t, 360, types.StringNull(), types.StringNull(), "ns1.vestest.secondary.com", "ns1crt.f5clouddns-test.com"),
		)

		merged, diags := MergeRRSetListForRead(ctx, prior, api)
		require.False(t, diags.HasError())

		want := testRRSetListValue(t,
			testCustomNSRecordRRSet(t, 360, types.StringNull(), types.StringNull(), "ns1.vestest.secondary.com"),
		)
		assert.Equal(t, want, merged)
	})
}

// TestMergeRRSetListForApply_MultipleLBRecordsWithSameName verifies that multiple lb_record
// entries sharing the same record name (e.g. "dynamic") but targeting different load
// balancers are uniquely keyed by (type, name, namespace, target_name) and reconciled
// without duplicate-key collisions even when the API returns them in a different order.
//
// Why: In GSLB/DNS e2e configurations (XC-6129), a single default_rr_set_group or
// rr_set_group can declare several lb_record entries with name="dynamic" pointing to
// different load balancers. Keying only by LB:dynamic caused indexRRSetAttrValues to fail
// on duplicate keys and fall back to positional indexing.
//
// How: Builds a planned list with two lb_record entries sharing name="dynamic" (with
// unknown tenant and description) and an API list in reversed order with concrete tenants,
// then verifies MergeRRSetListForApply preserves planned order and resolves each record's
// tenant from its matching load balancer target.
//
// Parameters:
//   - t: testing context.
//
// Return values:
//   - None.
//
// Example:
//
//	go test -run TestMergeRRSetListForApply_MultipleLBRecordsWithSameName ./f5xc/terraform/tpf/utils/...
func TestMergeRRSetListForApply_MultipleLBRecordsWithSameName(t *testing.T) {
	ctx := context.Background()
	elemType := types.ObjectType{AttrTypes: testRRSetWithLBAttrTypes()}

	plannedRec1 := testLBRecordRRSet(t, "dynamic", "auto-three-lb-rules-oaqq", "system", types.StringUnknown(), types.StringUnknown())
	plannedRec2 := testLBRecordRRSet(t, "dynamic", "auto-lb-aaaa1-drxe", "system", types.StringUnknown(), types.StringUnknown())
	plannedList, diags := types.ListValue(elemType, []attr.Value{plannedRec1, plannedRec2})
	require.False(t, diags.HasError())

	// API returns the two lb_records in reverse order with distinct resolved tenants
	apiRec2 := testLBRecordRRSet(t, "dynamic", "auto-lb-aaaa1-drxe", "system", types.StringValue("tenant-aaaa1"), types.StringNull())
	apiRec1 := testLBRecordRRSet(t, "dynamic", "auto-three-lb-rules-oaqq", "system", types.StringValue("tenant-oaqq"), types.StringNull())
	apiList, diags := types.ListValue(elemType, []attr.Value{apiRec2, apiRec1})
	require.False(t, diags.HasError())

	result, diags := MergeRRSetListForApply(ctx, plannedList, apiList)
	require.False(t, diags.HasError())
	assert.False(t, ValueContainsUnknown(result))

	expectedRec1 := testLBRecordRRSet(t, "dynamic", "auto-three-lb-rules-oaqq", "system", types.StringValue("tenant-oaqq"), types.StringNull())
	expectedRec2 := testLBRecordRRSet(t, "dynamic", "auto-lb-aaaa1-drxe", "system", types.StringValue("tenant-aaaa1"), types.StringNull())
	expectedList, diags := types.ListValue(elemType, []attr.Value{expectedRec1, expectedRec2})
	require.False(t, diags.HasError())

	assert.Equal(t, expectedList, result)
}

// TestMergeRRSetListForApply_IndexShiftedDescriptionDoesNotBleed verifies that when
// the backend prepends a default NS record at API[0] and specToModel positionally
// copies Plan[1].description onto API[1] (which is actually Record 0),
// MergeRRSetListForApply keeps Record 0's omitted description as null.
//
// Why: Prevents index-shifted positional healing in specToModel from bleeding a
// neighbor record's description onto a record whose description was omitted.
//
// How: Constructs a planned list where Record 0 has unknown description and Record 1
// has description="rec2-desc", and an API list with a prepended default NS at API[0]
// where API[1] (Record 0) was positionally contaminated with "rec2-desc".
//
// Parameters:
//   - t: testing context.
//
// Return values:
//   - None.
func TestMergeRRSetListForApply_IndexShiftedDescriptionDoesNotBleed(t *testing.T) {
	ctx := context.Background()

	planned := testRRSetListValue(t,
		testARecordRRSet(t, "rec1", "1.1.1.1", types.StringUnknown()),
		testARecordRRSet(t, "rec2", "2.2.2.2", types.StringValue("rec2-desc")),
	)

	// Simulate specToModel output where API[0] is the backend default NS record,
	// API[1] is rec1 (whose empty description was positionally healed from Plan[1] "rec2-desc"),
	// and API[2] is rec2.
	api := testRRSetListValue(t,
		testNSRecordRRSet(t, "", "ns1crt.f5clouddns-test.com", types.StringNull()),
		testARecordRRSet(t, "rec1", "1.1.1.1", types.StringValue("rec2-desc")),
		testARecordRRSet(t, "rec2", "2.2.2.2", types.StringValue("rec2-desc")),
	)

	merged, diags := MergeRRSetListForApply(ctx, planned, api)
	require.False(t, diags.HasError())
	assert.False(t, ValueContainsUnknown(merged))

	want := testRRSetListValue(t,
		testARecordRRSet(t, "rec1", "1.1.1.1", types.StringNull()),
		testARecordRRSet(t, "rec2", "2.2.2.2", types.StringValue("rec2-desc")),
	)
	assert.Equal(t, want, merged)
}

// TestMergeRRSetListForRead_IndexShiftedDescriptionDoesNotBleed verifies that during
// Read refresh, when the backend prepends a default NS record at API[0] and specToModel
// positionally copies State[1].description onto API[1] (which is actually Record 0),
// MergeRRSetListForRead restores Record 0's null description from its key-matched
// prior state element.
//
// Why: Without sanitizing index-shifted descriptions before filtering out API[0],
// MergeRRSetListForRead would overwrite Record 0's null description with Record 1's
// description on every refresh, causing perpetual drift.
//
// How: Constructs a prior state list where Record 0 has null description and Record 1
// has description="rec2-desc", and an API list with a prepended default NS at API[0]
// where API[1] (Record 0) has description="rec2-desc".
//
// Parameters:
//   - t: testing context.
//
// Return values:
//   - None.
func TestMergeRRSetListForRead_IndexShiftedDescriptionDoesNotBleed(t *testing.T) {
	ctx := context.Background()

	prior := testRRSetListValue(t,
		testARecordRRSet(t, "rec1", "1.1.1.1", types.StringNull()),
		testARecordRRSet(t, "rec2", "2.2.2.2", types.StringValue("rec2-desc")),
	)

	api := testRRSetListValue(t,
		testNSRecordRRSet(t, "", "ns1crt.f5clouddns-test.com", types.StringNull()),
		testARecordRRSet(t, "rec1", "1.1.1.1", types.StringValue("rec2-desc")),
		testARecordRRSet(t, "rec2", "2.2.2.2", types.StringValue("rec2-desc")),
	)

	merged, diags := MergeRRSetListForRead(ctx, prior, api)
	require.False(t, diags.HasError())
	assert.Equal(t, prior, merged)
}

// TestMergeRRSetListForRead_PreservesIPv6CasingAndMultiValueOrder verifies that
// MergeRRSetListForRead preserves the prior state's representation when the backend
// lowercases IPv6 hex digits on an aaaa_record or reorders multi-value slices, while
// still surfacing genuine out-of-band value changes.
//
// Why: In TestGSLBRecords (XC-6129), users configure uppercase IPv6 addresses
// ("AAAA::111:111:111:111") and multi-value records (["111.111.111.111", "22.22.22.22"]).
// Backend canonicalization or slice sorting must not cause a non-empty plan on refresh.
//
// How: Exercises subtests for IPv6 hex casing normalization, multi-value slice reordering,
// and genuine IPv6 out-of-band value drift.
//
// Parameters:
//   - t: testing context.
//
// Return values:
//   - None.
func TestMergeRRSetListForRead_PreservesIPv6CasingAndMultiValueOrder(t *testing.T) {
	ctx := context.Background()

	t.Run("ipv6-uppercase-preserved-when-api-lowercases", func(t *testing.T) {
		prior := testRRSetWithAAAAListValue(t,
			testAAAARecordRRSetMulti(t, "static", types.StringNull(), "AAAA::111:111:111:111"),
		)
		api := testRRSetWithAAAAListValue(t,
			testAAAARecordRRSetMulti(t, "static", types.StringNull(), "aaaa::111:111:111:111"),
		)

		merged, diags := MergeRRSetListForRead(ctx, prior, api)
		require.False(t, diags.HasError())
		assert.Equal(t, prior, merged)
	})

	t.Run("multi-value-a-record-order-preserved-when-api-reorders", func(t *testing.T) {
		prior := testRRSetWithAAAAListValue(t,
			testARecordRRSetMultiWithAAAA(t, "static", types.StringNull(), "111.111.111.111", "22.22.22.22"),
		)
		api := testRRSetWithAAAAListValue(t,
			testARecordRRSetMultiWithAAAA(t, "static", types.StringNull(), "22.22.22.22", "111.111.111.111"),
		)

		merged, diags := MergeRRSetListForRead(ctx, prior, api)
		require.False(t, diags.HasError())
		assert.Equal(t, prior, merged)
	})

	t.Run("genuine-ipv6-value-change-surfaces-as-drift", func(t *testing.T) {
		prior := testRRSetWithAAAAListValue(t,
			testAAAARecordRRSetMulti(t, "static", types.StringNull(), "AAAA::111:111:111:111"),
		)
		api := testRRSetWithAAAAListValue(t,
			testAAAARecordRRSetMulti(t, "static", types.StringNull(), "aaaa::222:222:222:222"),
		)

		merged, diags := MergeRRSetListForRead(ctx, prior, api)
		require.False(t, diags.HasError())
		assert.Equal(t, api, merged)
	})
}

// TestMergeRRSetListForRead_RRSetGroup_PreservesEmptyMetadataDescription verifies
// that an explicit empty string description ("") on rr_set_group[*].metadata.description
// is preserved when specToModel normalizes the API response to null.
//
// Why: Prevents false read drift when a user configures metadata.description = "".
//
// How: Builds a prior RRSetGroup with metadata.description = "" and an API RRSetGroup
// with metadata.description = null, then verifies MergeRRSetListForRead preserves "".
//
// Parameters:
//   - t: testing context.
//
// Return values:
//   - None.
func TestMergeRRSetListForRead_RRSetGroup_PreservesEmptyMetadataDescription(t *testing.T) {
	ctx := context.Background()
	elemType := types.ObjectType{AttrTypes: testRRSetGroupWithLBAttrTypes()}

	priorMeta, diags := types.ObjectValue(testRRSetGroupMetaAttrTypes(), map[string]attr.Value{
		"name":        types.StringValue("group1"),
		"description": types.StringValue(""),
	})
	require.False(t, diags.HasError())

	rrsetElemType := types.ObjectType{AttrTypes: testRRSetWithLBAttrTypes()}
	rec := testLBRecordRRSet(t, "rec1", "lb1", "sys", types.StringValue("t1"), types.StringNull())
	rrsetList, diags := types.ListValue(rrsetElemType, []attr.Value{rec})
	require.False(t, diags.HasError())

	priorGroup, diags := types.ObjectValue(testRRSetGroupWithLBAttrTypes(), map[string]attr.Value{
		"metadata": priorMeta,
		"rr_set":   rrsetList,
	})
	require.False(t, diags.HasError())
	priorList, diags := types.ListValue(elemType, []attr.Value{priorGroup})
	require.False(t, diags.HasError())

	apiGroup := testRRSetGroupObject(t, "group1", rec)
	apiList, diags := types.ListValue(elemType, []attr.Value{apiGroup})
	require.False(t, diags.HasError())

	merged, diags := MergeRRSetListForRead(ctx, priorList, apiList)
	require.False(t, diags.HasError())
	assert.Equal(t, priorList, merged)
}

// TestMergeRRSetListForRead_FirstReadWithDuplicateKeysPopulatesState verifies that
// on first read or ImportState (when prior is null), MergeRRSetListForRead populates
// state from the sanitized API list even if the API list contains duplicate keys.
//
// Why: Evaluating indexRRSetAttrValues before checking prior.IsNull() previously
// caused ImportState to return null whenever duplicate record keys existed.
//
// How: Calls MergeRRSetListForRead with prior = ListNull and an API list containing
// two identical A records plus a backend default NS record, and verifies the two A
// records are returned.
//
// Parameters:
//   - t: testing context.
//
// Return values:
//   - None.
func TestMergeRRSetListForRead_FirstReadWithDuplicateKeysPopulatesState(t *testing.T) {
	ctx := context.Background()
	prior := types.ListNull(types.ObjectType{AttrTypes: testRRSetAttrTypes()})

	api := testRRSetListValue(t,
		testARecordRRSet(t, "dup.example.com", "1.1.1.1", types.StringNull()),
		testARecordRRSet(t, "dup.example.com", "2.2.2.2", types.StringNull()),
		testNSRecordRRSet(t, "", "ns1crt.f5clouddns-test.com", types.StringNull()),
	)

	merged, diags := MergeRRSetListForRead(ctx, prior, api)
	require.False(t, diags.HasError())

	want := testRRSetListValue(t,
		testARecordRRSet(t, "dup.example.com", "1.1.1.1", types.StringNull()),
		testARecordRRSet(t, "dup.example.com", "2.2.2.2", types.StringNull()),
	)
	assert.Equal(t, want, merged)
}

// testRRSetTypeChoiceWithAAAAAttrTypes returns the oneof object attribute map including aaaa_record.
//
// Why: Supports testing IPv6 aaaa_record canonicalization alongside a_record and ns_record.
//
// How: Constructs a map with a_record, aaaa_record, and ns_record ObjectTypes.
//
// Parameters:
//   - None.
//
// Return values:
//   - map[string]attr.Type: schema attribute types.
func testRRSetTypeChoiceWithAAAAAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"a_record":    types.ObjectType{AttrTypes: testARecordAttrTypes()},
		"aaaa_record": types.ObjectType{AttrTypes: testARecordAttrTypes()},
		"ns_record":   types.ObjectType{AttrTypes: testNSRecordAttrTypes()},
	}
}

// testRRSetWithAAAAAttrTypes returns the RRSet object attribute map including aaaa_record.
//
// Why: Defines the RRSet schema used by IPv6 and multi-value merge tests.
//
// How: Constructs a map with description, ttl, and type_record_set_choice ObjectType.
//
// Parameters:
//   - None.
//
// Return values:
//   - map[string]attr.Type: schema attribute types.
func testRRSetWithAAAAAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"description":            types.StringType,
		"ttl":                    types.Int64Type,
		"type_record_set_choice": types.ObjectType{AttrTypes: testRRSetTypeChoiceWithAAAAAttrTypes()},
	}
}

// testRRSetWithAAAAListValue creates a typed RRSet list supporting aaaa_record branches.
//
// Why: Builds list fixtures for IPv6 and multi-value read reconciliation tests.
//
// How: Wraps the given RRSet objects into a types.ListValue of testRRSetWithAAAAAttrTypes.
//
// Parameters:
//   - t: testing context.
//   - rrsets: RRSet objects to include in the list.
//
// Return values:
//   - types.List: typed RRSet list value.
func testRRSetWithAAAAListValue(t *testing.T, rrsets ...types.Object) types.List {
	t.Helper()
	elements := make([]attr.Value, 0, len(rrsets))
	for _, r := range rrsets {
		elements = append(elements, r)
	}
	list, diags := types.ListValue(types.ObjectType{AttrTypes: testRRSetWithAAAAAttrTypes()}, elements)
	require.False(t, diags.HasError())
	return list
}

// testAAAARecordRRSetMulti creates an AAAA record RRSet object with one or more IPv6 values.
//
// Why: Models user-configured and API-returned AAAA records in unit tests.
//
// How: Populates the aaaa_record branch of testRRSetWithAAAAAttrTypes.
//
// Parameters:
//   - t: testing context.
//   - name: record name.
//   - description: record description.
//   - values: IPv6 address strings.
//
// Return values:
//   - types.Object: constructed RRSet object.
func testAAAARecordRRSetMulti(t *testing.T, name string, description types.String, values ...string) types.Object {
	t.Helper()
	choiceObj, diags := types.ObjectValue(testRRSetTypeChoiceWithAAAAAttrTypes(), map[string]attr.Value{
		"a_record": types.ObjectNull(testARecordAttrTypes()),
		"aaaa_record": testRRSetBranchObject(t, testARecordAttrTypes(), map[string]attr.Value{
			"name":   types.StringValue(name),
			"values": testStringListValue(t, values...),
		}),
		"ns_record": types.ObjectNull(testNSRecordAttrTypes()),
	})
	require.False(t, diags.HasError())

	obj, diags := types.ObjectValue(testRRSetWithAAAAAttrTypes(), map[string]attr.Value{
		"description":            description,
		"ttl":                    types.Int64Value(60),
		"type_record_set_choice": choiceObj,
	})
	require.False(t, diags.HasError())
	return obj
}

// testARecordRRSetMultiWithAAAA creates a multi-value A record RRSet object using the AAAA-capable schema.
//
// Why: Allows testing multi-value A record slice reordering with the same schema helper.
//
// How: Populates the a_record branch of testRRSetWithAAAAAttrTypes.
//
// Parameters:
//   - t: testing context.
//   - name: record name.
//   - description: record description.
//   - values: IPv4 address strings.
//
// Return values:
//   - types.Object: constructed RRSet object.
func testARecordRRSetMultiWithAAAA(t *testing.T, name string, description types.String, values ...string) types.Object {
	t.Helper()
	choiceObj, diags := types.ObjectValue(testRRSetTypeChoiceWithAAAAAttrTypes(), map[string]attr.Value{
		"a_record": testRRSetBranchObject(t, testARecordAttrTypes(), map[string]attr.Value{
			"name":   types.StringValue(name),
			"values": testStringListValue(t, values...),
		}),
		"aaaa_record": types.ObjectNull(testARecordAttrTypes()),
		"ns_record":   types.ObjectNull(testNSRecordAttrTypes()),
	})
	require.False(t, diags.HasError())

	obj, diags := types.ObjectValue(testRRSetWithAAAAAttrTypes(), map[string]attr.Value{
		"description":            description,
		"ttl":                    types.Int64Value(60),
		"type_record_set_choice": choiceObj,
	})
	require.False(t, diags.HasError())
	return obj
}

// mismatchNullType is a string-backed custom type whose null value has the
// plain StringType. Collections of this type cannot hold the resolved null.
type mismatchNullType struct{ basetypes.StringType }

func (mismatchNullType) Equal(o attr.Type) bool {
	_, ok := o.(mismatchNullType)
	return ok
}

func (mismatchNullType) String() string { return "mismatchNullType" }

func (mismatchNullType) ValueType(context.Context) attr.Value { return mismatchNullValue{} }

func (mismatchNullType) ValueFromTerraform(context.Context, tftypes.Value) (attr.Value, error) {
	return types.StringNull(), nil
}

type mismatchNullValue struct{ basetypes.StringValue }

func (mismatchNullValue) Type(context.Context) attr.Type { return mismatchNullType{} }

// failingNullType cannot build a null value.
type failingNullType struct{ basetypes.StringType }

func (failingNullType) ValueFromTerraform(context.Context, tftypes.Value) (attr.Value, error) {
	return nil, errors.New("no null value")
}

func unknownMismatch() mismatchNullValue {
	return mismatchNullValue{StringValue: types.StringUnknown()}
}

// --- retry and wait helpers ---

// The public delete wrapper uses the default config and returns nil on success.
func TestDeleteWithTransientRetry_DefaultConfig(t *testing.T) {
	calls := 0
	err := DeleteWithTransientRetry(context.Background(), "r", func() error {
		calls++
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, calls)
	assert.Equal(t, 6, defaultTransientDeleteRetryConfig().maxAttempts)
}

// A zero attempt count, a negative interval and a nil sleep function are
// normalized to one attempt with no wait.
func TestDeleteWithTransientRetry_NormalizesConfig(t *testing.T) {
	calls := 0
	err := deleteWithTransientRetryWithConfig(context.Background(), "r", func() error {
		calls++
		return errors.New("status code 503")
	}, transientDeleteRetryConfig{maxAttempts: 0, retryInterval: -time.Second})
	require.Error(t, err)
	assert.Equal(t, 1, calls)
}

// The public create wrapper uses the default config and returns the response.
func TestCreateWithTransientRetry_DefaultConfig(t *testing.T) {
	calls := 0
	_, err := CreateWithTransientRetry(context.Background(), "r", func() (proto.Message, error) {
		calls++
		return nil, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, calls)
	assert.Equal(t, 6, defaultTransientCreateRetryConfig().maxAttempts)
}

// A zero attempt count, a negative interval and a nil sleep function are
// normalized to one attempt with no wait.
func TestCreateWithTransientRetry_NormalizesConfig(t *testing.T) {
	calls := 0
	_, err := createWithTransientRetryWithConfig(context.Background(), "r", func() (proto.Message, error) {
		calls++
		return nil, errors.New("status code 503")
	}, transientCreateRetryConfig{maxAttempts: 0, retryInterval: -time.Second})
	require.Error(t, err)
	assert.Equal(t, 1, calls)
}

// The public initializer wait returns nil when nothing is pending.
func TestWaitForPendingInitializers_DefaultConfig(t *testing.T) {
	t.Parallel()
	getter := func(context.Context, string, string) (*mockPendingResource, error) {
		return &mockPendingResource{}, nil
	}
	require.NoError(t, WaitForPendingInitializers(context.Background(), "n", "ns", "uid", getter))
	assert.Equal(t, 20, defaultWaitConfig().maxRetries)
}

// The public finalizer wait returns nil when the object is gone.
func TestWaitForPendingFinalizers_DefaultConfig(t *testing.T) {
	t.Parallel()
	getter := func(context.Context, string, string) (*mockFinalizerResource, error) {
		return nil, errors.New("status code 404")
	}
	require.NoError(t, WaitForPendingFinalizers(context.Background(), "n", "ns", "uid", getter))
	assert.Equal(t, 30, defaultFinalizerWaitConfig().maxRetries)
}

// Backoff never exceeds maxBackoff, and a negative retry count skips polling.
func TestWaitConfigs_BackoffCapAndNoRetries(t *testing.T) {
	ctx := context.Background()
	var sleeps []time.Duration
	record := func(d time.Duration) { sleeps = append(sleeps, d) }

	pending := func(context.Context, string, string) (*mockPendingResource, error) {
		return &mockPendingResource{pending: [][]string{{"init"}}}, nil
	}
	initCfg := waitForPendingInitializersConfig{maxRetries: 2, initialBackoff: 2, maxBackoff: 1, sleep: record}
	require.Error(t, waitForPendingInitializersWithConfig(ctx, "n", "ns", "uid", pending, initCfg))
	assert.Equal(t, []time.Duration{2, 2, 1}, sleeps)

	sleeps = nil
	exists := func(context.Context, string, string) (*mockFinalizerResource, error) {
		return &mockFinalizerResource{}, nil
	}
	finCfg := waitForPendingFinalizersConfig{maxRetries: 2, initialBackoff: 2, maxBackoff: 1, sleep: record}
	require.Error(t, waitForPendingFinalizersWithConfig(ctx, "n", "ns", "uid", exists, finCfg))
	assert.Equal(t, []time.Duration{2, 2, 1}, sleeps)

	initCfg.maxRetries = -1
	finCfg.maxRetries = -1
	assert.NoError(t, waitForPendingInitializersWithConfig(ctx, "n", "ns", "uid", pending, initCfg))
	assert.NoError(t, waitForPendingFinalizersWithConfig(ctx, "n", "ns", "uid", exists, finCfg))
}

// --- error sanitization ---

// SanitizeAPIError falls back to the raw error when the status or body cannot
// be parsed or the status is below 400, and uses a generic message for an
// empty 4xx message.
func TestSanitizeAPIError_Fallbacks(t *testing.T) {
	ctx := context.Background()
	tests := map[string]struct {
		err  string
		want string
	}{
		"status_overflow": {`status code 99999999999999999999, body {"code":1}`, "got error: status code 9999"},
		"invalid_json":    {`status code 400, body {not-json}`, "got error: status code 400"},
		"empty_4xx_msg":   {`status code 404, body {"code":5}`, "request rejected by API"},
		"below_400":       {`status code 302, body {"code":0}`, "got error: status code 302"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := SanitizeAPIError(ctx, "read", "foo", errors.New(tc.err))
			assert.Contains(t, got, tc.want)
		})
	}
}

// --- enum and map helpers ---

// Empty underscore-separated tokens are skipped; input with no tokens is
// returned unchanged.
func TestResolveEnumStateValue_EmptyTokens(t *testing.T) {
	assert.Equal(t, "AB", ResolveEnumStateValue("A__B", true))
	assert.Equal(t, "_", ResolveEnumStateValue("_", true))
	assert.Equal(t, "Mixed_Case", ResolveEnumStateValue("Mixed_Case", true))
}

// A map whose elements are not strings cannot be converted.
func TestTerraformMapToUint32Map_NonStringElements(t *testing.T) {
	in := types.MapValueMust(types.Int64Type, map[string]attr.Value{"1": types.Int64Value(1)})
	out, diags := TerraformMapToUint32Map(context.Background(), in)
	assert.True(t, diags.HasError())
	assert.Nil(t, out)
}

// MergeStringMaps returns the non-null side when the other is null or
// unknown, lets override keys win, and keeps base on conversion errors.
func TestMergeStringMaps(t *testing.T) {
	base := types.MapValueMust(types.StringType, map[string]attr.Value{
		"a": types.StringValue("1"), "b": types.StringValue("2"),
	})
	override := types.MapValueMust(types.StringType, map[string]attr.Value{
		"b": types.StringValue("20"), "c": types.StringValue("3"),
	})
	intMap := types.MapValueMust(types.Int64Type, map[string]attr.Value{"x": types.Int64Value(1)})

	got, diags := MergeStringMaps(base, types.MapNull(types.StringType))
	require.False(t, diags.HasError())
	assert.True(t, got.Equal(base))

	got, diags = MergeStringMaps(types.MapUnknown(types.StringType), override)
	require.False(t, diags.HasError())
	assert.True(t, got.Equal(override))

	got, diags = MergeStringMaps(base, override)
	require.False(t, diags.HasError())
	want := types.MapValueMust(types.StringType, map[string]attr.Value{
		"a": types.StringValue("1"), "b": types.StringValue("20"), "c": types.StringValue("3"),
	})
	assert.True(t, got.Equal(want))

	got, diags = MergeStringMaps(intMap, override)
	assert.True(t, diags.HasError())
	assert.True(t, got.Equal(intMap))

	got, diags = MergeStringMaps(base, intMap)
	assert.True(t, diags.HasError())
	assert.True(t, got.Equal(base))
}

// --- struct conversion ---

type covStruct struct {
	A string `tfsdk:"a"`
}

// GoStructSliceToSetValue converts each struct into an object element and
// returns a null set when an element does not match the attribute map.
func TestGoStructSliceToSetValue(t *testing.T) {
	ctx := context.Background()
	attrs := map[string]attr.Type{"a": types.StringType}

	set, diags := GoStructSliceToSetValue(ctx, attrs, []covStruct{{A: "x"}, {A: "y"}})
	require.False(t, diags.HasError())
	assert.Len(t, set.Elements(), 2)

	empty, diags := GoStructSliceToSetValue[covStruct](ctx, attrs, nil)
	require.False(t, diags.HasError())
	assert.False(t, empty.IsNull())
	assert.Empty(t, empty.Elements())

	bad, diags := GoStructSliceToSetValue(ctx, map[string]attr.Type{"b": types.StringType}, []covStruct{{A: "x"}})
	assert.True(t, diags.HasError())
	assert.True(t, bad.IsNull())
}

// GoStructSliceToListValue returns a null list when an element does not match
// the attribute map.
func TestGoStructSliceToListValue_Mismatch(t *testing.T) {
	list, diags := GoStructSliceToListValue(context.Background(), map[string]attr.Type{"b": types.StringType}, []covStruct{{A: "x"}})
	assert.True(t, diags.HasError())
	assert.True(t, list.IsNull())
}

// --- unknown detection and resolution ---

// ValueContainsUnknown inspects set, map and tuple children and treats nil as
// known.
func TestValueContainsUnknown_Collections(t *testing.T) {
	assert.False(t, ValueContainsUnknown(nil))

	unknownSet := types.SetValueMust(types.StringType, []attr.Value{types.StringUnknown()})
	knownSet := types.SetValueMust(types.StringType, []attr.Value{types.StringValue("x")})
	unknownMap := types.MapValueMust(types.StringType, map[string]attr.Value{"k": types.StringUnknown()})
	knownMap := types.MapValueMust(types.StringType, map[string]attr.Value{"k": types.StringValue("x")})
	unknownTuple := types.TupleValueMust([]attr.Type{types.StringType}, []attr.Value{types.StringUnknown()})
	knownTuple := types.TupleValueMust([]attr.Type{types.StringType}, []attr.Value{types.StringValue("x")})

	assert.True(t, ValueContainsUnknown(unknownSet))
	assert.False(t, ValueContainsUnknown(knownSet))
	assert.True(t, ValueContainsUnknown(unknownMap))
	assert.False(t, ValueContainsUnknown(knownMap))
	assert.True(t, ValueContainsUnknown(unknownTuple))
	assert.False(t, ValueContainsUnknown(knownTuple))
}

// ResolveUnknowns fills unknown set, map and tuple elements from the API value
// and uses typed nulls when the API value is absent.
func TestResolveUnknowns_SetMapTuple(t *testing.T) {
	planSet := types.SetValueMust(types.StringType, []attr.Value{types.StringUnknown()})
	gotSet := ResolveUnknowns(planSet, types.SetValueMust(types.StringType, []attr.Value{types.StringValue("x")}))
	assert.True(t, gotSet.Equal(types.SetValueMust(types.StringType, []attr.Value{types.StringValue("x")})))
	gotSet = ResolveUnknowns(planSet, types.SetNull(types.StringType))
	assert.True(t, gotSet.Equal(types.SetValueMust(types.StringType, []attr.Value{types.StringNull()})))

	planMap := types.MapValueMust(types.StringType, map[string]attr.Value{"k": types.StringUnknown()})
	gotMap := ResolveUnknowns(planMap, types.MapValueMust(types.StringType, map[string]attr.Value{"k": types.StringValue("v")}))
	assert.True(t, gotMap.Equal(types.MapValueMust(types.StringType, map[string]attr.Value{"k": types.StringValue("v")})))
	gotMap = ResolveUnknowns(planMap, types.MapNull(types.StringType))
	assert.True(t, gotMap.Equal(types.MapValueMust(types.StringType, map[string]attr.Value{"k": types.StringNull()})))

	tupleTypes := []attr.Type{types.StringType}
	planTuple := types.TupleValueMust(tupleTypes, []attr.Value{types.StringUnknown()})
	gotTuple := ResolveUnknowns(planTuple, types.TupleValueMust(tupleTypes, []attr.Value{types.StringValue("x")}))
	assert.True(t, gotTuple.Equal(types.TupleValueMust(tupleTypes, []attr.Value{types.StringValue("x")})))
	gotTuple = ResolveUnknowns(planTuple, types.TupleNull(tupleTypes))
	assert.True(t, gotTuple.Equal(types.TupleValueMust(tupleTypes, []attr.Value{types.StringNull()})))
}

// When a resolved child does not match the declared element type, the
// collection keeps its plan value, so the unknown child remains.
func TestResolveUnknowns_TypeMismatchKeepsPlan(t *testing.T) {
	mt := mismatchNullType{}
	u := unknownMismatch()

	values := map[string]attr.Value{
		"object": types.ObjectValueMust(map[string]attr.Type{"a": mt}, map[string]attr.Value{"a": u}),
		"list":   types.ListValueMust(mt, []attr.Value{u}),
		"set":    types.SetValueMust(mt, []attr.Value{u}),
		"map":    types.MapValueMust(mt, map[string]attr.Value{"k": u}),
		"tuple":  types.TupleValueMust([]attr.Type{mt}, []attr.Value{u}),
	}
	for name, v := range values {
		t.Run(name, func(t *testing.T) {
			got := ResolveUnknowns(v, nil)
			assert.True(t, ValueContainsUnknown(got))
			assert.True(t, got.Type(context.Background()).Equal(v.Type(context.Background())))
		})
	}

	// The resolved null is not a mismatchNullValue, so the plan value is kept.
	assert.True(t, ResolveUnknowns(u, u).IsUnknown())
}

// typedNull returns nil when no type is given or the type cannot build a null.
func TestTypedNull_Failures(t *testing.T) {
	ctx := context.Background()
	assert.Nil(t, typedNull(ctx, nil))
	assert.Nil(t, typedNull(ctx, failingNullType{}))
}

// --- RRSet fixtures ---

func covRRSet(t *testing.T, desc types.String, ttl types.Int64, choice attr.Value) types.Object {
	t.Helper()
	return testRRSetObject(t, map[string]attr.Value{
		"description":            desc,
		"ttl":                    ttl,
		"type_record_set_choice": choice,
	})
}

func covNSChoice(t *testing.T, name types.String, values types.List) types.Object {
	t.Helper()
	return testRRSetTypeChoiceObject(t, map[string]attr.Value{
		"a_record": types.ObjectNull(testARecordAttrTypes()),
		"ns_record": testRRSetBranchObject(t, testNSRecordAttrTypes(), map[string]attr.Value{
			"name":   name,
			"values": values,
		}),
	})
}

func covEmptyChoice(t *testing.T) types.Object {
	t.Helper()
	return testRRSetTypeChoiceObject(t, map[string]attr.Value{
		"a_record":  types.ObjectNull(testARecordAttrTypes()),
		"ns_record": types.ObjectNull(testNSRecordAttrTypes()),
	})
}

func covUnkeyableRRSet(t *testing.T) types.Object {
	t.Helper()
	return covRRSet(t, types.StringNull(), types.Int64Value(60), covEmptyChoice(t))
}

func covLBRRSet(t *testing.T, lbName, lbNamespace types.String) types.Object {
	t.Helper()
	choice := types.ObjectValueMust(testRRSetTypeChoiceWithLBAttrTypes(), map[string]attr.Value{
		"a_record":  types.ObjectNull(testARecordAttrTypes()),
		"ns_record": types.ObjectNull(testNSRecordAttrTypes()),
		"lb_record": types.ObjectValueMust(testLBRecordAttrTypes(), map[string]attr.Value{
			"name": types.StringValue("www"),
			"value": types.ObjectValueMust(testLBRecordValueAttrTypes(), map[string]attr.Value{
				"name":      lbName,
				"namespace": lbNamespace,
				"tenant":    types.StringNull(),
			}),
		}),
	})
	return types.ObjectValueMust(testRRSetWithLBAttrTypes(), map[string]attr.Value{
		"description":            types.StringNull(),
		"ttl":                    types.Int64Value(60),
		"type_record_set_choice": choice,
	})
}

var defaultNSValues = []string{"ns1.f5clouddns.com", "ns2.f5clouddns.com"}

// --- RRSet helper edge cases ---

// isBackendDefaultRRSet returns false for every malformed or incomplete
// RRSet and true for a backend default with a null name.
func TestIsBackendDefaultRRSet_EdgeCases(t *testing.T) {
	ttl := types.Int64Value(86400)
	defaults := testStringListValue(t, defaultNSValues...)
	tcType := types.ObjectType{AttrTypes: testRRSetTypeChoiceAttrTypes()}

	cases := map[string]attr.Value{
		"not_object": types.StringValue("x"),
		"null_rrset": types.ObjectNull(testRRSetAttrTypes()),
		"no_choice": types.ObjectValueMust(map[string]attr.Type{"ttl": types.Int64Type},
			map[string]attr.Value{"ttl": ttl}),
		"null_choice":    covRRSet(t, types.StringNull(), ttl, types.ObjectNull(testRRSetTypeChoiceAttrTypes())),
		"no_branch":      covRRSet(t, types.StringNull(), ttl, covEmptyChoice(t)),
		"unknown_name":   covRRSet(t, types.StringNull(), ttl, covNSChoice(t, types.StringUnknown(), defaults)),
		"null_ttl":       covRRSet(t, types.StringNull(), types.Int64Null(), covNSChoice(t, types.StringValue(""), defaults)),
		"null_values":    covRRSet(t, types.StringNull(), ttl, covNSChoice(t, types.StringValue(""), types.ListNull(types.StringType))),
		"unknown_values": covRRSet(t, types.StringNull(), ttl, covNSChoice(t, types.StringValue(""), types.ListValueMust(types.StringType, []attr.Value{types.StringUnknown()}))),
		"no_ttl": types.ObjectValueMust(map[string]attr.Type{"type_record_set_choice": tcType},
			map[string]attr.Value{"type_record_set_choice": covNSChoice(t, types.StringValue(""), defaults)}),
	}
	for name, v := range cases {
		t.Run(name, func(t *testing.T) {
			assert.False(t, isBackendDefaultRRSet(v))
		})
	}

	assert.True(t, isBackendDefaultRRSet(covRRSet(t, types.StringNull(), ttl, covNSChoice(t, types.StringNull(), defaults))))
}

// stripBackendDefaultNSValues leaves malformed, named or valueless records
// unchanged and removes default NS values from an apex record with a null name.
func TestStripBackendDefaultNSValues_EdgeCases(t *testing.T) {
	ttl := types.Int64Value(86400)
	values := testStringListValue(t, "ns1.f5clouddns.com", "custom.example.com")

	unchanged := map[string]attr.Value{
		"not_object": types.StringValue("x"),
		"no_choice": types.ObjectValueMust(map[string]attr.Type{"ttl": types.Int64Type},
			map[string]attr.Value{"ttl": ttl}),
		"null_choice":  covRRSet(t, types.StringNull(), ttl, types.ObjectNull(testRRSetTypeChoiceAttrTypes())),
		"unknown_name": covRRSet(t, types.StringNull(), ttl, covNSChoice(t, types.StringUnknown(), values)),
		"named":        covRRSet(t, types.StringNull(), ttl, covNSChoice(t, types.StringValue("sub"), values)),
		"null_values":  covRRSet(t, types.StringNull(), ttl, covNSChoice(t, types.StringValue(""), types.ListNull(types.StringType))),
	}
	for name, v := range unchanged {
		t.Run(name, func(t *testing.T) {
			assert.True(t, stripBackendDefaultNSValues(nil, v).Equal(v))
		})
	}

	api := covRRSet(t, types.StringNull(), ttl, covNSChoice(t, types.StringNull(), values))
	want := covRRSet(t, types.StringNull(), ttl, covNSChoice(t, types.StringNull(), testStringListValue(t, "custom.example.com")))
	assert.True(t, stripBackendDefaultNSValues(nil, api).Equal(want))
}

// preserveRRSetEmptyFields keeps the reference form of empty descriptions and
// names, and clears a description the reference leaves unknown.
func TestPreserveRRSetEmptyFields_EdgeCases(t *testing.T) {
	ttl := types.Int64Value(3600)
	values := testStringListValue(t, "ns.example.com")
	target := covRRSet(t, types.StringValue("x"), ttl, covNSChoice(t, types.StringNull(), values))

	assert.True(t, preserveRRSetEmptyFields(types.StringValue("x"), target).Equal(target))

	ref := covRRSet(t, types.StringUnknown(), ttl, covNSChoice(t, types.StringNull(), values))
	got := preserveRRSetEmptyFields(ref, target)
	assert.True(t, got.Equal(covRRSet(t, types.StringNull(), ttl, covNSChoice(t, types.StringNull(), values))))

	ref = covRRSet(t, types.StringValue(""), ttl, covNSChoice(t, types.StringValue(""), values))
	target = covRRSet(t, types.StringNull(), ttl, covNSChoice(t, types.StringNull(), values))
	got = preserveRRSetEmptyFields(ref, target)
	assert.True(t, got.Equal(ref))
}

// sanitizeIndexShiftedDescriptions skips null API elements and keeps a
// description that already matches the reference with the same key.
func TestSanitizeIndexShiftedDescriptions_EdgeCases(t *testing.T) {
	desc := types.StringValue("d")
	a := testARecordRRSet(t, "a", "1.1.1.1", desc)
	b := testARecordRRSet(t, "b", "2.2.2.2", desc)

	api := []attr.Value{types.ObjectNull(testRRSetAttrTypes())}
	assert.Equal(t, api, sanitizeIndexShiftedDescriptions([]attr.Value{a}, api))

	api = []attr.Value{b, a}
	got := sanitizeIndexShiftedDescriptions([]attr.Value{a, b}, api)
	assert.True(t, got[0].Equal(b))
	assert.True(t, got[1].Equal(a))
}

// preserveRRSetGroupMetadataEmptyFields leaves the target unchanged for a
// non-object or unknown reference description.
func TestPreserveRRSetGroupMetadataEmptyFields_EdgeCases(t *testing.T) {
	metaTypes := testRRSetGroupMetaAttrTypes()
	target := types.ObjectValueMust(metaTypes, map[string]attr.Value{"name": types.StringValue("g"), "description": types.StringNull()})
	ref := types.ObjectValueMust(metaTypes, map[string]attr.Value{"name": types.StringValue("g"), "description": types.StringUnknown()})

	assert.True(t, preserveRRSetGroupMetadataEmptyFields(types.StringValue("x"), target).Equal(target))
	assert.True(t, preserveRRSetGroupMetadataEmptyFields(ref, target).Equal(target))
}

// newRRSetList returns a null list with diagnostics for mistyped elements.
func TestNewRRSetList_TypeMismatch(t *testing.T) {
	list, diags := newRRSetList(types.StringType, []attr.Value{types.Int64Value(1)})
	assert.True(t, diags.HasError())
	assert.True(t, list.IsNull())
}

// Group detection and keys reject nil types, non-object values and groups
// without a known metadata name.
func TestRRSetGroupDetectionAndKeys(t *testing.T) {
	assert.False(t, isRRSetGroupType(nil))
	assert.False(t, isRRSetGroupType(types.StringType))

	group := testRRSetGroupObject(t, "g")
	assert.True(t, isRRSetGroupElements([]attr.Value{types.ObjectNull(testRRSetAttrTypes()), group}))

	nameOnly := map[string]attr.Type{"description": types.StringType}
	metaNoName := types.ObjectValueMust(map[string]attr.Type{"metadata": types.ObjectType{AttrTypes: nameOnly}},
		map[string]attr.Value{"metadata": types.ObjectValueMust(nameOnly, map[string]attr.Value{"description": types.StringNull()})})
	metaNullName := types.ObjectValueMust(map[string]attr.Type{"metadata": types.ObjectType{AttrTypes: testRRSetGroupMetaAttrTypes()}},
		map[string]attr.Value{"metadata": types.ObjectValueMust(testRRSetGroupMetaAttrTypes(),
			map[string]attr.Value{"name": types.StringNull(), "description": types.StringNull()})})

	for name, v := range map[string]attr.Value{
		"not_object":  types.StringValue("x"),
		"no_metadata": types.ObjectValueMust(map[string]attr.Type{"x": types.StringType}, map[string]attr.Value{"x": types.StringNull()}),
		"no_name":     metaNoName,
		"null_name":   metaNullName,
	} {
		t.Run(name, func(t *testing.T) {
			_, ok := rrsetGroupAttrValueKey(v)
			assert.False(t, ok)
		})
	}
}

// rrsetAttrValueKey rejects records whose identity is not fully known, and
// selectedKnownRRSetBranch rejects choices with more than one branch.
func TestRRSetAttrValueKey_EdgeCases(t *testing.T) {
	ttl := types.Int64Value(60)
	twoBranches := testRRSetTypeChoiceObject(t, map[string]attr.Value{
		"a_record": testRRSetBranchObject(t, testARecordAttrTypes(), map[string]attr.Value{
			"name": types.StringValue("a"), "values": testStringListValue(t, "1.1.1.1"),
		}),
		"ns_record": testRRSetBranchObject(t, testNSRecordAttrTypes(), map[string]attr.Value{
			"name": types.StringValue("a"), "values": testStringListValue(t, "ns.example.com"),
		}),
	})

	for name, v := range map[string]attr.Value{
		"not_object":      types.StringValue("x"),
		"null_choice":     covRRSet(t, types.StringNull(), ttl, types.ObjectNull(testRRSetTypeChoiceAttrTypes())),
		"no_branch":       covUnkeyableRRSet(t),
		"two_branches":    covRRSet(t, types.StringNull(), ttl, twoBranches),
		"unknown_name":    covRRSet(t, types.StringNull(), ttl, covNSChoice(t, types.StringUnknown(), testStringListValue(t, "x"))),
		"unknown_lb_name": covLBRRSet(t, types.StringUnknown(), types.StringValue("ns")),
		"unknown_lb_ns":   covLBRRSet(t, types.StringValue("lb"), types.StringUnknown()),
	} {
		t.Run(name, func(t *testing.T) {
			_, ok := rrsetAttrValueKey(v)
			assert.False(t, ok)
		})
	}

	key, ok := rrsetAttrValueKey(covLBRRSet(t, types.StringNull(), types.StringNull()))
	require.True(t, ok)
	assert.Equal(t, "LB:www:/", key)
}

// --- MergeRRSetListForApply ---

// MergeRRSetListForApply handles null, unknown and unindexable inputs
// without leaving unknown values in the result.
func TestMergeRRSetListForApply_EdgeCases(t *testing.T) {
	ctx := context.Background()
	elemType := types.ObjectType{AttrTypes: testRRSetAttrTypes()}
	knownA := testARecordRRSet(t, "a", "1.1.1.1", types.StringValue("x"))
	unknownA := testARecordRRSet(t, "a", "1.1.1.1", types.StringUnknown())

	t.Run("zero_plan_returns_null_list", func(t *testing.T) {
		got, diags := MergeRRSetListForApply(ctx, types.List{}, testRRSetListValue(t, knownA))
		require.False(t, diags.HasError())
		assert.True(t, got.IsNull())
	})
	t.Run("unknown_plan_null_api", func(t *testing.T) {
		got, diags := MergeRRSetListForApply(ctx, types.ListUnknown(elemType), types.ListNull(elemType))
		require.False(t, diags.HasError())
		assert.True(t, got.IsNull())
	})
	t.Run("null_api_keeps_plan", func(t *testing.T) {
		planned := testRRSetListValue(t, unknownA)
		got, diags := MergeRRSetListForApply(ctx, planned, types.ListNull(elemType))
		require.False(t, diags.HasError())
		assert.True(t, got.Equal(planned))
	})
	t.Run("unkeyable_plan_element", func(t *testing.T) {
		planned := testRRSetListValue(t, unknownA, covUnkeyableRRSet(t))
		got, diags := MergeRRSetListForApply(ctx, planned, testRRSetListValue(t, knownA))
		require.False(t, diags.HasError())
		assert.False(t, ValueContainsUnknown(got))
	})
	t.Run("plan_element_missing_from_api", func(t *testing.T) {
		unknownNS := testNSRecordRRSet(t, "x", "ns.example.com", types.StringUnknown())
		planned := testRRSetListValue(t, unknownA, unknownNS)
		got, diags := MergeRRSetListForApply(ctx, planned, testRRSetListValue(t, knownA))
		require.False(t, diags.HasError())
		assert.False(t, ValueContainsUnknown(got))
		assert.Len(t, got.Elements(), 2)
	})
}

// Group apply keeps unkeyable and unmatched planned groups, and merges a
// group whose API form has no rr_set attribute.
func TestMergeRRSetListForApply_RRSetGroup_EdgeCases(t *testing.T) {
	ctx := context.Background()
	groupType := types.ObjectType{AttrTypes: testRRSetGroupWithLBAttrTypes()}
	unknownRec := testLBRecordRRSet(t, "www", "lb", "ns", types.StringNull(), types.StringUnknown())

	planned := types.ListValueMust(groupType, []attr.Value{
		testUnkeyableRRSetGroupObject(t),
		testRRSetGroupObject(t, "g1", unknownRec),
	})
	api := types.ListValueMust(groupType, []attr.Value{testRRSetGroupObject(t, "g2")})
	got, diags := MergeRRSetListForApply(ctx, planned, api)
	require.False(t, diags.HasError())
	assert.Len(t, got.Elements(), 2)
	assert.False(t, ValueContainsUnknown(got))

	metaOnlyTypes := map[string]attr.Type{"metadata": types.ObjectType{AttrTypes: testRRSetGroupMetaAttrTypes()}}
	metaG1 := types.ObjectValueMust(testRRSetGroupMetaAttrTypes(), map[string]attr.Value{
		"name": types.StringValue("g1"), "description": types.StringNull(),
	})
	apiNoRRSet := types.ListValueMust(types.ObjectType{AttrTypes: metaOnlyTypes}, []attr.Value{
		types.ObjectValueMust(metaOnlyTypes, map[string]attr.Value{"metadata": metaG1}),
	})
	planned = types.ListValueMust(groupType, []attr.Value{testRRSetGroupObject(t, "g1", unknownRec)})
	got, diags = MergeRRSetListForApply(ctx, planned, apiNoRRSet)
	require.False(t, diags.HasError())
	assert.Len(t, got.Elements(), 1)
}

// Group apply reports diagnostics and resolves from the API when a nested
// rr_set list cannot be built from API elements of a different type.
func TestMergeRRSetListForApply_RRSetGroup_NestedTypeMismatch(t *testing.T) {
	ctx := context.Background()
	groupType := types.ObjectType{AttrTypes: testRRSetGroupWithLBAttrTypes()}
	rrsetType := types.ObjectType{AttrTypes: testRRSetWithLBAttrTypes()}
	meta := types.ObjectValueMust(testRRSetGroupMetaAttrTypes(), map[string]attr.Value{
		"name": types.StringValue("g1"), "description": types.StringNull(),
	})
	planned := types.ListValueMust(groupType, []attr.Value{
		types.ObjectValueMust(testRRSetGroupWithLBAttrTypes(), map[string]attr.Value{
			"metadata": meta,
			"rr_set":   types.ListUnknown(rrsetType),
		}),
	})

	apiGroupTypes := map[string]attr.Type{
		"metadata": types.ObjectType{AttrTypes: testRRSetGroupMetaAttrTypes()},
		"rr_set":   types.ListType{ElemType: types.StringType},
	}
	api := types.ListValueMust(types.ObjectType{AttrTypes: apiGroupTypes}, []attr.Value{
		types.ObjectValueMust(apiGroupTypes, map[string]attr.Value{
			"metadata": meta,
			"rr_set":   testStringListValue(t, "x"),
		}),
	})

	_, diags := MergeRRSetListForApply(ctx, planned, api)
	assert.True(t, diags.HasError())
}

// --- MergeRRSetListForRead ---

// MergeRRSetListForRead handles a zero prior, a null API list, duplicate API
// keys and unkeyable prior elements.
func TestMergeRRSetListForRead_EdgeCases(t *testing.T) {
	ctx := context.Background()
	elemType := types.ObjectType{AttrTypes: testRRSetAttrTypes()}
	a := testARecordRRSet(t, "a", "1.1.1.1", types.StringNull())

	t.Run("zero_prior_uses_api", func(t *testing.T) {
		got, diags := MergeRRSetListForRead(ctx, types.List{}, testRRSetListValue(t, a))
		require.False(t, diags.HasError())
		require.Len(t, got.Elements(), 1)
		assert.True(t, got.Elements()[0].Equal(a))
	})
	t.Run("null_api_keeps_prior", func(t *testing.T) {
		prior := testRRSetListValue(t, a)
		got, diags := MergeRRSetListForRead(ctx, prior, types.ListNull(elemType))
		require.False(t, diags.HasError())
		assert.True(t, got.Equal(prior))
	})
	t.Run("duplicate_api_keys_keep_prior", func(t *testing.T) {
		prior := testRRSetListValue(t, a)
		got, diags := MergeRRSetListForRead(ctx, prior, testRRSetListValue(t, a, a))
		require.False(t, diags.HasError())
		assert.True(t, got.Equal(prior))
	})
	t.Run("unkeyable_prior_uses_api", func(t *testing.T) {
		api := testRRSetListValue(t, a)
		got, diags := MergeRRSetListForRead(ctx, testRRSetListValue(t, covUnkeyableRRSet(t)), api)
		require.False(t, diags.HasError())
		assert.True(t, got.Equal(api))
	})
}

// Group read keeps prior state for a null API list, and reports diagnostics
// when the API group has no rr_set attribute.
func TestMergeRRSetListForRead_RRSetGroup_EdgeCases(t *testing.T) {
	ctx := context.Background()
	groupType := types.ObjectType{AttrTypes: testRRSetGroupWithLBAttrTypes()}
	rec := testLBRecordRRSet(t, "www", "lb", "ns", types.StringNull(), types.StringNull())
	prior := types.ListValueMust(groupType, []attr.Value{testRRSetGroupObject(t, "g1", rec)})

	got, diags := mergeRRSetGroupListForRead(ctx, groupType, prior, types.ListNull(groupType))
	require.False(t, diags.HasError())
	assert.True(t, got.Equal(prior))

	metaOnlyTypes := map[string]attr.Type{"metadata": types.ObjectType{AttrTypes: testRRSetGroupMetaAttrTypes()}}
	metaG1 := types.ObjectValueMust(testRRSetGroupMetaAttrTypes(), map[string]attr.Value{
		"name": types.StringValue("g1"), "description": types.StringNull(),
	})
	apiNoRRSet := types.ListValueMust(types.ObjectType{AttrTypes: metaOnlyTypes}, []attr.Value{
		types.ObjectValueMust(metaOnlyTypes, map[string]attr.Value{"metadata": metaG1}),
	})
	got, diags = MergeRRSetListForRead(ctx, prior, apiNoRRSet)
	assert.True(t, diags.HasError())
	assert.True(t, got.Equal(prior))
}
