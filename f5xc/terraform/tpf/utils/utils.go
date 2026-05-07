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
	"strings"
	"time"

	googleProtobuf "github.com/gogo/protobuf/types"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

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

// GoMapToObjectMapValue converts a map of Go structs (map[string]T)
// into a TPF MapValue where the elements are ObjectValues.
//
// Arguments:
// - ctx: The context.
// - elementAttributeMap: The schema definition (AttrTypes) for the nested object (the value of the map).
// - goMap: The input map (e.g., map[string]AlertPolicyAlertPolicyLabelMatcher).
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
// Arguments:
// - ctx: The context.
// - attributeMap: The schema definition (AttrTypes) for the object.
// - goStruct: The input Go struct instance to be converted.
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

// GoStructSliceToListValue is a generic function that converts a slice of any Go struct type (T)
// into a TPF ListValue of ObjectValues.
//
// Arguments:
// - ctx: The context.
// - attributeMap: The schema definition (AttrTypes) for the nested object.
// - goSlice: The input slice of Go structs (e.g., []VirtualNetworkStaticRouteType).
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

// PendingInitializersResource is implemented by any resource model that provides pending initializers.
type PendingInitializersResource interface {
	GetObjPendingInitializers() []string
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
	const (
		maxRetries     = 20
		maxBackoff     = 5 * time.Second
		initialBackoff = 1 * time.Second
	)

	backoff := initialBackoff
	time.Sleep(backoff)

	for attempt := 0; attempt <= maxRetries; attempt++ {
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

		if attempt == maxRetries {
			return fmt.Errorf("resource %s still has pending initializers after %d retries: %v", name, maxRetries, pending)
		}

		tflog.Info(ctx, fmt.Sprintf("[INFO] Resource %s has pending initializers: %v. Retrying in %s...", name, pending, backoff))
		time.Sleep(backoff)
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}

	return nil
}
