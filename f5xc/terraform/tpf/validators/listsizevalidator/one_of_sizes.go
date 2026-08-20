//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

// Package listsizevalidator provides a Terraform list validator that enforces
// membership in a discrete set of allowed list sizes.
//
// Why: Some proto rules allow non-contiguous list counts (for example 0, 1, 3).
// Built-in listvalidator.Any(...) yields multiple branch diagnostics, which is
// technically correct but noisy. This validator reports a single actionable
// diagnostic while preserving the same constraint semantics.
package listsizevalidator

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.List = oneOfSizesValidator{}

// oneOfSizesValidator enforces that a list attribute's length matches one of a discrete set of allowed sizes.
type oneOfSizesValidator struct {
	allowed []int64
}

// OneOfSizes returns a list validator that accepts only the provided list sizes.
//
// Why: Enforces non-contiguous list count validation rules derived from protobuf schema annotations.
//
// How: Normalizes the allowed size values and checks list length against the allowed set.
//
// Parameters:
//   - allowed: discrete allowed list sizes (for example 0, 1, 3).
//
// Returns:
//   - validator.List: a validator that emits a single diagnostic when the list length is not allowed.
//
// Example:
//
//	"dns_servers": schema.ListAttribute{
//	    Validators: []validator.List{
//	        listsizevalidator.OneOfSizes(0, 1, 3),
//	    },
//	}
func OneOfSizes(allowed ...int64) validator.List {
	normalized := normalizeAllowed(allowed)
	return oneOfSizesValidator{allowed: normalized}
}

// Description returns the human-readable description of the allowed list sizes constraint.
//
// Parameters:
//   - ctx: context.Context for context-aware operations.
//
// Returns:
//   - string: description indicating the allowed list sizes.
func (v oneOfSizesValidator) Description(_ context.Context) string {
	return fmt.Sprintf("list size must be one of: %s", formatAllowed(v.allowed))
}

// MarkdownDescription returns the markdown-formatted description of the allowed list sizes constraint.
//
// Parameters:
//   - ctx: context.Context for context-aware operations.
//
// Returns:
//   - string: markdown description of the constraint.
func (v oneOfSizesValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateList enforces that a configured list length matches one of the allowed values.
//
// Why: Ensures configured list attributes meet schema cardinality rules before making API calls.
//
// How: Checks if the length of the configured list is in the allowed slice.
// Null and unknown values are skipped.
//
// Parameters:
//   - ctx: context.Context for context-aware operations.
//   - req: validator.ListRequest containing the attribute path and list value.
//   - resp: *validator.ListResponse to record validation errors.
func (v oneOfSizesValidator) ValidateList(_ context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if len(v.allowed) == 0 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Validator Configuration",
			"OneOfSizes validator requires at least one non-negative allowed size.",
		)
		return
	}

	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	actual := int64(len(req.ConfigValue.Elements()))
	if contains(v.allowed, actual) {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid Attribute Value",
		fmt.Sprintf("Attribute %s list size must be one of [%s], got: %d", req.Path.String(), formatAllowed(v.allowed), actual),
	)
}

// normalizeAllowed removes negative values and duplicates from allowed sizes, then sorts them ascending.
//
// Parameters:
//   - allowed: raw slice of allowed int64 sizes.
//
// Returns:
//   - []int64: deduplicated, non-negative, sorted slice of allowed sizes.
func normalizeAllowed(allowed []int64) []int64 {
	if len(allowed) == 0 {
		return nil
	}

	seen := make(map[int64]struct{}, len(allowed))
	normalized := make([]int64, 0, len(allowed))
	for _, n := range allowed {
		if n < 0 {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		normalized = append(normalized, n)
	}

	sort.Slice(normalized, func(i, j int) bool { return normalized[i] < normalized[j] })
	return normalized
}

// formatAllowed renders a slice of allowed int64 sizes as a comma-separated string.
//
// Parameters:
//   - allowed: slice of allowed int64 sizes.
//
// Returns:
//   - string: comma-separated list of sizes (e.g. "0, 1, 3").
func formatAllowed(allowed []int64) string {
	if len(allowed) == 0 {
		return ""
	}

	parts := make([]string, 0, len(allowed))
	for _, n := range allowed {
		parts = append(parts, strconv.FormatInt(n, 10))
	}
	return strings.Join(parts, ", ")
}

// contains reports whether target integer appears within the allowed slice.
//
// Parameters:
//   - allowed: slice of allowed int64 sizes.
//   - target: integer value to search for.
//
// Returns:
//   - bool: true if target is in allowed, false otherwise.
func contains(allowed []int64, target int64) bool {
	for _, n := range allowed {
		if n == target {
			return true
		}
	}
	return false
}
