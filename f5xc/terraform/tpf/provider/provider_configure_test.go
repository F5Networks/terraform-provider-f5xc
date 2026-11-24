//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package provider

import (
	"context"
	"crypto/rand"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/client"
)

const (
	testProviderURL   = "https://127.0.0.1:1/api"
	missingClientDiag = "Unable to Create F5XCProvider API Client"
)

// testSecret returns a random credential value, so the tests never embed
// token, password, key or certificate literals.
func testSecret() string { return rand.Text() }

// providerEnvVars lists every environment variable that Configure reads.
var providerEnvVars = []string{
	"F5XCProvider_URL",
	"F5XCProvider_TEST",
	"F5XCProvider_TENANT",
	"F5XCProvider_VESENV",
	"F5XCProvider_TIMEOUT",
	"F5XCProvider_API_KEY",
	"F5XCProvider_API_CERT",
	"F5XCProvider_API_TOKEN",
	"F5XCProvider_API_CA_CERT",
	"F5XCProvider_API_P12_FILE",
	"VES_P12_PASSWORD",
}

// clearProviderEnv makes the developer's environment invisible to Configure.
// t.Setenv also prevents t.Parallel in the calling test.
func clearProviderEnv(t *testing.T) {
	t.Helper()
	for _, k := range providerEnvVars {
		t.Setenv(k, "")
	}
}

// providerConfig returns a provider config with attrs set and every other
// attribute null.
func providerConfig(t *testing.T, attrs map[string]tftypes.Value) tfsdk.Config {
	t.Helper()
	ctx := context.Background()
	schemaResp := &provider.SchemaResponse{}
	New("test")().Schema(ctx, provider.SchemaRequest{}, schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())

	objType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	require.True(t, ok)

	vals := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, typ := range objType.AttributeTypes {
		if v, set := attrs[name]; set {
			vals[name] = v
			continue
		}
		vals[name] = tftypes.NewValue(typ, nil)
	}
	return tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, vals)}
}

// throttlingType returns the Terraform type of the throttling attribute.
func throttlingType(t *testing.T) tftypes.Object {
	t.Helper()
	ctx := context.Background()
	schemaResp := &provider.SchemaResponse{}
	New("test")().Schema(ctx, provider.SchemaRequest{}, schemaResp)
	objType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	require.True(t, ok)
	tt, ok := objType.AttributeTypes["throttling"].(tftypes.Object)
	require.True(t, ok)
	return tt
}

func runConfigure(t *testing.T, cfg tfsdk.Config) *provider.ConfigureResponse {
	t.Helper()
	resp := &provider.ConfigureResponse{}
	New("test")().Configure(context.Background(), provider.ConfigureRequest{
		Config:           cfg,
		TerraformVersion: "1.9.0",
	}, resp)
	return resp
}

func str(v string) tftypes.Value   { return tftypes.NewValue(tftypes.String, v) }
func boolean(v bool) tftypes.Value { return tftypes.NewValue(tftypes.Bool, v) }

func hasDiagSummary(diags diag.Diagnostics, summary string) bool {
	for _, d := range diags {
		if strings.Contains(d.Summary(), summary) {
			return true
		}
	}
	return false
}

// requireClientShared verifies that a successful Configure exposes one client
// to data sources, resources and ephemeral resources.
func requireClientShared(t *testing.T, resp *provider.ConfigureResponse) {
	t.Helper()
	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
	c, ok := resp.DataSourceData.(*client.APIClient)
	require.True(t, ok, "DataSourceData must be *client.APIClient")
	require.NotNil(t, c)
	assert.Same(t, c, resp.ResourceData)
	assert.Same(t, c, resp.EphemeralResourceData)
}

// requireNoClient verifies that a failed Configure exposes no client.
func requireNoClient(t *testing.T, resp *provider.ConfigureResponse) {
	t.Helper()
	assert.Nil(t, resp.DataSourceData)
	assert.Nil(t, resp.ResourceData)
	assert.Nil(t, resp.EphemeralResourceData)
}

// TestConfigure_RejectsInvalidInput verifies that each invalid setting stops
// Configure with a diagnostic and no client.
func TestConfigure_RejectsInvalidInput(t *testing.T) {
	missingFile := filepath.Join(t.TempDir(), "missing.p12")

	tests := []struct {
		name        string
		env         map[string]string
		attrs       map[string]tftypes.Value
		wantSummary string
	}{
		{
			name:        "unknown url",
			attrs:       map[string]tftypes.Value{"url": tftypes.NewValue(tftypes.String, tftypes.UnknownValue)},
			wantSummary: "Unknown F5XCProvider API url",
		},
		{
			name:        "test env is not a boolean",
			env:         map[string]string{"F5XCProvider_URL": testProviderURL, "F5XCProvider_TEST": "notabool"},
			wantSummary: "Unknown F5XCProvider test",
		},
		{
			name:        "vesenv env is not a boolean",
			env:         map[string]string{"F5XCProvider_URL": testProviderURL, "F5XCProvider_VESENV": "notabool"},
			wantSummary: "Unknown F5XCProvider vesenv",
		},
		{
			name:        "missing url",
			attrs:       map[string]tftypes.Value{"api_token": str(testSecret())},
			wantSummary: "Missing F5XCProvider API url",
		},
		{
			name:        "invalid timeout from config",
			attrs:       map[string]tftypes.Value{"url": str(testProviderURL), "timeout": str("notaduration")},
			wantSummary: missingClientDiag,
		},
		{
			name: "missing p12 file from config",
			env:  map[string]string{"VES_P12_PASSWORD": testSecret()},
			attrs: map[string]tftypes.Value{
				"url":          str(testProviderURL),
				"api_p12_file": str(missingFile),
			},
			wantSummary: missingClientDiag,
		},
		{
			name: "missing p12 file from env",
			env: map[string]string{
				"F5XCProvider_URL":          testProviderURL,
				"F5XCProvider_API_P12_FILE": missingFile,
				"VES_P12_PASSWORD":          testSecret(),
			},
			wantSummary: missingClientDiag,
		},
		{
			name: "invalid throttling backoff",
			attrs: map[string]tftypes.Value{
				"url": str(testProviderURL),
				"throttling": tftypes.NewValue(throttlingType(t), map[string]tftypes.Value{
					"max_retries":         tftypes.NewValue(tftypes.Number, nil),
					"initial_backoff":     str("notaduration"),
					"max_backoff":         tftypes.NewValue(tftypes.String, nil),
					"multiplier":          tftypes.NewValue(tftypes.Number, nil),
					"jitter_ratio":        tftypes.NewValue(tftypes.Number, nil),
					"respect_retry_after": tftypes.NewValue(tftypes.Bool, nil),
				}),
			},
			wantSummary: missingClientDiag,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clearProviderEnv(t)
			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			resp := runConfigure(t, providerConfig(t, tc.attrs))

			require.True(t, resp.Diagnostics.HasError())
			assert.True(t, hasDiagSummary(resp.Diagnostics, tc.wantSummary),
				"want diagnostic %q, got %v", tc.wantSummary, resp.Diagnostics)
			requireNoClient(t, resp)
		})
	}
}

// TestConfigure_BuildsClient verifies that valid settings, from the
// environment, from the config or both, produce one shared client. Each case
// sets a random API token because the client requires one credential.
func TestConfigure_BuildsClient(t *testing.T) {
	caFile := filepath.Join(t.TempDir(), "ca.pem")

	tests := []struct {
		name  string
		env   map[string]string
		attrs map[string]tftypes.Value
	}{
		{
			name: "values from env only",
			env: map[string]string{
				"F5XCProvider_URL":       testProviderURL,
				"F5XCProvider_API_TOKEN": testSecret(),
				"F5XCProvider_TEST":      "false",
				"F5XCProvider_VESENV":    "false",
				"F5XCProvider_TENANT":    "tenant-a",
				"F5XCProvider_TIMEOUT":   "45s",
			},
		},
		{
			name: "config overrides env",
			env: map[string]string{
				"F5XCProvider_URL":     "https://env.invalid/api",
				"F5XCProvider_TIMEOUT": "notaduration",
				"F5XCProvider_VESENV":  "true",
			},
			attrs: map[string]tftypes.Value{
				"url":       str(testProviderURL),
				"test":      boolean(false),
				"vesenv":    boolean(false),
				"tenant":    str("tenant-b"),
				"timeout":   str("10s"),
				"api_token": str(testSecret()),
				"api_key":   str(testSecret()),
				"api_cert":  str(testSecret()),
			},
		},
		{
			name:  "default timeout",
			attrs: map[string]tftypes.Value{"url": str(testProviderURL)},
		},
		{
			name:  "ca cert path from config",
			attrs: map[string]tftypes.Value{"url": str(testProviderURL), "api_ca_cert": str(caFile)},
		},
		{
			name: "ca cert path from env",
			env:  map[string]string{"F5XCProvider_URL": testProviderURL, "F5XCProvider_API_CA_CERT": caFile},
		},
		{
			name: "throttling with every field set",
			attrs: map[string]tftypes.Value{
				"url": str(testProviderURL),
				"throttling": tftypes.NewValue(throttlingType(t), map[string]tftypes.Value{
					"max_retries":         tftypes.NewValue(tftypes.Number, 3),
					"initial_backoff":     str("100ms"),
					"max_backoff":         str("2s"),
					"multiplier":          tftypes.NewValue(tftypes.Number, 1.5),
					"jitter_ratio":        tftypes.NewValue(tftypes.Number, 0.1),
					"respect_retry_after": boolean(false),
				}),
			},
		},
		{
			name: "empty throttling block uses defaults",
			attrs: map[string]tftypes.Value{
				"url": str(testProviderURL),
				"throttling": tftypes.NewValue(throttlingType(t), map[string]tftypes.Value{
					"max_retries":         tftypes.NewValue(tftypes.Number, nil),
					"initial_backoff":     tftypes.NewValue(tftypes.String, nil),
					"max_backoff":         tftypes.NewValue(tftypes.String, nil),
					"multiplier":          tftypes.NewValue(tftypes.Number, nil),
					"jitter_ratio":        tftypes.NewValue(tftypes.Number, nil),
					"respect_retry_after": tftypes.NewValue(tftypes.Bool, nil),
				}),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clearProviderEnv(t)
			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			attrs := map[string]tftypes.Value{"api_token": str(testSecret())}
			for k, v := range tc.attrs {
				attrs[k] = v
			}

			requireClientShared(t, runConfigure(t, providerConfig(t, attrs)))
		})
	}
}

// TestConfigure_InvalidConfigValue verifies that a config that does not
// decode into the provider model stops Configure before it builds a client.
func TestConfigure_InvalidConfigValue(t *testing.T) {
	clearProviderEnv(t)
	cfg := providerConfig(t, nil)
	cfg.Raw = tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"url": tftypes.Number,
	}}, map[string]tftypes.Value{"url": tftypes.NewValue(tftypes.Number, 1)})

	resp := runConfigure(t, cfg)

	require.True(t, resp.Diagnostics.HasError())
	requireNoClient(t, resp)
}

// TestProvider_EphemeralResourcesAndFunctions verifies that the provider
// exposes only the blindfold secret ephemeral resource and no functions.
func TestProvider_EphemeralResourcesAndFunctions(t *testing.T) {
	ctx := context.Background()
	p, ok := New("test")().(*F5XCProvider)
	require.True(t, ok)

	ephemerals := p.EphemeralResources(ctx)
	require.Len(t, ephemerals, 1)
	metaResp := &ephemeral.MetadataResponse{}
	ephemerals[0]().Metadata(ctx, ephemeral.MetadataRequest{ProviderTypeName: "f5xc"}, metaResp)
	assert.Equal(t, "f5xc_blindfold_secret", metaResp.TypeName)

	assert.Empty(t, p.Functions(ctx))
}
