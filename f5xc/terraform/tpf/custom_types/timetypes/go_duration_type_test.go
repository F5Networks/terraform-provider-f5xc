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

func TestDurationTypeValueFromTerraform(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		in          tftypes.Value
		expectation attr.Value
		expectedErr string
	}{
		"value": {
			in:          tftypes.NewValue(tftypes.String, "72h3m0.5s"),
			expectation: timetypes.NewGoDurationValueFromStringMust("72h3m0.5s"),
		},
		"unknown": {
			in:          tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
			expectation: timetypes.NewGoDurationUnknown(),
		},
		"null": {
			in:          tftypes.NewValue(tftypes.String, nil),
			expectation: timetypes.NewGoDurationNull(),
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

			got, err := timetypes.GoDurationType{}.ValueFromTerraform(ctx, testCase.in)
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

// The types report stable names, value types, and equality only with their own kind.
func TestTypeMetadata(t *testing.T) {
	ctx := context.Background()

	if got := (timetypes.GoDurationType{}).String(); got != "timetypes.GoDurationType" {
		t.Errorf("GoDurationType.String() = %q", got)
	}
	if _, ok := (timetypes.GoDurationType{}).ValueType(ctx).(timetypes.GoDuration); !ok {
		t.Error("GoDurationType.ValueType() is not GoDuration")
	}
	if !(timetypes.GoDurationType{}).Equal(timetypes.GoDurationType{}) {
		t.Error("GoDurationType must equal itself")
	}
	if (timetypes.GoDurationType{}).Equal(basetypes.StringType{}) {
		t.Error("GoDurationType must not equal StringType")
	}
	if _, ok := timetypes.NewGoDurationNull().Type(ctx).(timetypes.GoDurationType); !ok {
		t.Error("GoDuration.Type() is not GoDurationType")
	}

	if got := (timetypes.RFC3339Type{}).String(); got != "timetypes.RFC3339Type" {
		t.Errorf("RFC3339Type.String() = %q", got)
	}
	if _, ok := (timetypes.RFC3339Type{}).ValueType(ctx).(timetypes.RFC3339); !ok {
		t.Error("RFC3339Type.ValueType() is not RFC3339")
	}
	if !(timetypes.RFC3339Type{}).Equal(timetypes.RFC3339Type{}) {
		t.Error("RFC3339Type must equal itself")
	}
	if (timetypes.RFC3339Type{}).Equal(basetypes.StringType{}) {
		t.Error("RFC3339Type must not equal StringType")
	}
	if _, ok := timetypes.NewRFC3339Null().Type(ctx).(timetypes.RFC3339Type); !ok {
		t.Error("RFC3339.Type() is not RFC3339Type")
	}
	if timetypes.NewRFC3339Null().Equal(basetypes.NewStringNull()) {
		t.Error("RFC3339 must not equal a plain StringValue")
	}
}

// ValueFromTerraform returns the underlying string type error for a non-string tftypes value.
func TestValueFromTerraformWrongType(t *testing.T) {
	ctx := context.Background()
	in := tftypes.NewValue(tftypes.Number, 1)

	if _, err := (timetypes.GoDurationType{}).ValueFromTerraform(ctx, in); err == nil {
		t.Error("GoDurationType: expected error for Number input")
	}
	if _, err := (timetypes.RFC3339Type{}).ValueFromTerraform(ctx, in); err == nil {
		t.Error("RFC3339Type: expected error for Number input")
	}
}
