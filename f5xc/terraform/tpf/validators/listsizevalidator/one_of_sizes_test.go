//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

package listsizevalidator_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/validators/listsizevalidator"
)

func TestOneOfSizes(t *testing.T) {
	v := listsizevalidator.OneOfSizes(0, 1, 3)

	tests := []struct {
		name      string
		listVal   types.List
		expectErr bool
	}{
		{
			name:      "null list",
			listVal:   types.ListNull(types.StringType),
			expectErr: false,
		},
		{
			name:      "unknown list",
			listVal:   types.ListUnknown(types.StringType),
			expectErr: false,
		},
		{
			name: "allowed length 0",
			listVal: types.ListValueMust(
				types.StringType,
				[]attr.Value{},
			),
			expectErr: false,
		},
		{
			name: "allowed length 1",
			listVal: types.ListValueMust(
				types.StringType,
				[]attr.Value{types.StringValue("a")},
			),
			expectErr: false,
		},
		{
			name: "disallowed length 2",
			listVal: types.ListValueMust(
				types.StringType,
				[]attr.Value{
					types.StringValue("a"),
					types.StringValue("b"),
				},
			),
			expectErr: true,
		},
		{
			name: "allowed length 3",
			listVal: types.ListValueMust(
				types.StringType,
				[]attr.Value{
					types.StringValue("a"),
					types.StringValue("b"),
					types.StringValue("c"),
				},
			),
			expectErr: false,
		},
		{
			name: "disallowed length 4",
			listVal: types.ListValueMust(
				types.StringType,
				[]attr.Value{
					types.StringValue("a"),
					types.StringValue("b"),
					types.StringValue("c"),
					types.StringValue("d"),
				},
			),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.ListRequest{
				Path:        path.Root("test"),
				ConfigValue: tt.listVal,
			}
			resp := validator.ListResponse{}

			v.ValidateList(context.Background(), req, &resp)

			if tt.expectErr && !resp.Diagnostics.HasError() {
				t.Errorf("expected validation error, got none")
			}
			if !tt.expectErr && resp.Diagnostics.HasError() {
				t.Errorf("expected no error, got: %v", resp.Diagnostics)
			}
		})
	}
}

func TestOneOfSizes_InvalidConfig(t *testing.T) {
	v := listsizevalidator.OneOfSizes() // no allowed sizes

	req := validator.ListRequest{
		Path: path.Root("test"),
		ConfigValue: types.ListValueMust(
			types.StringType,
			[]attr.Value{types.StringValue("a")},
		),
	}
	resp := validator.ListResponse{}

	v.ValidateList(context.Background(), req, &resp)

	if !resp.Diagnostics.HasError() {
		t.Errorf("expected diagnostic error for empty allowed sizes configuration")
	}
}

func TestOneOfSizes_Description(t *testing.T) {
	v := listsizevalidator.OneOfSizes(3, 1, 0, -1, 1) // duplicate, negative, unsorted

	desc := v.Description(context.Background())
	if !strings.Contains(desc, "0, 1, 3") {
		t.Errorf("expected normalized description '0, 1, 3', got %q", desc)
	}

	mDesc := v.MarkdownDescription(context.Background())
	if mDesc != desc {
		t.Errorf("MarkdownDescription = %q, want %q", mDesc, desc)
	}
}
