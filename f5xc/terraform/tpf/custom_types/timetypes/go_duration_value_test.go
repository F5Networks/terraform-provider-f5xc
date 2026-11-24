// Copyright (c) 2025 F5 Inc. All rights reserved.

package timetypes_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr/xattr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/custom_types/timetypes"
)

func TestDuration_Equals(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		currentDuration timetypes.GoDuration
		givenDuration   basetypes.StringValuable
		expectedMatch   bool
	}{
		"not equal - different durations": {
			currentDuration: timetypes.NewGoDurationValueFromStringMust("50s"),
			givenDuration:   timetypes.NewGoDurationValueFromStringMust("50m"),
			expectedMatch:   false,
		},
		"equal - exactly the same string": {
			currentDuration: timetypes.NewGoDurationValueFromStringMust("30h22m33s"),
			givenDuration:   timetypes.NewGoDurationValueFromStringMust("30h22m33s"),
			expectedMatch:   true,
		},
		"equal - same duration expressed differently": {
			currentDuration: timetypes.NewGoDurationValueFromStringMust("3h25m63s"),
			givenDuration:   timetypes.NewGoDurationValueFromStringMust("12363s"),
			expectedMatch:   false,
		},
		"error - not a Duration value": {
			currentDuration: timetypes.NewGoDurationValueFromStringMust("56s"),
			givenDuration:   basetypes.NewStringValue("abcdef"),
			expectedMatch:   false,
		},
	}
	for name, testCase := range testCases {

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			match := testCase.currentDuration.Equal(testCase.givenDuration)

			if testCase.expectedMatch != match {
				t.Errorf("Expected StringSemanticEquals to return: %t, but got: %t", testCase.expectedMatch, match)
			}
		})
	}
}

func TestDurationValidateAttribute(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		Duration      timetypes.GoDuration
		expectedDiags diag.Diagnostics
	}{
		"empty-struct": {
			Duration: timetypes.GoDuration{},
		},
		"null": {
			Duration: timetypes.NewGoDurationNull(),
		},
		"unknown": {
			Duration: timetypes.NewGoDurationUnknown(),
		},
		"valid duration": {
			Duration: timetypes.NewGoDurationValueFromStringMust("42h"),
		},
		"invalid duration": {
			Duration: timetypes.GoDuration{
				StringValue: basetypes.NewStringValue("nope"),
			},
			expectedDiags: diag.Diagnostics{
				diag.NewAttributeErrorDiagnostic(
					path.Root("test"),
					"Invalid Time Duration String Value",
					"A string value was provided that is not a valid Go Time Duration string format. "+
						`A duration string is a sequence of numbers, each with optional fraction and a unit suffix, such as "300ms", "-1.5h" or "2h45m". `+
						`Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".\n\n`+
						"Given Value: nope\n"+
						"Error: time: invalid duration \"nope\"",
				),
			},
		},
	}
	for name, testCase := range testCases {

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resp := xattr.ValidateAttributeResponse{}

			testCase.Duration.ValidateAttribute(
				context.Background(),
				xattr.ValidateAttributeRequest{
					Path: path.Root("test"),
				},
				&resp,
			)

			if diff := cmp.Diff(resp.Diagnostics, testCase.expectedDiags); diff != "" {
				t.Errorf("Unexpected diagnostics (-got, +expected): %s", diff)
			}
		})
	}
}

func TestDurationValidateParameter(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		Duration        timetypes.GoDuration
		expectedFuncErr *function.FuncError
	}{
		"empty-struct": {
			Duration: timetypes.GoDuration{},
		},
		"null": {
			Duration: timetypes.NewGoDurationNull(),
		},
		"unknown": {
			Duration: timetypes.NewGoDurationUnknown(),
		},
		"valid duration": {
			Duration: timetypes.NewGoDurationValueFromStringMust("42h"),
		},
		"invalid duration": {
			Duration: timetypes.GoDuration{
				StringValue: basetypes.NewStringValue("nope"),
			},
			expectedFuncErr: function.NewArgumentFuncError(
				0,
				"Invalid Go Time Duration String Value: "+
					"A string value was provided that is not a valid Go Time Duration string format. "+
					`A duration string is a sequence of numbers, each with optional fraction and a unit suffix, such as "300ms", "-1.5h" or "2h45m". `+
					`Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".\n\n`+
					"Given Value: nope\n"+
					"Error: time: invalid duration \"nope\"",
			),
		},
	}
	for name, testCase := range testCases {

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resp := function.ValidateParameterResponse{}

			testCase.Duration.ValidateParameter(
				context.Background(),
				function.ValidateParameterRequest{
					Position: int64(0),
				},
				&resp,
			)

			if diff := cmp.Diff(resp.Error, testCase.expectedFuncErr); diff != "" {
				t.Errorf("Unexpected diagnostics (-got, +expected): %s", diff)
			}
		})
	}
}

func TestDuration_StringSemanticEquals(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		currentDuration timetypes.GoDuration
		givenDuration   basetypes.StringValuable
		expectedMatch   bool
		expectedDiags   diag.Diagnostics
	}{
		"not equal - different durations": {
			currentDuration: timetypes.NewGoDurationValueFromStringMust("50s"),
			givenDuration:   timetypes.NewGoDurationValueFromStringMust("50m"),
			expectedMatch:   false,
		},
		"equal - exactly the same string": {
			currentDuration: timetypes.NewGoDurationValueFromStringMust("30h22m33s"),
			givenDuration:   timetypes.NewGoDurationValueFromStringMust("30h22m33s"),
			expectedMatch:   true,
		},
		"equal - same duration expressed differently": {
			currentDuration: timetypes.NewGoDurationValueFromStringMust("3h25m63s"),
			givenDuration:   timetypes.NewGoDurationValueFromStringMust("12363s"),
			expectedMatch:   true,
		},
		"error - not a Duration value": {
			currentDuration: timetypes.NewGoDurationValueFromStringMust("56s"),
			givenDuration:   basetypes.NewStringValue("abcdef"),
			expectedMatch:   false,
			expectedDiags: diag.Diagnostics{
				diag.NewErrorDiagnostic(
					"Semantic Equality Check Error",
					"An unexpected value type was received while performing semantic equality checks. "+
						"Please report this to the provider developers.\n\n"+
						"Expected Value Type: timetypes.GoDuration\n"+
						"Got Value Type: basetypes.StringValue",
				),
			},
		},
	}

	for name, testCase := range testCases {

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			match, diags := testCase.currentDuration.StringSemanticEquals(context.Background(), testCase.givenDuration)

			if testCase.expectedMatch != match {
				t.Errorf("Expected StringSemanticEquals to return: %t, but got: %t", testCase.expectedMatch, match)
			}

			if diff := cmp.Diff(diags, testCase.expectedDiags); diff != "" {
				t.Errorf("Unexpected diagnostics (-got, +expected): %s", diff)
			}
		})
	}
}

func TestDuration_ValueDuration(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		Duration         timetypes.GoDuration
		expectedDuration string
		expectedDiags    diag.Diagnostics
	}{
		"Duration string value is null ": {
			Duration: timetypes.NewGoDurationNull(),
			expectedDiags: diag.Diagnostics{
				diag.NewErrorDiagnostic(
					"Go Duration ValueDuration Error",
					"Duration string value is null",
				),
			},
		},
		"Duration string value is unknown ": {
			Duration: timetypes.NewGoDurationUnknown(),
			expectedDiags: diag.Diagnostics{
				diag.NewErrorDiagnostic(
					"Go Duration ValueDuration Error",
					"Duration string value is unknown",
				),
			},
		},
		"valid duration": {
			Duration:         timetypes.NewGoDurationValueFromStringMust("1h1m1s"),
			expectedDuration: "1h1m1s",
		},
	}
	for name, testCase := range testCases {

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			duration, diags := testCase.Duration.ValueGoDuration()
			expectedDuration, _ := time.ParseDuration(testCase.expectedDuration)

			if duration != expectedDuration {
				t.Errorf("Unexpected difference in time.Duration, got: %s, expected: %s", duration, expectedDuration)
			}

			if diff := cmp.Diff(diags, testCase.expectedDiags); diff != "" {
				t.Errorf("Unexpected diagnostics (-got, +expected): %s", diff)
			}
		})
	}
}

// ValueGoDuration and ValueRFC3339Time report an error for a known value that does not parse.
func TestValueConversionParseError(t *testing.T) {
	d := timetypes.GoDuration{StringValue: basetypes.NewStringValue("bad")}
	if _, diags := d.ValueGoDuration(); !diags.HasError() {
		t.Error("ValueGoDuration: expected error")
	}

	r := timetypes.RFC3339{StringValue: basetypes.NewStringValue("bad")}
	if _, diags := r.ValueRFC3339Time(); !diags.HasError() {
		t.Error("ValueRFC3339Time: expected error")
	}
}

// The GoDuration constructors produce null for nil input, known values for valid input, and errors or panics for invalid input.
func TestGoDurationConstructors(t *testing.T) {
	dur := 90 * time.Second

	if got := timetypes.NewGoDurationValue(dur).ValueString(); got != "1m30s" {
		t.Errorf("NewGoDurationValue = %q", got)
	}
	if !timetypes.NewGoDurationPointerValue(nil).IsNull() {
		t.Error("NewGoDurationPointerValue(nil) must be null")
	}
	if got := timetypes.NewGoDurationPointerValue(&dur).ValueString(); got != "1m30s" {
		t.Errorf("NewGoDurationPointerValue = %q", got)
	}

	if v, diags := timetypes.NewGoDurationValueFromString("5m"); diags.HasError() || v.ValueString() != "5m" {
		t.Errorf("NewGoDurationValueFromString valid: %v %v", v, diags)
	}
	if v, diags := timetypes.NewGoDurationValueFromString("bad"); !diags.HasError() || !v.IsUnknown() {
		t.Errorf("NewGoDurationValueFromString invalid: %v %v", v, diags)
	}

	if v, diags := timetypes.NewGoDurationValueFromPointerString(nil); diags.HasError() || !v.IsNull() {
		t.Errorf("NewGoDurationValueFromPointerString(nil): %v %v", v, diags)
	}
	s := "1h"
	if v, diags := timetypes.NewGoDurationValueFromPointerString(&s); diags.HasError() || v.ValueString() != "1h" {
		t.Errorf("NewGoDurationValueFromPointerString: %v %v", v, diags)
	}

	if !timetypes.NewGoDurationValueFromPointerStringMust(nil).IsNull() {
		t.Error("NewGoDurationValueFromPointerStringMust(nil) must be null")
	}
	if got := timetypes.NewGoDurationValueFromPointerStringMust(&s).ValueString(); got != "1h" {
		t.Errorf("NewGoDurationValueFromPointerStringMust = %q", got)
	}

	assertPanics(t, "NewGoDurationValueFromStringMust", func() { timetypes.NewGoDurationValueFromStringMust("bad") })
}

func assertPanics(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s: expected panic", name)
		}
	}()
	fn()
}
