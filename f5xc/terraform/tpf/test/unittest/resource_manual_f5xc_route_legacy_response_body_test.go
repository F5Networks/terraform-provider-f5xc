//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package unittest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/client"
	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/provider"
	tpfresource "gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/resource"
)

const (
	legacyTestNamespace = "default"
	legacyTestName      = "legacy-route"
)

// directResponsePath is the Terraform path of route_direct_response in the first route.
var directResponsePath = path.Root("routes").AtListIndex(0).
	AtName("route_action_choice").AtName("route_direct_response")

// TestRouteRead_LegacyResponseBodyFallback checks that Read() fills response_body_encoded from the
// hidden legacy response_body, so import of UI/REST/SDKv2 objects plans no diff.
func TestRouteRead_LegacyResponseBodyFallback(t *testing.T) {
	tests := []struct {
		name        string
		apiResponse map[string]any
		want        types.String
	}{
		{
			name:        "legacy-only-is-encoded",
			apiResponse: map[string]any{"response_code": 403, "response_body": "<p>Denied</p>"},
			want:        types.StringValue("string:///PHA+RGVuaWVkPC9wPg=="),
		},
		{
			// The encoded field wins when the API returns both.
			name: "encoded-passthrough",
			apiResponse: map[string]any{
				"response_code":         200,
				"response_body":         "ignored",
				"response_body_encoded": "string:///b2s=",
			},
			want: types.StringValue("string:///b2s="),
		},
		{
			name:        "both-empty-is-null",
			apiResponse: map[string]any{"response_code": 204},
			want:        types.StringNull(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newRouteAPIServer(t, tt.apiResponse)
			// The provider URL carries the /api path, as for a real tenant.
			got := readDirectResponseBodyEncoded(t, srv.URL+"/api")
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestRouteUpgradeState_DropsLegacyResponseBody checks that state written by a provider version
// that exposed response_body still decodes without an UpgradeState implementation.
func TestRouteUpgradeState_DropsLegacyResponseBody(t *testing.T) {
	ctx := context.Background()
	srv, err := providerserver.NewProtocol6WithError(provider.New("test")())()
	require.NoError(t, err)

	priorState := map[string]any{
		"id":        legacyTestNamespace + "/" + legacyTestName,
		"name":      legacyTestName,
		"namespace": legacyTestNamespace,
		"routes": []any{map[string]any{
			"route_action_choice": map[string]any{
				"route_direct_response": map[string]any{
					"response_body":         "ok",
					"response_body_encoded": "string:///b2s=",
					"response_code":         200,
				},
			},
		}},
	}
	raw, err := json.Marshal(priorState)
	require.NoError(t, err)

	resp, err := srv.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{
		TypeName: "f5xc_route",
		Version:  0,
		RawState: &tfprotov6.RawState{JSON: raw},
	})
	require.NoError(t, err)
	require.Empty(t, resp.Diagnostics, "prior state with response_body must decode")
	require.NotNil(t, resp.UpgradedState)

	schemaType := routeSchemaType(t, srv)
	val, err := resp.UpgradedState.Unmarshal(schemaType)
	require.NoError(t, err)

	var encoded string
	require.NoError(t, legacyStringAt(val, &encoded))
	assert.Equal(t, "string:///b2s=", encoded)
}

// newRouteAPIServer serves GET /.../namespaces/<ns>/routes/<name> with one direct-response route.
func newRouteAPIServer(t *testing.T, directResponse map[string]any) *httptest.Server {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"metadata": map[string]any{"name": legacyTestName, "namespace": legacyTestNamespace},
		"system_metadata": map[string]any{
			"uid": "11111111-2222-3333-4444-555555555555",
		},
		"spec": map[string]any{
			"routes": []any{map[string]any{"route_direct_response": directResponse}},
		},
	})
	require.NoError(t, err)

	wantSuffix := fmt.Sprintf("/namespaces/%s/routes/%s", legacyTestNamespace, legacyTestName)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, wantSuffix) {
			http.Error(w, "unexpected request "+r.Method+" "+r.URL.Path, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// readDirectResponseBodyEncoded runs the generated f5xc_route Read() against apiURL and returns
// the response_body_encoded value written to state.
func readDirectResponseBodyEncoded(t *testing.T, apiURL string) types.String {
	t.Helper()
	ctx := context.Background()

	apiClient, err := client.NewVesAPIClient(&client.ProviderConfig{
		Url:      apiURL,
		Timeout:  "10s",
		ApiToken: "unit-test-token",
	})
	require.NoError(t, err)

	r := tpfresource.NewResourceF5XCRoute()
	rc, ok := r.(resource.ResourceWithConfigure)
	require.True(t, ok)
	var cfgResp resource.ConfigureResponse
	rc.Configure(ctx, resource.ConfigureRequest{ProviderData: apiClient}, &cfgResp)
	require.False(t, cfgResp.Diagnostics.HasError(), "%v", cfgResp.Diagnostics)

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError(), "%v", schemaResp.Diagnostics)

	// Import leaves only namespace and name in state before Read().
	state := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}
	require.False(t, state.SetAttribute(ctx, path.Root("namespace"), legacyTestNamespace).HasError())
	require.False(t, state.SetAttribute(ctx, path.Root("name"), legacyTestName).HasError())

	readResp := resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, &readResp)
	require.False(t, readResp.Diagnostics.HasError(), "%v", readResp.Diagnostics)

	var got types.String
	diags := readResp.State.GetAttribute(ctx, directResponsePath.AtName("response_body_encoded"), &got)
	require.False(t, diags.HasError(), "%v", diags)
	return got
}

// routeSchemaType returns the tftypes type of the f5xc_route resource schema from the provider server.
func routeSchemaType(t *testing.T, srv tfprotov6.ProviderServer) tftypes.Type {
	t.Helper()
	resp, err := srv.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	require.NoError(t, err)
	s, ok := resp.ResourceSchemas["f5xc_route"]
	require.True(t, ok)
	return s.ValueType()
}

// legacyStringAt reads routes[0].route_action_choice.route_direct_response.response_body_encoded from val.
func legacyStringAt(val tftypes.Value, out *string) error {
	p := tftypes.NewAttributePath().WithAttributeName("routes").WithElementKeyInt(0).
		WithAttributeName("route_action_choice").WithAttributeName("route_direct_response").
		WithAttributeName("response_body_encoded")
	got, _, err := tftypes.WalkAttributePath(val, p)
	if err != nil {
		return err
	}
	v, ok := got.(tftypes.Value)
	if !ok {
		return fmt.Errorf("unexpected type %T at %s", got, p)
	}
	return v.As(out)
}
