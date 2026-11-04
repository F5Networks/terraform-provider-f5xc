//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package manual_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	ephemeralschema "github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/client"
	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/resource/manual"
)

// TestEphemeralF5XCBlindfoldSecret_Metadata verifies the type name returned by Metadata.
//
// Why: Terraform Plugin Framework requires ephemeral resources to advertise a
// deterministic `<provider>_<resource>` type name so configurations referencing
// `ephemeral "f5xc_blindfold_secret"` resolve to this implementation.
//
// How: Invokes Metadata with `ProviderTypeName: "f5xc"` and asserts `resp.TypeName`.
//
// Parameters:
//   - t: testing context for assertions.
//
// Returns:
//   - None (fails test on assertion mismatch).
func TestEphemeralF5XCBlindfoldSecret_Metadata(t *testing.T) {
	res := manual.NewEphemeralF5XCBlindfoldSecret()
	require.NotNil(t, res)

	var resp ephemeral.MetadataResponse
	res.Metadata(context.Background(), ephemeral.MetadataRequest{
		ProviderTypeName: "f5xc",
	}, &resp)

	assert.Equal(t, "f5xc_blindfold_secret", resp.TypeName)
}

// TestEphemeralF5XCBlindfoldSecret_Schema verifies the schema attributes and sensitivity flags.
//
// Why: The Partner Premier ephemeral resource must mark `content` and `location`
// as sensitive and define all required/computed attributes for Blindfold envelope encryption.
//
// How: Calls Schema on the ephemeral resource and inspects each attribute definition.
//
// Parameters:
//   - t: testing context for assertions.
//
// Returns:
//   - None (fails test on assertion mismatch).
func TestEphemeralF5XCBlindfoldSecret_Schema(t *testing.T) {
	res := manual.NewEphemeralF5XCBlindfoldSecret()
	var resp ephemeral.SchemaResponse
	res.Schema(context.Background(), ephemeral.SchemaRequest{}, &resp)

	assert.False(t, resp.Diagnostics.HasError())
	assert.NotEmpty(t, resp.Schema.Description)
	assert.NotEmpty(t, resp.Schema.MarkdownDescription)
	require.Len(t, resp.Schema.Attributes, 7)

	nsAttr, ok := resp.Schema.Attributes["namespace"].(ephemeralschema.StringAttribute)
	require.True(t, ok)
	assert.True(t, nsAttr.IsRequired())
	assert.NotEmpty(t, nsAttr.Validators)

	policyAttr, ok := resp.Schema.Attributes["policy_name"].(ephemeralschema.StringAttribute)
	require.True(t, ok)
	assert.True(t, policyAttr.IsRequired())
	assert.NotEmpty(t, policyAttr.Validators)

	contentAttr, ok := resp.Schema.Attributes["content"].(ephemeralschema.StringAttribute)
	require.True(t, ok)
	assert.True(t, contentAttr.IsRequired())
	assert.True(t, contentAttr.IsSensitive())

	keyVersionAttr, ok := resp.Schema.Attributes["key_version"].(ephemeralschema.Int64Attribute)
	require.True(t, ok)
	assert.True(t, keyVersionAttr.IsOptional())
	assert.True(t, keyVersionAttr.IsComputed())

	locationAttr, ok := resp.Schema.Attributes["location"].(ephemeralschema.StringAttribute)
	require.True(t, ok)
	assert.True(t, locationAttr.IsComputed())
	assert.True(t, locationAttr.IsSensitive())

	policyIDAttr, ok := resp.Schema.Attributes["policy_id"].(ephemeralschema.Int64Attribute)
	require.True(t, ok)
	assert.True(t, policyIDAttr.IsComputed())

	tenantAttr, ok := resp.Schema.Attributes["tenant"].(ephemeralschema.StringAttribute)
	require.True(t, ok)
	assert.True(t, tenantAttr.IsComputed())
}

// TestEphemeralF5XCBlindfoldSecret_Configure verifies provider client injection and type checking.
//
// Why: Configure must safely ignore nil provider data during early lifecycle phases,
// accept a valid `*client.APIClient`, and emit a clear diagnostic on unexpected types.
//
// How: Exercises Configure across three subtests covering nil, invalid, and valid ProviderData.
//
// Parameters:
//   - t: testing context for subtests and assertions.
//
// Returns:
//   - None (fails test on assertion mismatch).
func TestEphemeralF5XCBlindfoldSecret_Configure(t *testing.T) {
	t.Run("nil-provider-data", func(t *testing.T) {
		res, ok := manual.NewEphemeralF5XCBlindfoldSecret().(ephemeral.EphemeralResourceWithConfigure)
		require.True(t, ok)

		var resp ephemeral.ConfigureResponse
		res.Configure(context.Background(), ephemeral.ConfigureRequest{ProviderData: nil}, &resp)
		assert.False(t, resp.Diagnostics.HasError())
	})

	t.Run("unexpected-provider-data-type", func(t *testing.T) {
		res, ok := manual.NewEphemeralF5XCBlindfoldSecret().(ephemeral.EphemeralResourceWithConfigure)
		require.True(t, ok)

		var resp ephemeral.ConfigureResponse
		res.Configure(context.Background(), ephemeral.ConfigureRequest{ProviderData: "invalid-client"}, &resp)
		require.True(t, resp.Diagnostics.HasError())
		assert.Equal(t, "Unexpected Ephemeral Resource Configure Type", resp.Diagnostics.Errors()[0].Summary())
	})

	t.Run("valid-api-client", func(t *testing.T) {
		res, ok := manual.NewEphemeralF5XCBlindfoldSecret().(ephemeral.EphemeralResourceWithConfigure)
		require.True(t, ok)

		var resp ephemeral.ConfigureResponse
		res.Configure(context.Background(), ephemeral.ConfigureRequest{ProviderData: &client.APIClient{}}, &resp)
		assert.False(t, resp.Diagnostics.HasError())
	})
}

// TestEphemeralF5XCBlindfoldSecret_OpenValidation verifies pre-flight validation in Open.
//
// Why: Open must reject unconfigured provider clients and empty plaintext inputs
// before attempting network calls to the F5XC secret management endpoints.
//
// How: Constructs a `tfsdk.Config` matching the ephemeral schema and invokes Open
// with an unconfigured client and with an empty content attribute.
//
// Parameters:
//   - t: testing context for subtests and assertions.
//
// Returns:
//   - None (fails test on assertion mismatch).
func TestEphemeralF5XCBlindfoldSecret_OpenValidation(t *testing.T) {
	ctx := context.Background()
	res := manual.NewEphemeralF5XCBlindfoldSecret()

	var schemaResp ephemeral.SchemaResponse
	res.Schema(ctx, ephemeral.SchemaRequest{}, &schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())

	buildConfig := func(content string) tfsdk.Config {
		tfType := schemaResp.Schema.Type().TerraformType(ctx)
		rawVal := tftypes.NewValue(tfType, map[string]tftypes.Value{
			"namespace":   tftypes.NewValue(tftypes.String, "system"),
			"policy_name": tftypes.NewValue(tftypes.String, "ves-io-allow-volterra"),
			"content":     tftypes.NewValue(tftypes.String, content),
			"key_version": tftypes.NewValue(tftypes.Number, nil),
			"location":    tftypes.NewValue(tftypes.String, nil),
			"policy_id":   tftypes.NewValue(tftypes.Number, nil),
			"tenant":      tftypes.NewValue(tftypes.String, nil),
		})
		return tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    rawVal,
		}
	}

	t.Run("unconfigured-client", func(t *testing.T) {
		var openResp ephemeral.OpenResponse
		res.Open(ctx, ephemeral.OpenRequest{
			Config: buildConfig("non-empty-input"),
		}, &openResp)

		require.True(t, openResp.Diagnostics.HasError())
		assert.Equal(t, "Provider Not Configured", openResp.Diagnostics.Errors()[0].Summary())
	})

	t.Run("empty-content", func(t *testing.T) {
		configurable, ok := res.(ephemeral.EphemeralResourceWithConfigure)
		require.True(t, ok)

		var cfgResp ephemeral.ConfigureResponse
		configurable.Configure(ctx, ephemeral.ConfigureRequest{
			ProviderData: &client.APIClient{
				RestClient: &client.AuthClient{},
			},
		}, &cfgResp)
		require.False(t, cfgResp.Diagnostics.HasError())

		var openResp ephemeral.OpenResponse
		res.Open(ctx, ephemeral.OpenRequest{
			Config: buildConfig(""),
		}, &openResp)

		require.True(t, openResp.Diagnostics.HasError())
		assert.Equal(t, "Empty content", openResp.Diagnostics.Errors()[0].Summary())
	})
}
