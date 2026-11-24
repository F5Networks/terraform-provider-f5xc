// Copyright (c) 2025 F5 Inc. All rights reserved.

package timetypes_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/custom_types/timetypes"
)

func TestRFC3339TypeValueFromTerraform(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		in          tftypes.Value
		expectation attr.Value
		expectedErr string
	}{
		"true": {
			in:          tftypes.NewValue(tftypes.String, "2023-07-25T20:43:16+00:00"),
			expectation: timetypes.NewRFC3339ValueMust("2023-07-25T20:43:16+00:00"),
		},
		"unknown": {
			in:          tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
			expectation: timetypes.NewRFC3339Unknown(),
		},
		"null": {
			in:          tftypes.NewValue(tftypes.String, nil),
			expectation: timetypes.NewRFC3339Null(),
		},
		"wrongType": {
			in:          tftypes.NewValue(tftypes.Number, 123),
			expectedErr: "can't unmarshal tftypes.Number into *string, expected string",
		},
	}
	for name, testCase := range testCases {

		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			got, err := timetypes.RFC3339Type{}.ValueFromTerraform(ctx, testCase.in)
			if err != nil {
				if testCase.expectedErr == "" {
					t.Fatalf("Unexpected error: %s", err)
				}
				if testCase.expectedErr != err.Error() {
					t.Fatalf("Expected error to be %q, got %q", testCase.expectedErr, err.Error())
				}
				return
			}
			if err == nil && testCase.expectedErr != "" {
				t.Fatalf("Expected error to be %q, didn't get an error", testCase.expectedErr)
			}
			if !got.Equal(testCase.expectation) {
				t.Errorf("Expected %+v, got %+v", testCase.expectation, got)
			}
			if testCase.expectation.IsNull() != testCase.in.IsNull() {
				t.Errorf("Expected null-ness match: expected %t, got %t", testCase.expectation.IsNull(), testCase.in.IsNull())
			}
			if testCase.expectation.IsUnknown() != !testCase.in.IsKnown() {
				t.Errorf("Expected unknown-ness match: expected %t, got %t", testCase.expectation.IsUnknown(), !testCase.in.IsKnown())
			}
		})
	}
}

// RFC3339Type.ValueFromTerraform rejects a string that is not RFC 3339.
func TestRFC3339ValueFromTerraformInvalid(t *testing.T) {
	_, err := (timetypes.RFC3339Type{}).ValueFromTerraform(context.Background(), tftypes.NewValue(tftypes.String, "not-a-time"))
	if err == nil {
		t.Error("expected error for invalid RFC3339 string")
	}
}

// RFC3339Type.ValueFromString maps empty and structural-nil strings to null, and rejects invalid strings with a null value.
func TestRFC3339ValueFromStringSpecialCases(t *testing.T) {
	ctx := context.Background()
	typ := timetypes.RFC3339Type{}

	for _, s := range []string{"", "(timestamp: nil Timestamp)"} {
		v, diags := typ.ValueFromString(ctx, basetypes.NewStringValue(s))
		if diags.HasError() {
			t.Errorf("%q: unexpected diagnostics %v", s, diags)
		}
		if !v.(timetypes.RFC3339).IsNull() {
			t.Errorf("%q: expected null value", s)
		}
	}

	v, diags := typ.ValueFromString(ctx, basetypes.NewStringValue("bad"))
	if !diags.HasError() {
		t.Error("expected error diagnostic for invalid string")
	}
	if !v.(timetypes.RFC3339).IsNull() {
		t.Error("expected null value for invalid string")
	}
}
