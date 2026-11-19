// Copyright (c) 2026 F5, Inc. All rights reserved.

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderSchema_SensitiveAttributes(t *testing.T) {
	p := &F5XCProvider{}
	resp := &provider.SchemaResponse{}
	req := provider.SchemaRequest{}

	p.Schema(context.Background(), req, resp)

	require.False(t, resp.Diagnostics.HasError(), "Schema should not return errors")
	require.NotNil(t, resp.Schema.Attributes, "Schema attributes should not be nil")

	sensitiveAttrs := []string{
		"api_key",
		"api_cert",
		"api_token",
		"api_ca_cert",
		"api_p12_file",
	}

	for _, attrName := range sensitiveAttrs {
		attr, ok := resp.Schema.Attributes[attrName]
		require.True(t, ok, "Attribute %s should be present in provider schema", attrName)
		assert.True(t, attr.IsSensitive(), "Attribute %s must be marked as Sensitive", attrName)
	}
}

func TestProviderMetadata_Version(t *testing.T) {
	for _, tc := range []struct {
		name    string
		version string
	}{
		{name: "default-dev-version", version: "dev"},
		{name: "release-version", version: "2.1.0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := New(tc.version)()
			resp := &provider.MetadataResponse{}
			p.Metadata(context.Background(), provider.MetadataRequest{}, resp)

			assert.Equal(t, "f5xc", resp.TypeName)
			assert.Equal(t, tc.version, resp.Version)
		})
	}
}
