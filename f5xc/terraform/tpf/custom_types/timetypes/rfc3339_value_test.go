// Copyright (c) 2025 F5 Inc. All rights reserved.

package timetypes_test

import (
	"context"
	"testing"
	"time"

	gogo_types "github.com/gogo/protobuf/types"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr/xattr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/custom_types/timetypes"
)

func TestRFC3339_StringSemanticEquals(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		currentRFC3339time timetypes.RFC3339
		givenRFC3339time   basetypes.StringValuable
		expectedMatch      bool
		expectedDiags      diag.Diagnostics
	}{
		"not equal - different dates": {
			currentRFC3339time: timetypes.NewRFC3339ValueMust("2023-07-25T23:43:16Z"),
			givenRFC3339time:   timetypes.NewRFC3339ValueMust("2023-07-26T23:43:16Z"),
			expectedMatch:      false,
		},
		"not equal - different times": {
			currentRFC3339time: timetypes.NewRFC3339ValueMust("2023-07-25T23:43:16Z"),
			givenRFC3339time:   timetypes.NewRFC3339ValueMust("2023-07-25T23:01:16Z"),
			expectedMatch:      false,
		},
		"not equal - different offset times": {
			currentRFC3339time: timetypes.NewRFC3339ValueMust("2023-07-25T23:43:16Z"),
			givenRFC3339time:   timetypes.NewRFC3339ValueMust("2023-07-25T23:43:16+03:00"),
			expectedMatch:      false,
		},
		"not equal - UTC time and local time": {
			currentRFC3339time: timetypes.NewRFC3339ValueMust("2023-07-25T23:43:16Z"),
			givenRFC3339time:   timetypes.NewRFC3339ValueMust("2023-07-25T20:43:16-03:00"),
			expectedMatch:      false,
		},
		"semantically equal - Z suffix and positive zero num offset": {
			currentRFC3339time: timetypes.NewRFC3339ValueMust("2023-07-25T23:43:16Z"),
			givenRFC3339time:   timetypes.NewRFC3339ValueMust("2023-07-25T23:43:16+00:00"),
			expectedMatch:      true,
		},
		"semantically equal - Z suffix and negative zero num offset": {
			currentRFC3339time: timetypes.NewRFC3339ValueMust("2023-07-25T23:43:16Z"),
			givenRFC3339time:   timetypes.NewRFC3339ValueMust("2023-07-25T23:43:16-00:00"),
			expectedMatch:      true,
		},
		"semantically equal - negative zero and positive zero num offset": {
			currentRFC3339time: timetypes.NewRFC3339ValueMust("2023-07-25T23:43:16-00:00"),
			givenRFC3339time:   timetypes.NewRFC3339ValueMust("2023-07-25T23:43:16+00:00"),
			expectedMatch:      true,
		},
		"semantically equal - byte for byte match": {
			currentRFC3339time: timetypes.NewRFC3339ValueMust("2023-07-25T23:43:16Z"),
			givenRFC3339time:   timetypes.NewRFC3339ValueMust("2023-07-25T23:43:16Z"),
			expectedMatch:      true,
		},
		"error - not given RFC3339 value": {
			currentRFC3339time: timetypes.NewRFC3339ValueMust("2023-07-25T23:43:16Z"),
			givenRFC3339time:   basetypes.NewStringValue("0000-00-00T00:00:00-00:00"),
			expectedMatch:      false,
			expectedDiags: diag.Diagnostics{
				diag.NewErrorDiagnostic(
					"Semantic Equality Check Error",
					"An unexpected value type was received while performing semantic equality checks. "+
						"Please report this to the provider developers.\n\n"+
						"Expected Value Type: timetypes.RFC3339\n"+
						"Got Value Type: basetypes.StringValue",
				),
			},
		},
	}
	for name, testCase := range testCases {

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			match, diags := testCase.currentRFC3339time.StringSemanticEquals(context.Background(), testCase.givenRFC3339time)

			if testCase.expectedMatch != match {
				t.Errorf("Expected StringSemanticEquals to return: %t, but got: %t", testCase.expectedMatch, match)
			}

			if diff := cmp.Diff(diags, testCase.expectedDiags); diff != "" {
				t.Errorf("Unexpected diagnostics (-got, +expected): %s", diff)
			}
		})
	}
}

func TestRFC3339ValidateAttribute(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		RFC3339       timetypes.RFC3339
		expectedDiags diag.Diagnostics
	}{
		"empty-struct": {
			RFC3339: timetypes.RFC3339{},
		},
		"null": {
			RFC3339: timetypes.NewRFC3339Null(),
		},
		"unknown": {
			RFC3339: timetypes.NewRFC3339Unknown(),
		},
		"valid RFC3339": {
			RFC3339: timetypes.NewRFC3339ValueMust("2023-07-25T20:43:16+00:00"),
		},
		"valid RFC3339 - Zulu": {
			RFC3339: timetypes.NewRFC3339ValueMust("2023-07-25T20:43:16Z"),
		},
		"valid RFC3339 - UTC Offset": {
			RFC3339: timetypes.NewRFC3339ValueMust("2023-07-25T20:43:16-05:00"),
		},
		"invalid RFC3339 - no date": {
			RFC3339: timetypes.RFC3339{
				StringValue: basetypes.NewStringValue("20:43:16-05:00"),
			},
			expectedDiags: diag.Diagnostics{
				diag.NewAttributeErrorDiagnostic(
					path.Root("test"),
					"Invalid RFC3339 String Value",
					"A string value was provided that is not valid RFC3339 string format.\n\n"+
						"Given Value: 20:43:16-05:00\n"+
						"Error: parsing time \"20:43:16-05:00\" as \"2006-01-02T15:04:05Z07:00\": "+
						"cannot parse \"20:43:16-05:00\" as \"2006\"",
				),
			},
		},
		"invalid RFC3339 - no time": {
			RFC3339: timetypes.RFC3339{
				StringValue: basetypes.NewStringValue("2023-07-25T"),
			},
			expectedDiags: diag.Diagnostics{
				diag.NewAttributeErrorDiagnostic(
					path.Root("test"),
					"Invalid RFC3339 String Value",
					"A string value was provided that is not valid RFC3339 string format.\n\n"+
						"Given Value: 2023-07-25T\n"+
						"Error: parsing time \"2023-07-25T\" as \"2006-01-02T15:04:05Z07:00\": "+
						"cannot parse \"\" as \"15\"",
				),
			},
		},
		"invalid RFC3339 - normal string": {
			RFC3339: timetypes.RFC3339{
				StringValue: basetypes.NewStringValue("notvalidrfc3339"),
			},
			expectedDiags: diag.Diagnostics{
				diag.NewAttributeErrorDiagnostic(
					path.Root("test"),
					"Invalid RFC3339 String Value",
					"A string value was provided that is not valid RFC3339 string format.\n\n"+
						"Given Value: notvalidrfc3339\n"+
						"Error: parsing time \"notvalidrfc3339\" as \"2006-01-02T15:04:05Z07:00\": "+
						"cannot parse \"notvalidrfc3339\" as \"2006\"",
				),
			},
		},
	}
	for name, testCase := range testCases {

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resp := xattr.ValidateAttributeResponse{}

			testCase.RFC3339.ValidateAttribute(
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

func TestRFC3339ValidateParameter(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		RFC3339         timetypes.RFC3339
		expectedFuncErr *function.FuncError
	}{
		"empty-struct": {
			RFC3339: timetypes.RFC3339{},
		},
		"null": {
			RFC3339: timetypes.NewRFC3339Null(),
		},
		"unknown": {
			RFC3339: timetypes.NewRFC3339Unknown(),
		},
		"valid RFC3339": {
			RFC3339: timetypes.NewRFC3339ValueMust("2023-07-25T20:43:16+00:00"),
		},
		"valid RFC3339 - Zulu": {
			RFC3339: timetypes.NewRFC3339ValueMust("2023-07-25T20:43:16Z"),
		},
		"valid RFC3339 - UTC Offset": {
			RFC3339: timetypes.NewRFC3339ValueMust("2023-07-25T20:43:16-05:00"),
		},
		"invalid RFC3339 - no date": {
			RFC3339: timetypes.RFC3339{
				StringValue: basetypes.NewStringValue("20:43:16-05:00"),
			},
			expectedFuncErr: function.NewArgumentFuncError(
				0,
				"Invalid RFC3339 String Value: "+
					"A string value was provided that is not valid RFC3339 string format.\n\n"+
					"Given Value: 20:43:16-05:00\n"+
					"Error: parsing time \"20:43:16-05:00\" as \"2006-01-02T15:04:05Z07:00\": "+
					"cannot parse \"20:43:16-05:00\" as \"2006\"",
			),
		},
		"invalid RFC3339 - no time": {
			RFC3339: timetypes.RFC3339{
				StringValue: basetypes.NewStringValue("2023-07-25T"),
			},
			expectedFuncErr: function.NewArgumentFuncError(
				0,
				"Invalid RFC3339 String Value: "+
					"A string value was provided that is not valid RFC3339 string format.\n\n"+
					"Given Value: 2023-07-25T\n"+
					"Error: parsing time \"2023-07-25T\" as \"2006-01-02T15:04:05Z07:00\": "+
					"cannot parse \"\" as \"15\"",
			),
		},
		"invalid RFC3339 - normal string": {
			RFC3339: timetypes.RFC3339{
				StringValue: basetypes.NewStringValue("notvalidrfc3339"),
			},
			expectedFuncErr: function.NewArgumentFuncError(
				0,
				"Invalid RFC3339 String Value: "+
					"A string value was provided that is not valid RFC3339 string format.\n\n"+
					"Given Value: notvalidrfc3339\n"+
					"Error: parsing time \"notvalidrfc3339\" as \"2006-01-02T15:04:05Z07:00\": "+
					"cannot parse \"notvalidrfc3339\" as \"2006\"",
			),
		},
	}
	for name, testCase := range testCases {

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resp := function.ValidateParameterResponse{}

			testCase.RFC3339.ValidateParameter(
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

func TestRFC3339_ValueRFC3339Time(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		RFC3339           timetypes.RFC3339
		expectedTimestamp string
		expectedDiags     diag.Diagnostics
	}{
		"RFC3339 string value is null ": {
			RFC3339: timetypes.NewRFC3339Null(),
			expectedDiags: diag.Diagnostics{
				diag.NewErrorDiagnostic(
					"RFC3339 ValueRFC3339Time Error",
					"RFC3339 string value is null",
				),
			},
		},
		"RFC3339 string value is unknown ": {
			RFC3339: timetypes.NewRFC3339Unknown(),
			expectedDiags: diag.Diagnostics{
				diag.NewErrorDiagnostic(
					"RFC3339 ValueRFC3339Time Error",
					"RFC3339 string value is unknown",
				),
			},
		},
		"valid RFC3339 Timestamp - Zulu suffix": {
			RFC3339:           timetypes.NewRFC3339ValueMust("2023-07-25T23:43:16Z"),
			expectedTimestamp: "2023-07-25T23:43:16Z",
		},
		"valid RFC3339 Timestamp - UTC offset ": {
			RFC3339:           timetypes.NewRFC3339ValueMust("2023-07-25T23:43:16-00:00"),
			expectedTimestamp: "2023-07-25T23:43:16-00:00",
		},
		"valid RFC3339 Timestamp - EDT offset ": {
			RFC3339:           timetypes.NewRFC3339ValueMust("2023-07-25T23:43:16-04:00"),
			expectedTimestamp: "2023-07-25T23:43:16-04:00",
		},
	}
	for name, testCase := range testCases {

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rfc3339Time, diags := testCase.RFC3339.ValueRFC3339Time()
			expectedRFC3339Time, _ := time.Parse(time.RFC3339, testCase.expectedTimestamp)

			if rfc3339Time != expectedRFC3339Time {
				t.Errorf("Unexpected difference in time.Time, got: %s, expected: %s", rfc3339Time, expectedRFC3339Time)
			}

			if diff := cmp.Diff(diags, testCase.expectedDiags); diff != "" {
				t.Errorf("Unexpected diagnostics (-got, +expected): %s", diff)
			}
		})
	}
}

// RFC3339.ValidateParameter skips the empty and structural-nil strings.
func TestRFC3339ValidateParameterSkipsNilStrings(t *testing.T) {
	for _, s := range []string{"", "(timestamp: nil Timestamp)"} {
		resp := &function.ValidateParameterResponse{}
		timetypes.RFC3339{StringValue: basetypes.NewStringValue(s)}.ValidateParameter(context.Background(), function.ValidateParameterRequest{}, resp)
		if resp.Error != nil {
			t.Errorf("%q: unexpected error %v", s, resp.Error)
		}
	}
}

// The RFC3339 constructors produce null for nil input, known values for valid input, and errors or panics for invalid input.
func TestRFC3339Constructors(t *testing.T) {
	ts := time.Date(2023, 7, 25, 20, 43, 16, 0, time.UTC)
	const want = "2023-07-25T20:43:16Z"

	if got := timetypes.NewRFC3339TimeValue(ts).ValueString(); got != want {
		t.Errorf("NewRFC3339TimeValue = %q", got)
	}
	if !timetypes.NewRFC3339TimePointerValue(nil).IsNull() {
		t.Error("NewRFC3339TimePointerValue(nil) must be null")
	}
	if got := timetypes.NewRFC3339TimePointerValue(&ts).ValueString(); got != want {
		t.Errorf("NewRFC3339TimePointerValue = %q", got)
	}

	if !timetypes.NewRFC3339ProtobufPointerValue(nil).IsNull() {
		t.Error("NewRFC3339ProtobufPointerValue(nil) must be null")
	}
	pb := &gogo_types.Timestamp{Seconds: ts.Unix()}
	if got := timetypes.NewRFC3339ProtobufPointerValue(pb).ValueString(); got != want {
		t.Errorf("NewRFC3339ProtobufPointerValue = %q", got)
	}

	if v, diags := timetypes.NewRFC3339Value(want); diags.HasError() || v.ValueString() != want {
		t.Errorf("NewRFC3339Value valid: %v %v", v, diags)
	}
	if v, diags := timetypes.NewRFC3339Value("bad"); !diags.HasError() || !v.IsUnknown() {
		t.Errorf("NewRFC3339Value invalid: %v %v", v, diags)
	}

	if v, diags := timetypes.NewRFC3339PointerValue(nil); diags.HasError() || !v.IsNull() {
		t.Errorf("NewRFC3339PointerValue(nil): %v %v", v, diags)
	}
	s := want
	if v, diags := timetypes.NewRFC3339PointerValue(&s); diags.HasError() || v.ValueString() != want {
		t.Errorf("NewRFC3339PointerValue: %v %v", v, diags)
	}

	if !timetypes.NewRFC3339PointerValueMust(nil).IsNull() {
		t.Error("NewRFC3339PointerValueMust(nil) must be null")
	}
	if got := timetypes.NewRFC3339PointerValueMust(&s).ValueString(); got != want {
		t.Errorf("NewRFC3339PointerValueMust = %q", got)
	}

	assertPanics(t, "NewRFC3339ValueMust", func() { timetypes.NewRFC3339ValueMust("bad") })
}
