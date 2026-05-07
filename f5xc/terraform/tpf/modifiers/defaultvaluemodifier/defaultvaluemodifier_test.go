// Copyright (c) 2025 F5 Inc. All rights reserved.

package defaultvaluemodifier

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/stretchr/testify/assert"
)

// TestDefaultValueModifier_Description verifies the Description method returns the expected string.
func TestDefaultValueModifier_Description(t *testing.T) {
	m := DefaultValueModifier{Suppressors: []string{"foo"}}
	desc := m.Description(context.Background())
	assert.Contains(t, desc, "Suppresses the default value")
}

// TestDefaultValueModifier_MarkdownDescription verifies MarkdownDescription delegates to Description.
func TestDefaultValueModifier_MarkdownDescription(t *testing.T) {
	m := DefaultValueModifier{Suppressors: []string{"foo"}}
	desc := m.MarkdownDescription(context.Background())
	assert.Equal(t, m.Description(context.Background()), desc)
}

// TestNewDefaultValueModifierInt32 verifies the constructor for Int32 plan modifier.
func TestNewDefaultValueModifierInt32(t *testing.T) {
	m := NewDefaultValueModifierInt32("foo", "bar")
	assert.Implements(t, (*planmodifier.Int32)(nil), m)
	assert.Equal(t, []string{"foo", "bar"}, m.(DefaultValueModifier).Suppressors)
}

// TestNewDefaultValueModifierString verifies the constructor for String plan modifier.
func TestNewDefaultValueModifierString(t *testing.T) {
	m := NewDefaultValueModifierString("foo", "bar")
	assert.Implements(t, (*planmodifier.String)(nil), m)
	assert.Equal(t, []string{"foo", "bar"}, m.(DefaultValueModifier).Suppressors)
}
