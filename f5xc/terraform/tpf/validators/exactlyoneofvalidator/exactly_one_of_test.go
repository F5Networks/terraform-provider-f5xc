//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

package exactlyoneofvalidator_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/validators/exactlyoneofvalidator"
)

// attrTypes returns a simple attr.Type map for the given keys (all string).
func attrTypes(keys ...string) map[string]attr.Type {
	m := make(map[string]attr.Type, len(keys))
	for _, k := range keys {
		m[k] = types.StringType
	}
	return m
}

// objectWithValues builds a types.Object where the given set keys have a non-null value
// and all other keys (from allKeys) are null.
func objectWithValues(allKeys []string, setKeys map[string]string) types.Object {
	atTypes := attrTypes(allKeys...)
	vals := make(map[string]attr.Value, len(allKeys))
	for _, k := range allKeys {
		if v, ok := setKeys[k]; ok {
			vals[k] = types.StringValue(v)
		} else {
			vals[k] = types.StringNull()
		}
	}
	return types.ObjectValueMust(atTypes, vals)
}

// --- ExactlyOneOf (optional block) ---

func TestExactlyOneOf_ExactlyOneSet(t *testing.T) {
	t.Parallel()

	v := exactlyoneofvalidator.ExactlyOneOf("a", "b", "c")
	obj := objectWithValues([]string{"a", "b", "c"}, map[string]string{"b": "value"})
	req := validator.ObjectRequest{ConfigValue: obj}
	resp := &validator.ObjectResponse{}

	v.ValidateObject(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error when exactly one key is set, got: %s", resp.Diagnostics)
	}
}

func TestExactlyOneOf_NoneSet(t *testing.T) {
	t.Parallel()

	v := exactlyoneofvalidator.ExactlyOneOf("a", "b", "c")
	obj := objectWithValues([]string{"a", "b", "c"}, map[string]string{})
	req := validator.ObjectRequest{ConfigValue: obj}
	resp := &validator.ObjectResponse{}

	v.ValidateObject(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error when no keys are set, got none")
	}
}

func TestExactlyOneOf_MultipleSet(t *testing.T) {
	t.Parallel()

	v := exactlyoneofvalidator.ExactlyOneOf("a", "b", "c")
	obj := objectWithValues([]string{"a", "b", "c"}, map[string]string{"a": "v1", "b": "v2"})
	req := validator.ObjectRequest{ConfigValue: obj}
	resp := &validator.ObjectResponse{}

	v.ValidateObject(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error when multiple keys are set, got none")
	}
}

func TestExactlyOneOf_NullBlock_Optional(t *testing.T) {
	t.Parallel()

	// ExactlyOneOf is optional — null block is acceptable
	v := exactlyoneofvalidator.ExactlyOneOf("a", "b")
	req := validator.ObjectRequest{ConfigValue: types.ObjectNull(attrTypes("a", "b"))}
	resp := &validator.ObjectResponse{}

	v.ValidateObject(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for null block with optional validator, got: %s", resp.Diagnostics)
	}
}

func TestExactlyOneOf_UnknownBlock(t *testing.T) {
	t.Parallel()

	v := exactlyoneofvalidator.ExactlyOneOf("a", "b")
	req := validator.ObjectRequest{ConfigValue: types.ObjectUnknown(attrTypes("a", "b"))}
	resp := &validator.ObjectResponse{}

	v.ValidateObject(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for unknown block, got: %s", resp.Diagnostics)
	}
}

// --- RequiredExactlyOneOf (required block) ---

func TestRequiredExactlyOneOf_ExactlyOneSet(t *testing.T) {
	t.Parallel()

	v := exactlyoneofvalidator.RequiredExactlyOneOf("a", "b")
	obj := objectWithValues([]string{"a", "b"}, map[string]string{"a": "val"})
	req := validator.ObjectRequest{ConfigValue: obj}
	resp := &validator.ObjectResponse{}

	v.ValidateObject(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error when exactly one key is set, got: %s", resp.Diagnostics)
	}
}

func TestRequiredExactlyOneOf_NullBlock_Required(t *testing.T) {
	t.Parallel()

	// RequiredExactlyOneOf — null block must be an error
	v := exactlyoneofvalidator.RequiredExactlyOneOf("a", "b")
	req := validator.ObjectRequest{ConfigValue: types.ObjectNull(attrTypes("a", "b"))}
	resp := &validator.ObjectResponse{}

	v.ValidateObject(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error for null block with required validator, got none")
	}
}

func TestRequiredExactlyOneOf_MultipleSet(t *testing.T) {
	t.Parallel()

	v := exactlyoneofvalidator.RequiredExactlyOneOf("a", "b")
	obj := objectWithValues([]string{"a", "b"}, map[string]string{"a": "v1", "b": "v2"})
	req := validator.ObjectRequest{ConfigValue: obj}
	resp := &validator.ObjectResponse{}

	v.ValidateObject(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected error when multiple keys are set, got none")
	}
}

func TestExactlyOneOf_Description(t *testing.T) {
	t.Parallel()

	v := exactlyoneofvalidator.ExactlyOneOf("a", "b")
	desc := v.Description(context.Background())
	if desc == "" {
		t.Error("expected non-empty description")
	}
}

func TestExactlyOneOf_MarkdownDescription(t *testing.T) {
	t.Parallel()

	v := exactlyoneofvalidator.ExactlyOneOf("a", "b")
	desc := v.MarkdownDescription(context.Background())
	if desc == "" {
		t.Error("expected non-empty markdown description")
	}
}

// TestExactlyOneOf_DescriptionRequired verifies that both descriptions state
// the block requirement and that the markdown list joins three or more keys
// with commas and a final "or".
func TestExactlyOneOf_DescriptionRequired(t *testing.T) {
	ctx := context.Background()
	v := exactlyoneofvalidator.ExactlyOneOfValidator{KeysToCheck: []string{"a", "b", "c"}, Required: true}

	if got, want := v.Description(ctx), "Ensures that exactly one of the following attributes is specified (block is required): [a b c]"; got != want {
		t.Errorf("Description() = %q, want %q", got, want)
	}
	if got, want := v.MarkdownDescription(ctx), "This block is required. Exactly one of the following attributes must be defined: `a`, `b` or `c`."; got != want {
		t.Errorf("MarkdownDescription() = %q, want %q", got, want)
	}
}

// TestExactlyOneOf_KeyMissingFromObject verifies that a key absent from the
// object attributes is ignored and does not count as set.
func TestExactlyOneOf_KeyMissingFromObject(t *testing.T) {
	v := exactlyoneofvalidator.ExactlyOneOfValidator{KeysToCheck: []string{"a", "missing"}}
	obj := objectWithValues([]string{"a"}, map[string]string{"a": "x"})

	resp := &validator.ObjectResponse{}
	v.ValidateObject(context.Background(), validator.ObjectRequest{ConfigValue: obj}, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error: %v", resp.Diagnostics)
	}
}
