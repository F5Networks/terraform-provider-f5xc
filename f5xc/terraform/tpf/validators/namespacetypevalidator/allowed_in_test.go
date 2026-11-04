//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package namespacetypevalidator

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAllowedIn(t *testing.T) {
	tests := []struct {
		name           string
		allowedTypes   []string
		namespace      string
		expectsInvalid bool
		expectedDetail string
	}{
		{name: "system-allows-system", allowedTypes: []string{"system"}, namespace: "system"},
		{name: "system-rejects-custom", allowedTypes: []string{"system"}, namespace: "custom", expectsInvalid: true, expectedDetail: "the system namespace instead."},
		{name: "shared-allows-shared", allowedTypes: []string{"shared"}, namespace: "shared"},
		{name: "shared-rejects-custom", allowedTypes: []string{"shared"}, namespace: "custom", expectsInvalid: true, expectedDetail: "shared namespace instead."},
		{name: "user-allows-custom", allowedTypes: []string{"user"}, namespace: "custom"},
		{name: "user-allows-empty", allowedTypes: []string{"user"}, namespace: ""},
		{name: "user-rejects-system", allowedTypes: []string{"user"}, namespace: "system", expectsInvalid: true, expectedDetail: "a user-defined namespace instead."},
		{name: "user-rejects-shared", allowedTypes: []string{"user"}, namespace: "shared", expectsInvalid: true, expectedDetail: "a user-defined namespace instead."},
		{name: "shared-user-allows-shared", allowedTypes: []string{"shared", "user"}, namespace: "shared"},
		{name: "shared-user-allows-custom", allowedTypes: []string{"shared", "user"}, namespace: "custom"},
		{name: "shared-user-rejects-system", allowedTypes: []string{"shared", "user"}, namespace: "system", expectsInvalid: true, expectedDetail: `Namespace "system" is not allowed. namespace must belong to one of: shared namespace or a user-defined namespace instead.`},
		{name: "system-user-allows-system", allowedTypes: []string{"system", "user"}, namespace: "system"},
		{name: "system-user-allows-custom", allowedTypes: []string{"system", "user"}, namespace: "custom"},
		{name: "system-user-rejects-shared", allowedTypes: []string{"system", "user"}, namespace: "shared", expectsInvalid: true, expectedDetail: "the system namespace or a user-defined namespace instead."},
		{name: "system-shared-allows-system", allowedTypes: []string{"system", "shared"}, namespace: "system"},
		{name: "system-shared-allows-shared", allowedTypes: []string{"system", "shared"}, namespace: "shared"},
		{name: "system-shared-rejects-custom", allowedTypes: []string{"system", "shared"}, namespace: "custom", expectsInvalid: true, expectedDetail: "the system or shared namespace instead."},
		{name: "all-types-allows-system", allowedTypes: []string{"system", "shared", "user"}, namespace: "system"},
		{name: "all-types-allows-shared", allowedTypes: []string{"system", "shared", "user"}, namespace: "shared"},
		{name: "all-types-allows-custom", allowedTypes: []string{"system", "shared", "user"}, namespace: "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := AllowedIn(tt.allowedTypes...)
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("namespace"),
				ConfigValue: types.StringValue(tt.namespace),
			}, resp)

			if gotInvalid := resp.Diagnostics.HasError(); gotInvalid != tt.expectsInvalid {
				t.Fatalf("validation error = %t, want %t: %v", gotInvalid, tt.expectsInvalid, resp.Diagnostics)
			}
			if tt.expectedDetail != "" && !strings.Contains(resp.Diagnostics[0].Detail(), tt.expectedDetail) {
				t.Fatalf("diagnostic detail = %q, want to contain %q", resp.Diagnostics[0].Detail(), tt.expectedDetail)
			}
		})
	}
}

func TestAllowedInSkipsNullAndUnknown(t *testing.T) {
	v := AllowedIn("system")
	for _, value := range []types.String{types.StringNull(), types.StringUnknown()} {
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{
			Path:        path.Root("namespace"),
			ConfigValue: value,
		}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected validation error for %v: %v", value, resp.Diagnostics)
		}
	}
}
