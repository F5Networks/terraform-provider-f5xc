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

	"github.com/gogo/protobuf/proto"
	googleProtobuf "github.com/gogo/protobuf/types"
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
