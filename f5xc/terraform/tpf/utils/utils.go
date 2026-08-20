//
// Copyright (c) 2025 F5 Inc. All rights reserved.
//

// Package utils contains utility functions for the Terraform Provider Framework (TPF) implementation.
// It includes functions for converting between Go types and TPF types, handling time parsing,
// managing pending initializers, and other common tasks that support the resource and data source implementations.
//
// These utilities help reduce code duplication across the provider by centralizing common patterns and conversions.
// They also handle diagnostics properly, allowing errors to be propagated back to the caller where they can be returned to Terraform.
package utils

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	googleProtobuf "github.com/gogo/protobuf/types"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// PendingInitializersResource is implemented by any resource model that provides pending initializers.
type PendingInitializersResource interface {
	GetObjPendingInitializers() []string
}

// PendingFinalizersResource is implemented by any resource model that provides finalizers.
// GetObjFinalizers is already present on generated DBObject types (e.g. *virtual_k8s.DBObject),
// so no additional extension file is needed.
type PendingFinalizersResource interface {
	GetObjFinalizers() []string
}

// transientErrorSubstrings enumerates lower-cased error message fragments that
// indicate a transient backend condition worth retrying (create or delete).
//
// Why: F5 XC DNS operations have async deployment windows. Immediately after a
// zone is created, its deployment state is not yet populated, so adding an rrset
// (or deleting during a pending change) can fail with HTTP 500/503 or a
// "failed to get zone deployment state" message even though a retry seconds
// later succeeds. These are not permanent failures.
var transientErrorSubstrings = []string{
	"previous dns zone change is pending",
	"failed to get zone deployment state",
	"status code 500",
	"status code 503",
	"status code 409",
	"is still being referred by",
	"still being referred by",
}

// isTransientError reports whether err matches any known transient backend
// failure fragment.
//
// How: it lower-cases the error message and checks it against
// transientErrorSubstrings using substring matching.
//
// Parameters:
//   - err: the error to classify; a nil error is never transient.
//
// Return value:
//   - bool: true when err's message contains a known transient fragment.
func isTransientError(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	for _, pattern := range transientErrorSubstrings {
		if strings.Contains(msg, pattern) {
			return true
		}
	}

	return false
}

type transientDeleteRetryConfig struct {
	maxAttempts   int
	retryInterval time.Duration
	sleep         func(time.Duration)
}

// defaultTransientDeleteRetryConfig returns the production delete-retry config:
// 6 attempts, 5s between attempts (up to ~30s total), using time.Sleep.
//
// Parameters: none.
//
// Return value:
//   - transientDeleteRetryConfig: ready-to-use config with production defaults.
func defaultTransientDeleteRetryConfig() transientDeleteRetryConfig {
	return transientDeleteRetryConfig{
		maxAttempts:   6,
		retryInterval: 5 * time.Second,
		sleep:         time.Sleep,
	}
}

// transientCreateRetryConfig controls create retry behavior; it mirrors
// transientDeleteRetryConfig so the two helpers share identical semantics.
type transientCreateRetryConfig struct {
	maxAttempts   int
	retryInterval time.Duration
	sleep         func(time.Duration)
}

// defaultTransientCreateRetryConfig returns the production create-retry config:
// 6 attempts, 5s between attempts (up to ~30s total), using time.Sleep.
//
// Parameters: none.
//
// Return value:
//   - transientCreateRetryConfig: ready-to-use config with production defaults.
func defaultTransientCreateRetryConfig() transientCreateRetryConfig {
	return transientCreateRetryConfig{
		maxAttempts:   6,
		retryInterval: 5 * time.Second,
		sleep:         time.Sleep,
	}
}

// IsNotFoundError reports whether err represents a not-found response.
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "status code 404") || strings.Contains(msg, "notfound")
}

// IsTransientDeleteError reports whether err matches known transient delete failures.
//
// This is used by retry-transient delete strategy to avoid failing immediately on
// backend propagation windows and temporary deployment-state lookup failures.
//
// Parameters:
//   - err: the error returned by the delete call; a nil error is never transient.
//
// Return value:
//   - bool: true when err represents a retryable transient backend condition.
func IsTransientDeleteError(err error) bool {
	return isTransientError(err)
}

// IsTransientCreateError reports whether err matches known transient create failures.
//
// Why: it is used by the retry-transient create strategy to avoid failing
// immediately when a create is issued during a backend deployment window (e.g.
// creating a DNS rrset right after its zone, before the zone deployment state is
// ready), which surfaces as HTTP 500/503 or "failed to get zone deployment state".
//
// Parameters:
//   - err: the error returned by the create call; a nil error is never transient.
//
// Return value:
//   - bool: true when err represents a retryable transient backend condition.
func IsTransientCreateError(err error) bool {
	return isTransientError(err)
}

// DeleteWithTransientRetry executes a delete operation with retries for known
// transient backend failures.
func DeleteWithTransientRetry(ctx context.Context, resourceName string, deleteFn func() error) error {
	return deleteWithTransientRetryWithConfig(ctx, resourceName, deleteFn, defaultTransientDeleteRetryConfig())
}

// deleteWithTransientRetryWithConfig is the internal implementation of DeleteWithTransientRetry
// that accepts an explicit config, allowing tests to inject zero-duration sleeps and a small
// attempt count.
//
// Parameters:
//   - ctx: context used for cancellation and structured logging.
//   - resourceName: human-readable resource name used in retry log messages.
//   - deleteFn: the delete operation to execute; must not be nil.
//   - cfg: configuration controlling max attempts, retry interval, and the sleep function.
//
// Return value:
//   - error: nil on success or when the resource is already gone (404); otherwise the last
//     (transient, exhausted) or first non-transient error, or an error if deleteFn is nil.
func deleteWithTransientRetryWithConfig(
	ctx context.Context,
	resourceName string,
	deleteFn func() error,
	cfg transientDeleteRetryConfig,
) error {
	if deleteFn == nil {
		return fmt.Errorf("delete operation is nil")
	}

	maxAttempts := cfg.maxAttempts
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	retryInterval := cfg.retryInterval
	if retryInterval < 0 {
		retryInterval = 0
	}

	sleepFn := cfg.sleep
	if sleepFn == nil {
		sleepFn = time.Sleep
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := deleteFn()
		if err == nil {
			return nil
		}
		if IsNotFoundError(err) {
			return nil
		}
		if !IsTransientDeleteError(err) {
			return err
		}
		if attempt == maxAttempts {
			return err
		}

		tflog.Warn(ctx, fmt.Sprintf("Transient delete error for %s (attempt %d/%d): %s. Retrying in %s.", resourceName, attempt, maxAttempts, err, retryInterval))
		sleepFn(retryInterval)
	}

	return nil
}

// CreateWithTransientRetry executes a create operation with retries for known
// transient backend failures.
//
// Why: creating a resource that depends on a just-created parent (e.g. a DNS
// rrset immediately after its zone) can transiently fail while the backend
// finishes deploying the parent. Retrying with a short backoff lets the create
// succeed once the deployment window closes, instead of failing the apply.
//
// How: it delegates to createWithTransientRetryWithConfig with the production
// default config (6 attempts, 5s interval).
//
// Parameters:
//   - ctx: context used for cancellation and structured logging.
//   - resourceName: human-readable resource name used in retry log messages.
//   - createFn: the create operation; it returns the create response message and
//     an error. Must not be nil.
//
// Return value:
//   - proto.Message: the create response from the first successful attempt, or
//     the last attempt's response on failure.
//   - error: nil on success; otherwise the last (transient, exhausted) or first
//     non-transient error, or an error if createFn is nil.
func CreateWithTransientRetry(ctx context.Context, resourceName string, createFn func() (proto.Message, error)) (proto.Message, error) {
	return createWithTransientRetryWithConfig(ctx, resourceName, createFn, defaultTransientCreateRetryConfig())
}

// createWithTransientRetryWithConfig is the internal implementation of CreateWithTransientRetry
// that accepts an explicit config, allowing tests to inject zero-duration sleeps and a small
// attempt count.
//
// Parameters:
//   - ctx: context used for cancellation and structured logging.
//   - resourceName: human-readable resource name used in retry log messages.
//   - createFn: the create operation; returns the response message and an error. Must not be nil.
//   - cfg: configuration controlling max attempts, retry interval, and the sleep function.
//
// Return value:
//   - proto.Message: the create response from the first successful attempt, or the last
//     attempt's response on failure.
//   - error: nil on success; otherwise the last (transient, exhausted) or first non-transient
//     error, or an error if createFn is nil.
func createWithTransientRetryWithConfig(
	ctx context.Context,
	resourceName string,
	createFn func() (proto.Message, error),
	cfg transientCreateRetryConfig,
) (proto.Message, error) {
	if createFn == nil {
		return nil, fmt.Errorf("create operation is nil")
	}

	maxAttempts := cfg.maxAttempts
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	retryInterval := cfg.retryInterval
	if retryInterval < 0 {
		retryInterval = 0
	}

	sleepFn := cfg.sleep
	if sleepFn == nil {
		sleepFn = time.Sleep
	}

	var (
		resp proto.Message
		err  error
	)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err = createFn()
		if err == nil {
			return resp, nil
		}
		if !IsTransientCreateError(err) {
			return resp, err
		}
		if attempt == maxAttempts {
			return resp, err
		}

		tflog.Warn(ctx, fmt.Sprintf("Transient create error for %s (attempt %d/%d): %s. Retrying in %s.", resourceName, attempt, maxAttempts, err, retryInterval))
		sleepFn(retryInterval)
	}

	return resp, err
}

// waitForPendingInitializersConfig holds configuration for WaitForPendingInitializers.
// It is unexported and used only for testing via waitForPendingInitializersWithConfig.
type waitForPendingInitializersConfig struct {
	maxRetries     int
	maxBackoff     time.Duration
	initialBackoff time.Duration
	sleep          func(time.Duration)
}

// waitForPendingFinalizersConfig holds configuration for WaitForPendingFinalizers.
// It is unexported and used only for testing via waitForPendingFinalizersWithConfig.
type waitForPendingFinalizersConfig struct {
	maxRetries     int
	maxBackoff     time.Duration
	initialBackoff time.Duration
	sleep          func(time.Duration)
}

// FormatToAbsolutePath converts a path string into a file URI compatible with the
// volterra stdlib TLS library, which uses "file:///" as a prefix and strips exactly
// 8 characters ("file:///") before passing the remainder to os.ReadFile.
//
// To produce a valid absolute path after stripping, the URI must have the form
// "file:////" + absolute-path-without-leading-slash, i.e. four slashes total,
// so that after stripping "file:///" (8 chars) the result is "/absolute/path".
//
// Parameters:
//   - p: The input path string, which may be empty, a plain absolute path (e.g., "/etc/hosts"),
//     a relative path (e.g., "etc/hosts"), or an existing file URI
//     (e.g., "file:///etc/hosts" or "file:////etc/hosts").
//
// Returns:
//   - A file URI string with four leading slashes (e.g., "file:////etc/hosts") that the
//     stdlib TLS library can correctly resolve to "/etc/hosts".
//     Returns an empty string if the input is empty.
func FormatToAbsolutePath(p string) string {
	if p == "" {
		return ""
	}

	// 1. Strip any existing file:// or file:/// prefix to get the raw path.
	p = strings.TrimPrefix(p, "file://")
	// After TrimPrefix("file://"), a "file:///etc/hosts" becomes "/etc/hosts"
	// and a "file:////etc/hosts" becomes "//etc/hosts" — normalise extra slashes below.
	p = strings.TrimLeft(p, "/")

	// 2. Re-add exactly one leading slash to make it an absolute path.
	p = "/" + p

	// 3. Prepend "file://" + the absolute path (which starts with "/").
	// This yields "file://" + "/absolute/path" = "file:///absolute/path" — but the
	// stdlib TLS library strips "file:///" (8 chars) and gets "absolute/path" (no leading slash).
	// To compensate, we add one extra slash so the library strips "file:///" and is left
	// with "/absolute/path":
	//   "file://" + "/" + "/absolute/path" = "file:////absolute/path"
	//   "file:////absolute/path"[8:]       = "/absolute/path"  ✓
	return "file:///" + p
}

// GetListStringDefaultValue returns a Terraform ListValue containing a single string element.
// This is used for setting default values in schema definitions where diagnostics cannot be propagated.
// Parameters:
//   - defaultValue: The string to use as the default list element.
//
// Returns:
//   - basetypes.ListValue containing the default string.
//   - Panics if the ListValue creation fails (should not occur in normal usage).
func GetListStringDefaultValue(defaultValue string) basetypes.ListValue {
	elements := []attr.Value{types.StringValue(defaultValue)}
	listValue, diagListValue := types.ListValue(types.StringType, elements)
	if diagListValue != nil && diagListValue.HasError() {
		// Option 1: Panic, since this is a framework error and should not occur at runtime
		panic(fmt.Sprintf("Failed to create ListValue for default: %v", diagListValue))
	}
	return listValue
}

// ParseTime parses a RFC3339-formatted string into a protobuf Timestamp.
// Parameters:
//   - t: The RFC3339 time string to parse.
//
// Returns:
//   - Pointer to googleProtobuf.Timestamp representing the parsed time.
//   - Error if parsing fails.
func ParseTime(t string) (*googleProtobuf.Timestamp, error) {
	timeFmt, err := time.Parse(time.RFC3339, t)
	if err != nil {
		return nil, err
	}
	return googleProtobuf.TimestampProto(timeFmt)
}

// ResolveEnumValue maps a Terraform enum string value to the protobuf enum value map.
//
// Parameters:
//   - input: The user-provided enum token from Terraform configuration.
//   - enumMap: The generated protobuf enum value map (e.g., AlertName_value).
//   - allowDisplayName: If true, fallback matching is enabled to map display-name style
//     tokens (e.g., ServiceHealthLow) to enum keys (e.g., SERVICE_HEALTH_LOW).
//
// Returns:
//   - int32 enum value when a match is found.
//   - bool indicating whether mapping succeeded.
func ResolveEnumValue(input string, enumMap map[string]int32, allowDisplayName bool) (int32, bool) {
	if v, ok := enumMap[input]; ok {
		return v, true
	}

	if !allowDisplayName {
		return 0, false
	}

	normalizedInput := normalizeEnumToken(input)
	for key, value := range enumMap {
		if normalizeEnumToken(key) == normalizedInput {
			return value, true
		}
	}

	return 0, false
}

// ResolveEnumStateValue normalizes enum strings written to Terraform state.
//
// When useDisplayName is true, proto enum keys (for example SERVICE_HEALTH_LOW)
// are converted to display-name style tokens (for example ServiceHealthLow) to
// stay consistent with schema validators that accept x-display-name values.
// Non-proto tokens (already display-name style) are returned unchanged.
func ResolveEnumStateValue(input string, useDisplayName bool) string {
	if !useDisplayName || input == "" {
		return input
	}

	if input != strings.ToUpper(input) {
		return input
	}

	parts := strings.Split(input, "_")
	if len(parts) == 0 {
		return input
	}

	knownTokens := map[string]string{
		"API":    "API",
		"CPU":    "CPU",
		"DDOS":   "DDoS",
		"DNS":    "DNS",
		"DOS":    "DoS",
		"GPU":    "GPU",
		"HTTP":   "HTTP",
		"HTTPS":  "HTTPS",
		"ID":     "ID",
		"IP":     "IP",
		"K8S":    "K8S",
		"L4":     "L4",
		"L7":     "L7",
		"L7DDOS": "L7DDoS",
		"OSI":    "OSI",
		"SSL":    "SSL",
		"TCP":    "TCP",
		"TLS":    "TLS",
		"UDP":    "UDP",
		"URI":    "URI",
		"URL":    "URL",
		"UUID":   "UUID",
		"WAF":    "WAF",
	}

	var builder strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}

		if v, ok := knownTokens[part]; ok {
			builder.WriteString(v)
			continue
		}

		lower := strings.ToLower(part)
		builder.WriteByte(byte(strings.ToUpper(lower[:1])[0]))
		if len(lower) > 1 {
			builder.WriteString(lower[1:])
		}
	}

	out := builder.String()
	if out == "" {
		return input
	}

	return out
}

func normalizeEnumToken(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))

	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r + ('a' - 'A'))
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

// Uint32MapToTerraformMap converts a map[uint32]string (API representation) to a Terraform types.Map (state representation).
// Parameters:
//   - in: Input map with uint32 keys and string values.
//
// Returns:
//   - types.Map with string keys and string values, suitable for Terraform state.
//   - diag.Diagnostics containing any errors encountered during conversion.
func Uint32MapToTerraformMap(in map[uint32]string) (types.Map, diag.Diagnostics) {
	if in == nil {
		return types.MapNull(types.StringType), nil
	}

	elements := make(map[string]attr.Value, len(in))
	for k, v := range in {
		// Stringify the uint32 key and wrap the string value
		elements[fmt.Sprintf("%d", k)] = types.StringValue(v)
	}

	// Even though we manually prepared the elements, MapValue returns diagnostics.
	// We capture them here (m, diags) and return them to the caller.
	m, diags := types.MapValue(types.StringType, elements)
	if diags.HasError() {
		// If something went wrong (e.g. internal framework issue),
		// we return the empty map and the error details.
		return types.MapNull(types.StringType), diags
	}

	return m, nil
}

// TerraformMapToUint32Map converts a Terraform types.Map (state representation, string keys) to a map[uint32]string (API representation).
// Terraform maps always have string keys, so this helper parses each key as a uint32 (base 10) — this is required for API fields such as
// http_loadbalancer more_option.custom_errors where the key is an HTTP status code (or status class 3/4/5).
//
// Parameters:
//   - ctx: Context for the conversion (used by ElementsAs for value extraction).
//   - in:  Input Terraform map with string keys and string values. If null or unknown, an empty (nil) result is returned.
//
// Returns:
//   - map[uint32]string suitable for the generated protobuf spec.
//   - diag.Diagnostics containing any conversion errors (invalid key format, value type mismatch, etc.).
func TerraformMapToUint32Map(ctx context.Context, in types.Map) (map[uint32]string, diag.Diagnostics) {
	var diags diag.Diagnostics

	if in.IsNull() || in.IsUnknown() {
		return nil, diags
	}

	// Extract string→string first, then re-key to uint32. This avoids the reflect
	// panic that occurs when framework's ElementsAs is asked to build a map with
	// a non-string key type.
	stringKeyed := make(map[string]string, len(in.Elements()))
	diags.Append(in.ElementsAs(ctx, &stringKeyed, false)...)
	if diags.HasError() {
		return nil, diags
	}

	out := make(map[uint32]string, len(stringKeyed))
	for k, v := range stringKeyed {
		parsed, err := strconv.ParseUint(k, 10, 32)
		if err != nil {
			diags.AddError(
				"Invalid map key",
				fmt.Sprintf("map key %q cannot be parsed as uint32: %s", k, err.Error()),
			)
			return nil, diags
		}
		out[uint32(parsed)] = v
	}
	return out, diags
}

// StringMapToTerraformMap converts a map[string]string (API representation) to a Terraform types.Map (state representation).
// Parameters:
//   - in: Input map with string keys and string values.
//
// Returns:
//   - types.Map with string keys and string values, suitable for Terraform state.
//   - diag.Diagnostics containing any errors encountered during conversion.
func StringMapToTerraformMap(in map[string]string) (types.Map, diag.Diagnostics) {
	if in == nil {
		return types.MapNull(types.StringType), nil
	}

	elements := make(map[string]attr.Value, len(in))
	for k, v := range in {
		// Explicitly wrap each string value
		elements[k] = types.StringValue(v)
	}

	// types.MapValue creates the MapValue from prepared elements
	return types.MapValue(types.StringType, elements)
}

// MergeStringMaps merges two types.Map values (both must have string element type) into a
// single types.Map where override values win over base values for keys that appear in both.
//
// Why: The MapMergeModifier sets plan = prior state on Update to satisfy Terraform Core's
// constraint (plan must equal config or prior state). After the API call the Update handler
// must store a map that (a) preserves backend-injected keys from the prior state and (b)
// reflects the user's new config values. A plain restore of planLabels (= prior state) would
// keep stale user-key values (e.g. "version":"v1" instead of "v2"), causing ImportStateVerify
// to fail because the API already has the new value.
//
// How: Start with all entries from base (prior state, contains backend-injected keys such as
// "ves.io/config-propagation-destination"). Then overwrite with all entries from override
// (config, contains the user's current intent). Keys present only in base are preserved;
// keys present in both use the override value; keys present only in override are added.
// If override is null or unknown the base is returned unchanged so that a user who removes
// all labels still gets the backend-injected keys preserved until the next Read.
//
// Parameters:
//   - base:     The prior-state map (may contain backend-injected keys). Must not be nil.
//   - override: The config map (user-intent values). May be null or unknown.
//
// Returns:
//   - types.Map: the merged map. Returns base unchanged if override is null/unknown.
//     Returns a null map if base is also null/unknown.
//   - diag.Diagnostics: any errors encountered during element extraction or map construction.
func MergeStringMaps(base, override types.Map) (types.Map, diag.Diagnostics) {
	var diags diag.Diagnostics

	// If override is not a known, non-null map, return base unchanged.
	if override.IsNull() || override.IsUnknown() {
		return base, diags
	}

	// If base is null or unknown, return override directly.
	if base.IsNull() || base.IsUnknown() {
		return override, diags
	}

	// Extract base elements.
	baseElems := make(map[string]string)
	diags.Append(base.ElementsAs(context.Background(), &baseElems, false)...)
	if diags.HasError() {
		return base, diags
	}

	// Extract override elements.
	overrideElems := make(map[string]string)
	diags.Append(override.ElementsAs(context.Background(), &overrideElems, false)...)
	if diags.HasError() {
		return base, diags
	}

	// Merge: start with base, then apply override values.
	merged := make(map[string]string, len(baseElems)+len(overrideElems))
	for k, v := range baseElems {
		merged[k] = v
	}
	for k, v := range overrideElems {
		merged[k] = v
	}

	return StringMapToTerraformMap(merged)
}

// ValueContainsUnknown reports whether a Terraform attr.Value is unknown either
// at the top level or anywhere within its nested elements.
//
// Why: plan-preservation paths for Optional+Computed nested objects and lists
// must not write a partially-known planned value back into state after apply.
// Terraform requires all post-apply values to be known, so a list that is known
// at the container level but still contains an unknown child (for example an
// omitted computed `description` field) must be treated as not safe to reuse.
//
// How: check the current value first, then recursively walk supported composite
// Terraform value kinds (object, list, set, map, tuple) until an unknown child
// is found.
//
// Parameters:
//   - value: the Terraform value to inspect. Null values are considered known.
//
// Return value:
//   - bool: true if value itself or any nested child is unknown; otherwise false.
//
// Example:
//   - A `types.List` with known length but an unknown nested attribute returns true.
//   - A `types.Object` with only null/known children returns false.
func ValueContainsUnknown(value attr.Value) bool {
	if value == nil {
		return false
	}

	if value.IsUnknown() {
		return true
	}

	if value.IsNull() {
		return false
	}

	switch v := value.(type) {
	case basetypes.ObjectValue:
		for _, child := range v.Attributes() {
			if ValueContainsUnknown(child) {
				return true
			}
		}
	case basetypes.ListValue:
		for _, child := range v.Elements() {
			if ValueContainsUnknown(child) {
				return true
			}
		}
	case basetypes.SetValue:
		for _, child := range v.Elements() {
			if ValueContainsUnknown(child) {
				return true
			}
		}
	case basetypes.MapValue:
		for _, child := range v.Elements() {
			if ValueContainsUnknown(child) {
				return true
			}
		}
	case basetypes.TupleValue:
		for _, child := range v.Elements() {
			if ValueContainsUnknown(child) {
				return true
			}
		}
	}

	return false
}

// backendDefaultRRSet describes a single backend-injected default RRSet that the
// F5 XC control plane adds automatically and that must never be tracked in
// Terraform state.
//
// Why: Every DNS zone create causes the backend to add a default NS RRSet the
// user never configured. If it were written into state it would either cause
// perpetual drift (Optional) or be silently pinned (Computed). Recognizing the
// exact default records lets us exclude *only* them, so every other backend-side
// change (additions, deletions, value edits of user-declared records) still
// surfaces as normal drift.
//
// Matching is by full identity: record type, record name, TTL, and the
// (order-insensitive) set of rdata values. The `description` field is
// intentionally ignored because it is computed/healed and may be null or empty.
type backendDefaultRRSet struct {
	recordType string
	name       string
	ttl        int64
	values     []string
}

// backendDefaultRRSets is the closed set of backend-injected default RRSets that
// must be excluded from Terraform state. Currently this is limited to the default
// NS record the control plane adds on zone creation (one entry per environment).
var backendDefaultRRSets = []backendDefaultRRSet{
	{recordType: "NS", name: "", ttl: 86400, values: []string{"ns1crt.f5clouddns-test.com"}},
	{recordType: "NS", name: "", ttl: 86400, values: []string{"ns1.f5clouddns.com", "ns2.f5clouddns.com"}},
	{recordType: "NS", name: "", ttl: 86400, values: []string{"ns1staging.f5clouddns-test.com"}},
}

// isBackendDefaultRRSet reports whether the given RRSet object exactly matches one
// of the known backend-injected default RRSets.
//
// How: it extracts the selected record branch (type, name, rdata values) and the
// TTL, then compares them against backendDefaultRRSets using order-insensitive
// value comparison. Any element whose fields cannot be read concretely returns
// false so that ambiguous records are never mistaken for a default.
//
// Parameters:
//   - value: an RRSet list element (expected to be a types.Object).
//
// Return values:
//   - bool: true only when the record matches a known backend default.
func isBackendDefaultRRSet(value attr.Value) bool {
	rrset, ok := value.(types.Object)
	if !ok || rrset.IsNull() || rrset.IsUnknown() {
		return false
	}

	typeChoiceValue, exists := rrset.Attributes()["type_record_set_choice"]
	if !exists {
		return false
	}
	typeChoice, ok := typeChoiceValue.(types.Object)
	if !ok || typeChoice.IsNull() || typeChoice.IsUnknown() {
		return false
	}

	branchName, branchValue, ok := selectedKnownRRSetBranch(typeChoice)
	if !ok {
		return false
	}
	recordType := strings.ToUpper(strings.TrimSuffix(branchName, "_record"))

	recordName := ""
	if nameValue, hasName := branchValue.Attributes()["name"]; hasName {
		nameAttr, isString := nameValue.(types.String)
		if !isString || nameAttr.IsUnknown() {
			return false
		}
		if !nameAttr.IsNull() {
			recordName = nameAttr.ValueString()
		}
	}

	ttlValue, exists := rrset.Attributes()["ttl"]
	if !exists {
		return false
	}
	ttlAttr, ok := ttlValue.(types.Int64)
	if !ok || ttlAttr.IsNull() || ttlAttr.IsUnknown() {
		return false
	}

	recordValues, ok := rrsetBranchStringValues(branchValue)
	if !ok {
		return false
	}

	for _, def := range backendDefaultRRSets {
		if def.recordType == recordType &&
			def.name == recordName &&
			def.ttl == ttlAttr.ValueInt64() &&
			stringSlicesEqualUnordered(def.values, recordValues) {
			return true
		}
	}

	return false
}

// rrsetBranchStringValues extracts the concrete `values` string slice from an
// RRSet record branch (e.g. an NS record's nameservers). It returns ok=false if
// the field is missing, not a string list, or contains null/unknown entries.
func rrsetBranchStringValues(branch types.Object) ([]string, bool) {
	valuesAttr, exists := branch.Attributes()["values"]
	if !exists {
		return nil, false
	}
	list, ok := valuesAttr.(types.List)
	if !ok || list.IsNull() || list.IsUnknown() {
		return nil, false
	}

	result := make([]string, 0, len(list.Elements()))
	for _, elem := range list.Elements() {
		strVal, ok := elem.(types.String)
		if !ok || strVal.IsNull() || strVal.IsUnknown() {
			return nil, false
		}
		result = append(result, strVal.ValueString())
	}

	return result, true
}

// stringSlicesEqualUnordered reports whether two string slices contain the same
// values regardless of order.
func stringSlicesEqualUnordered(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aCopy := append([]string(nil), a...)
	bCopy := append([]string(nil), b...)
	sort.Strings(aCopy)
	sort.Strings(bCopy)
	for i := range aCopy {
		if aCopy[i] != bCopy[i] {
			return false
		}
	}
	return true
}

// filterOutBackendDefaultRRSets returns the given elements minus any that match a
// known backend-injected default RRSet.
func filterOutBackendDefaultRRSets(elements []attr.Value) []attr.Value {
	filtered := make([]attr.Value, 0, len(elements))
	for _, element := range elements {
		if isBackendDefaultRRSet(element) {
			continue
		}
		filtered = append(filtered, element)
	}
	return filtered
}

// newRRSetList builds a typed RRSet list from the given elements, returning a
// null list (not an empty list) when there are no elements so an unconfigured
// attribute normalises to null.
func newRRSetList(elementType attr.Type, elements []attr.Value) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(elements) == 0 {
		return types.ListNull(elementType), diags
	}
	list, listDiags := types.ListValue(elementType, elements)
	diags.Append(listDiags...)
	if diags.HasError() {
		return types.ListNull(elementType), diags
	}
	return list, diags
}

// MergeRRSetListForApply keeps the planned RRSet list shape while replacing
// matching planned RRsets with API-derived values when needed.
//
// Why: F5 XC appends a backend-managed default NS RRSet to `default_rr_set_group`
// during create/update. Terraform still requires the immediate post-apply result
// to match the planned list shape, but planned list elements can contain unknown
// nested values such as computed `description`.
//
// How: If the planned list is null, return null. If it is top-level unknown,
// return the API list with backend-default RRsets excluded. If it is fully known,
// return it unchanged. Otherwise, index the API list by RRSet key `(type,name)`
// and rebuild a result list using the planned element order/cardinality,
// substituting API-derived elements for planned keys present in the API response.
// Because the result follows the planned cardinality, backend-only RRsets (which
// are never in the plan) are naturally excluded.
//
// Parameters:
//   - ctx: context used for typed list construction.
//   - planned: the Terraform planned list value for the RRSet attribute.
//   - api: the API-derived list value built by specToModel.
//
// Return values:
//   - types.List: a list suitable for the immediate post-apply state.
//   - diag.Diagnostics: diagnostics from typed list reconstruction, if any.
//
// Example:
//   - planned: [A www description=unknown]
//   - api:     [A www description=null, NS @ (default)]
//   - result:  [A www description=null]
func MergeRRSetListForApply(ctx context.Context, planned, api types.List) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	elementType := planned.ElementType(ctx)
	if elementType == nil {
		elementType = api.ElementType(ctx)
	}

	if planned.IsNull() {
		return planned, diags
	}

	if planned.IsUnknown() {
		if api.IsNull() || api.IsUnknown() {
			return api, diags
		}
		return newRRSetList(elementType, filterOutBackendDefaultRRSets(api.Elements()))
	}

	if !ValueContainsUnknown(planned) {
		return planned, diags
	}

	if api.IsNull() || api.IsUnknown() {
		return planned, diags
	}

	apiByKey, ok := indexRRSetAttrValues(api.Elements())
	if !ok {
		return planned, diags
	}

	plannedElements := planned.Elements()
	mergedElements := make([]attr.Value, 0, len(plannedElements))
	for _, plannedElement := range plannedElements {
		key, ok := rrsetAttrValueKey(plannedElement)
		if !ok {
			return planned, diags
		}

		if apiElement, exists := apiByKey[key]; exists {
			mergedElements = append(mergedElements, apiElement)
			continue
		}

		mergedElements = append(mergedElements, plannedElement)
	}

	mergedList, mergedDiags := types.ListValue(elementType, mergedElements)
	diags.Append(mergedDiags...)
	if diags.HasError() {
		return planned, diags
	}

	return mergedList, diags
}

// MergeRRSetListForRead reconciles the API RRSet list against prior state so that
// backend-injected default RRsets are excluded while every other backend-side
// change surfaces as normal drift.
//
// Why: The backend adds a default NS RRSet on zone creation. That specific record
// must never enter state (or it would show as perpetual drift). At the same time
// genuine drift — a user-declared record whose value changed, was deleted, or an
// unexpected record the backend added — must be visible so Terraform can
// reconcile it.
//
// How: The API is treated as the source of truth. Backend-default RRsets are
// filtered out first. Remaining records are emitted in prior-state order for
// stability; prior records missing from the API are dropped (out-of-band deletion
// becomes drift); and non-default records present only in the API are appended
// (unexpected backend additions become drift). Matching records take the
// API-derived value so healed fields (e.g. `description`) are applied.
//
// Parameters:
//   - ctx: context used for typed list construction.
//   - prior: the prior Terraform state list value.
//   - api: the API-derived list value built by specToModel.
//
// Return values:
//   - types.List: the reconciled list for state (null when empty).
//   - diag.Diagnostics: diagnostics from typed list reconstruction, if any.
func MergeRRSetListForRead(ctx context.Context, prior, api types.List) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	elementType := prior.ElementType(ctx)
	if elementType == nil {
		elementType = api.ElementType(ctx)
	}

	// Without an API list there is nothing to reconcile; keep prior unchanged so a
	// transient empty response does not wipe state.
	if api.IsNull() || api.IsUnknown() {
		return prior, diags
	}

	apiFiltered := filterOutBackendDefaultRRSets(api.Elements())

	apiByKey, ok := indexRRSetAttrValues(apiFiltered)
	if !ok {
		// The list is not keyable by RRSet identity (for example rr_set_group,
		// whose elements are groups rather than flat RRSets). Default recognition
		// does not apply; preserve prior state unchanged so these lists behave
		// exactly as they did before default-aware reconciliation.
		return prior, diags
	}

	// First read (no prior state): state is the filtered API list.
	if prior.IsNull() || prior.IsUnknown() {
		return newRRSetList(elementType, apiFiltered)
	}

	mergedElements := make([]attr.Value, 0, len(apiFiltered))
	seen := make(map[string]bool, len(apiFiltered))

	// Preserve prior ordering for records that still exist; prior records absent
	// from the API are dropped so out-of-band deletions surface as drift.
	for _, priorElement := range prior.Elements() {
		key, ok := rrsetAttrValueKey(priorElement)
		if !ok {
			return newRRSetList(elementType, apiFiltered)
		}
		if apiElement, exists := apiByKey[key]; exists {
			mergedElements = append(mergedElements, apiElement)
			seen[key] = true
		}
	}

	// Append non-default records the backend added that are not already present so
	// unexpected additions surface as drift instead of being silently hidden.
	for _, apiElement := range apiFiltered {
		key, ok := rrsetAttrValueKey(apiElement)
		if !ok {
			return newRRSetList(elementType, apiFiltered)
		}
		if !seen[key] {
			mergedElements = append(mergedElements, apiElement)
			seen[key] = true
		}
	}

	return newRRSetList(elementType, mergedElements)
}

// indexRRSetAttrValues builds an RRSet lookup keyed by `(type,name)`.
func indexRRSetAttrValues(elements []attr.Value) (map[string]attr.Value, bool) {
	indexed := make(map[string]attr.Value, len(elements))
	for _, element := range elements {
		key, ok := rrsetAttrValueKey(element)
		if !ok {
			return nil, false
		}
		if _, exists := indexed[key]; exists {
			return nil, false
		}
		indexed[key] = element
	}

	return indexed, true
}

// rrsetAttrValueKey extracts the stable RRSet merge key in the form TYPE:name.
func rrsetAttrValueKey(value attr.Value) (string, bool) {
	rrset, ok := value.(types.Object)
	if !ok || rrset.IsNull() || rrset.IsUnknown() {
		return "", false
	}

	typeChoiceValue, exists := rrset.Attributes()["type_record_set_choice"]
	if !exists {
		return "", false
	}

	typeChoice, ok := typeChoiceValue.(types.Object)
	if !ok || typeChoice.IsNull() || typeChoice.IsUnknown() {
		return "", false
	}

	branchName, branchValue, ok := selectedKnownRRSetBranch(typeChoice)
	if !ok {
		return "", false
	}

	recordType := strings.ToUpper(strings.TrimSuffix(branchName, "_record"))
	recordName := ""
	if nameValue, exists := branchValue.Attributes()["name"]; exists {
		nameAttr, ok := nameValue.(types.String)
		if !ok || nameAttr.IsUnknown() {
			return "", false
		}
		if !nameAttr.IsNull() {
			recordName = nameAttr.ValueString()
		}
	}

	return fmt.Sprintf("%s:%s", recordType, recordName), true
}

// selectedKnownRRSetBranch returns the single selected branch in a oneof-style RRSet object.
func selectedKnownRRSetBranch(obj types.Object) (string, types.Object, bool) {
	selectedBranch := ""
	var selectedValue types.Object

	for branchName, branchValue := range obj.Attributes() {
		branchObject, ok := branchValue.(types.Object)
		if !ok || branchObject.IsNull() || branchObject.IsUnknown() {
			continue
		}
		if selectedBranch != "" {
			return "", types.Object{}, false
		}
		selectedBranch = branchName
		selectedValue = branchObject
	}

	if selectedBranch == "" {
		return "", types.Object{}, false
	}

	return selectedBranch, selectedValue, true
}

// GoMapToObjectMapValue converts a map of Go structs (map[string]T)
// into a TPF MapValue where the elements are ObjectValues.
//
// Parameters:
//   - ctx: Context used for reflection-based conversion.
//   - elementAttributeMap: The schema definition (AttrTypes) for the nested object (the value of the map).
//   - goMap: The input map of Go structs (e.g., map[string]AlertPolicyAlertPolicyLabelMatcher).
//
// Returns:
//   - basetypes.MapValue containing the converted ObjectValues keyed by the original map keys.
//     Returns a null MapValue if goMap is empty.
//   - diag.Diagnostics containing any errors encountered during conversion.
func GoMapToObjectMapValue[T any](ctx context.Context, elementAttributeMap map[string]attr.Type, goMap map[string]T) (basetypes.MapValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	// The type of the elements inside the map (Objects)
	objectType := basetypes.ObjectType{AttrTypes: elementAttributeMap}

	if len(goMap) == 0 {
		return basetypes.NewMapNull(objectType), diags
	}

	elements := make(map[string]attr.Value, len(goMap))

	for key, val := range goMap {
		// Convert the struct val to an ObjectValue
		objValue, d := basetypes.NewObjectValueFrom(ctx, elementAttributeMap, val)
		diags.Append(d...)

		if d.HasError() {
			return basetypes.NewMapNull(objectType), diags
		}

		elements[key] = objValue
	}

	// Create the final MapValue
	mapValue, d := basetypes.NewMapValue(objectType, elements)
	diags.Append(d...)

	return mapValue, diags
}

// GoStructToObjectValue converts a single Go struct of any type (T)
// into a TPF ObjectValue using the provided attribute map.
//
// Parameters:
//   - ctx: Context used for reflection-based conversion.
//   - attributeMap: The schema definition (AttrTypes) for the object.
//   - goStruct: The input Go struct instance to be converted.
//
// Returns:
//   - basetypes.ObjectValue representing the converted struct.
//   - diag.Diagnostics containing any errors encountered during conversion.
func GoStructToObjectValue[T any](ctx context.Context, attributeMap map[string]attr.Type, goStruct T) (basetypes.ObjectValue, diag.Diagnostics) {
	// NewObjectValueFrom utilizes reflection to map the struct fields
	// to the TPF ObjectValue based on the tfsdk tags and the attributeMap.
	objectValue, diags := basetypes.NewObjectValueFrom(
		ctx,
		attributeMap,
		goStruct,
	)

	return objectValue, diags
}

// GoStructSliceToSetValue is a generic function that converts a slice of any Go struct type (T)
// into a TPF SetValue of ObjectValues.
//
// Unlike ListValue, SetValue is unordered and compares elements by content (hash), not position.
// This makes it suitable for API responses where element ordering is non-deterministic,
// preventing false Terraform drift detection.
//
// Parameters:
//   - ctx: Context used for reflection-based conversion.
//   - attributeMap: The schema definition (AttrTypes) for the nested object.
//   - goSlice: The input slice of Go structs (e.g., []UserNamespaceRoleType).
//
// Returns:
//   - basetypes.SetValue containing the converted ObjectValues.
//     Returns a null SetValue if conversion of any element fails.
//   - diag.Diagnostics containing any errors encountered during conversion.
func GoStructSliceToSetValue[T any](ctx context.Context, attributeMap map[string]attr.Type, goSlice []T) (basetypes.SetValue, diag.Diagnostics) {

	var diags diag.Diagnostics

	// Prepare TPF Types
	objectType := basetypes.ObjectType{AttrTypes: attributeMap}

	// Initialize the result slice with the required interface type: attr.Value
	setElements := make([]attr.Value, 0, len(goSlice))

	// Iterate and Convert
	for _, element := range goSlice {
		// NewObjectValueFrom safely handles the conversion of the generic struct 'element'
		// into a TPF ObjectValue.
		objectValue, d := basetypes.NewObjectValueFrom(
			ctx,
			attributeMap,
			element,
		)
		diags.Append(d...)

		if d.HasError() {
			// If conversion fails for one element, stop and return the errors
			return basetypes.NewSetNull(objectType), diags
		}

		// Append the ObjectValue, which satisfies the attr.Value interface.
		setElements = append(setElements, objectValue)
	}

	// Create the final TPF SetValue
	setValue, d := basetypes.NewSetValue(objectType, setElements)
	diags.Append(d...)

	return setValue, diags
}

// GoStructSliceToListValue is a generic function that converts a slice of any Go struct type (T)
// into a TPF ListValue of ObjectValues.
//
// Parameters:
//   - ctx: Context used for reflection-based conversion.
//   - attributeMap: The schema definition (AttrTypes) for the nested object.
//   - goSlice: The input slice of Go structs (e.g., []VirtualNetworkStaticRouteType).
//
// Returns:
//   - basetypes.ListValue containing the converted ObjectValues in the same order as the input slice.
//     Returns a null ListValue if conversion of any element fails.
//   - diag.Diagnostics containing any errors encountered during conversion.
func GoStructSliceToListValue[T any](ctx context.Context, attributeMap map[string]attr.Type, goSlice []T) (basetypes.ListValue, diag.Diagnostics) {

	var diags diag.Diagnostics

	// Prepare TPF Types
	objectType := basetypes.ObjectType{AttrTypes: attributeMap}

	// Initialize the result slice with the required interface type: attr.Value
	// This initialization is O(1) and correctly typed.
	listElements := make([]attr.Value, 0, len(goSlice))

	// Iterate and Convert
	for _, element := range goSlice {
		// NewObjectValueFrom safely handles the conversion of the generic struct 'element'
		// into a TPF ObjectValue.
		objectValue, d := basetypes.NewObjectValueFrom(
			ctx,
			attributeMap,
			element, // 'element' is of type T, which is passed directly.
		)
		diags.Append(d...)

		if d.HasError() {
			// If conversion fails for one element, stop and return the errors
			return basetypes.NewListNull(objectType), diags
		}

		// Append the ObjectValue, which satisfies the attr.Value interface.
		listElements = append(listElements, objectValue)
	}

	// Create the final TPF ListValue
	listValue, d := basetypes.NewListValue(objectType, listElements)
	diags.Append(d...)

	return listValue, diags
}

// defaultWaitConfig returns the production configuration for WaitForPendingInitializers.
//
// Returns:
//   - waitForPendingInitializersConfig with production defaults: 20 retries, 5s max backoff,
//     1s initial backoff, and time.Sleep as the sleep function.
func defaultWaitConfig() waitForPendingInitializersConfig {
	return waitForPendingInitializersConfig{
		maxRetries:     20,
		maxBackoff:     5 * time.Second,
		initialBackoff: 1 * time.Second,
		sleep:          time.Sleep,
	}
}

// WaitForPendingInitializers waits for all pending initializers of a resource to complete.
// It polls the resource status with exponential backoff, up to a maximum number of retries.
// If the resource no longer exists (404), it returns an error so the caller can handle cleanup (e.g., clear UID).
// If pending initializers remain after all retries, it returns an error.
// Otherwise, it returns nil when no pending initializers are found.
//
// Parameters:
//   - ctx: Context for logging and cancellation.
//   - name: Name of the resource.
//   - namespace: Namespace of the resource.
//   - uid: UID of the resource (for logging).
//   - getObject: Function to fetch the resource, returns (resource, error).
//
// Returns:
//   - error: nil if initializers are complete, otherwise an error.
func WaitForPendingInitializers[T PendingInitializersResource](
	ctx context.Context,
	name, namespace, uid string,
	getObject func(ctx context.Context, namespace, name string) (T, error),
) error {
	return waitForPendingInitializersWithConfig(ctx, name, namespace, uid, getObject, defaultWaitConfig())
}

// defaultFinalizerWaitConfig returns the production configuration for WaitForPendingFinalizers.
//
// Returns:
//   - waitForPendingFinalizersConfig with production defaults: 20 retries, 5s max backoff,
//     1s initial backoff, and time.Sleep as the sleep function.
func defaultFinalizerWaitConfig() waitForPendingFinalizersConfig {
	return waitForPendingFinalizersConfig{
		maxRetries:     30,
		maxBackoff:     10 * time.Second,
		initialBackoff: 2 * time.Second,
		sleep:          time.Sleep,
	}
}

// WaitForPendingFinalizers waits for all pending finalizers of a resource to complete.
// It polls the resource status with exponential backoff, up to a maximum number of retries.
// The logic is inverted compared to initializers: the resource is considered deleted (finalizers
// complete) when it no longer exists (404). As long as the resource still exists, finalizers
// are still running. If the resource still exists after all retries, it returns an error.
// Otherwise, it returns nil when the resource is gone (404).
//
// Parameters:
//   - ctx: Context for logging and cancellation.
//   - name: Name of the resource.
//   - namespace: Namespace of the resource.
//   - uid: UID of the resource (for logging).
//   - getObject: Function to fetch the resource, returns (resource, error).
//
// Returns:
//   - error: nil if finalizers are complete (resource gone), otherwise an error.
func WaitForPendingFinalizers[T PendingFinalizersResource](
	ctx context.Context,
	name, namespace, uid string,
	getObject func(ctx context.Context, namespace, name string) (T, error),
) error {
	return waitForPendingFinalizersWithConfig(ctx, name, namespace, uid, getObject, defaultFinalizerWaitConfig())
}

// waitForPendingFinalizersWithConfig is the internal implementation that accepts a config,
// allowing tests to inject zero-duration sleeps and a small retry count.
//
// Parameters:
//   - ctx: Context for logging and cancellation.
//   - name: Name of the resource being polled.
//   - namespace: Namespace of the resource.
//   - uid: UID of the resource (used in log and error messages).
//   - getObject: Function to fetch the current state of the resource.
//   - cfg: Configuration controlling retry count, backoff durations, and the sleep function.
//
// Returns:
//   - nil if the resource is gone (404), indicating finalizers have completed.
//   - error if a non-404 error occurs, or if the resource still exists after all retries.
func waitForPendingFinalizersWithConfig[T PendingFinalizersResource](
	ctx context.Context,
	name, namespace, uid string,
	getObject func(ctx context.Context, namespace, name string) (T, error),
	cfg waitForPendingFinalizersConfig,
) error {
	backoff := cfg.initialBackoff
	cfg.sleep(backoff)

	for attempt := 0; attempt <= cfg.maxRetries; attempt++ {
		obj, err := getObject(ctx, namespace, name)
		if err != nil {
			if strings.Contains(err.Error(), "status code 404") {
				// Resource is gone — finalizers have completed successfully.
				return nil
			}
			return fmt.Errorf("error polling resource %q for finalizer completion: %w", uid, err)
		}

		// Resource still exists — finalizers are still running.
		pending := obj.GetObjFinalizers()

		if attempt == cfg.maxRetries {
			return fmt.Errorf("resource %s still exists after %d retries, finalizers may not have completed", name, cfg.maxRetries)
		}

		tflog.Info(ctx, fmt.Sprintf("[INFO] Resource %s (uid: %s) still has pending finalizers: %v. Retrying in %s...", name, uid, pending, backoff))
		cfg.sleep(backoff)
		backoff *= 2
		if backoff > cfg.maxBackoff {
			backoff = cfg.maxBackoff
		}
	}

	return nil
}

// waitForPendingInitializersWithConfig is the internal implementation that accepts a config,
// allowing tests to inject zero-duration sleeps and a small retry count.
//
// Parameters:
//   - ctx: Context for logging and cancellation.
//   - name: Name of the resource being polled.
//   - namespace: Namespace of the resource.
//   - uid: UID of the resource (used in log and error messages).
//   - getObject: Function to fetch the current state of the resource.
//   - cfg: Configuration controlling retry count, backoff durations, and the sleep function.
//
// Returns:
//   - nil if no pending initializers remain.
//   - error if the resource no longer exists (404), a non-404 error occurs, or initializers
//     are still pending after all retries.
func waitForPendingInitializersWithConfig[T PendingInitializersResource](
	ctx context.Context,
	name, namespace, uid string,
	getObject func(ctx context.Context, namespace, name string) (T, error),
	cfg waitForPendingInitializersConfig,
) error {
	backoff := cfg.initialBackoff
	cfg.sleep(backoff)

	for attempt := 0; attempt <= cfg.maxRetries; attempt++ {
		obj, err := getObject(ctx, namespace, name)
		if err != nil {
			if strings.Contains(err.Error(), "status code 404") {
				return fmt.Errorf("resource with uid %s no longer exists: %w", uid, err)
			}
			return fmt.Errorf("error finding resource %q: %w", uid, err)
		}

		pending := obj.GetObjPendingInitializers()
		if len(pending) == 0 {
			return nil
		}

		if attempt == cfg.maxRetries {
			return fmt.Errorf("resource %s still has pending initializers after %d retries: %v", name, cfg.maxRetries, pending)
		}

		tflog.Info(ctx, fmt.Sprintf("[INFO] Resource %s has pending initializers: %v. Retrying in %s...", name, pending, backoff))
		cfg.sleep(backoff)
		backoff *= 2
		if backoff > cfg.maxBackoff {
			backoff = cfg.maxBackoff
		}
	}

	return nil
}
