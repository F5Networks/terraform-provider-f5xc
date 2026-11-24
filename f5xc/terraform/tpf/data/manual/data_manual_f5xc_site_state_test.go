//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package manual

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSiteState_Metadata verifies that the type name is the provider type name
// plus "_site_state".
func TestSiteState_Metadata(t *testing.T) {
	var resp datasource.MetadataResponse
	NewDataSourceF5XCSiteState().Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "f5xc"}, &resp)
	assert.Equal(t, "f5xc_site_state", resp.TypeName)
}

// TestSiteState_Schema verifies the attribute types and flags, and the nested
// node attributes. name is the only required input.
func TestSiteState_Schema(t *testing.T) {
	attrs := dataSourceSchema(t, NewDataSourceF5XCSiteState()).Attributes

	tests := []struct {
		key      string
		required bool
		optional bool
		computed bool
		check    func(t *testing.T, a schema.Attribute)
	}{
		{key: "name", required: true},
		{key: "namespace", optional: true, computed: true},
		{key: "id", computed: true},
		{key: "site_state", computed: true},
		{key: "is_online", computed: true, check: func(t *testing.T, a schema.Attribute) {
			assert.IsType(t, schema.BoolAttribute{}, a)
		}},
		{key: "site_type", computed: true},
		{key: "software_version", computed: true},
		{key: "operating_system_version", computed: true},
		{key: "nodes", computed: true, check: func(t *testing.T, a schema.Attribute) {
			nodes, ok := a.(schema.ListNestedAttribute)
			require.True(t, ok, "nodes must be a ListNestedAttribute, got %T", a)
			nested := nodes.NestedObject.Attributes
			assert.Len(t, nested, 3)
			for _, k := range []string{"name", "slo_address", "sli_address"} {
				require.Contains(t, nested, k)
				assert.True(t, nested[k].IsComputed(), "nodes.%s must be computed", k)
			}
		}},
	}

	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			a, ok := attrs[tc.key]
			require.True(t, ok, "missing attribute %q", tc.key)
			assert.Equal(t, tc.required, a.IsRequired(), "Required")
			assert.Equal(t, tc.optional, a.IsOptional(), "Optional")
			assert.Equal(t, tc.computed, a.IsComputed(), "Computed")
			if tc.check != nil {
				tc.check(t, a)
			}
		})
	}

	t.Run("timeouts", func(t *testing.T) {
		assert.Contains(t, attrs, "timeouts")
	})
	t.Run("no extra attributes", func(t *testing.T) {
		assert.Len(t, attrs, len(tests)+1)
	})
}

// TestSiteState_Configure verifies that Configure accepts nil and
// *client.APIClient, and rejects any other type with an error diagnostic.
func TestSiteState_Configure(t *testing.T) {
	for _, tc := range configureCases {
		t.Run(tc.name, func(t *testing.T) {
			var resp datasource.ConfigureResponse
			NewDataSourceF5XCSiteState().(datasource.DataSourceWithConfigure).Configure(
				context.Background(), datasource.ConfigureRequest{ProviderData: tc.providerData}, &resp)

			assert.Equal(t, tc.wantErr, resp.Diagnostics.HasError(), "diagnostics: %v", resp.Diagnostics)
			if tc.wantErr {
				assert.Equal(t, "Unexpected Data Source Configure Type", resp.Diagnostics.Errors()[0].Summary())
			}
		})
	}
}

// siteStateModel reads the state written by Read into the model.
func siteStateModel(t *testing.T, resp datasource.ReadResponse) DataSourceF5XCSiteStateModel {
	t.Helper()
	var got DataSourceF5XCSiteStateModel
	require.False(t, resp.State.Get(context.Background(), &got).HasError())
	return got
}

// TestSiteState_Read_OK verifies that a 200 response is mapped to state:
// is_online follows site_state, nodes keep their order, nil nodes are dropped,
// and namespace defaults to "system".
func TestSiteState_Read_OK(t *testing.T) {
	tests := []struct {
		name       string
		config     map[string]tftypes.Value
		body       string
		wantPath   string
		wantID     string
		wantState  string
		wantOnline bool
		wantNodes  []SiteNodeModel
	}{
		{
			name:   "online site with nodes",
			config: map[string]tftypes.Value{"name": strVal("ce1"), "namespace": strVal("system")},
			body: `{"metadata":{"name":"ce1","namespace":"system"},"spec":{
				"site_state":"ONLINE","site_type":"CUSTOMER_EDGE",
				"volterra_software_version":"crt-20260101","operating_system_version":"9.2026",
				"main_nodes":[
					{"name":"node-0","slo_address":"10.0.0.1","sli_address":"192.168.0.1"},
					{"name":"node-1","slo_address":"10.0.0.2"}
				]}}`,
			wantPath:   "/api/config/namespaces/system/sites/ce1",
			wantID:     "system/ce1",
			wantState:  "ONLINE",
			wantOnline: true,
			wantNodes: []SiteNodeModel{
				{Name: strType("node-0"), SloAddress: strType("10.0.0.1"), SliAddress: strType("192.168.0.1")},
				{Name: strType("node-1"), SloAddress: strType("10.0.0.2"), SliAddress: strType("")},
			},
		},
		{
			name:       "namespace defaults to system and offline site",
			config:     map[string]tftypes.Value{"name": strVal("ce2")},
			body:       `{"metadata":{"name":"ce2","namespace":"system"},"spec":{"site_state":"PROVISIONING"}}`,
			wantPath:   "/api/config/namespaces/system/sites/ce2",
			wantID:     "system/ce2",
			wantState:  "PROVISIONING",
			wantOnline: false,
			wantNodes:  []SiteNodeModel{},
		},
		{
			name:       "custom namespace",
			config:     map[string]tftypes.Value{"name": strVal("ce3"), "namespace": strVal("shared")},
			body:       `{"metadata":{"name":"ce3","namespace":"shared"},"spec":{"site_state":"ONLINE"}}`,
			wantPath:   "/api/config/namespaces/shared/sites/ce3",
			wantID:     "shared/ce3",
			wantState:  "ONLINE",
			wantOnline: true,
			wantNodes:  []SiteNodeModel{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, fake := newFakeAPIClient(t, http.StatusOK, tc.body)

			resp := readDataSource(t, NewDataSourceF5XCSiteState().(datasource.DataSourceWithConfigure), c, tc.config)

			require.False(t, resp.Diagnostics.HasError(), "diagnostics: %v", resp.Diagnostics)
			assert.Equal(t, []string{tc.wantPath}, fake.requestPaths())

			got := siteStateModel(t, resp)
			assert.Equal(t, tc.wantID, got.Id.ValueString())
			assert.Equal(t, tc.wantState, got.SiteState.ValueString())
			assert.Equal(t, tc.wantOnline, got.IsOnline.ValueBool())
			assert.Equal(t, tc.wantNodes, got.Nodes)
		})
	}
}

// TestSiteState_Read_Fields verifies that site_type and the version fields are
// copied from the spec.
func TestSiteState_Read_Fields(t *testing.T) {
	c, _ := newFakeAPIClient(t, http.StatusOK, `{"metadata":{"name":"ce1","namespace":"system"},"spec":{
		"site_state":"ONLINE","site_type":"CUSTOMER_EDGE",
		"volterra_software_version":"crt-20260101","operating_system_version":"9.2026"}}`)

	resp := readDataSource(t, NewDataSourceF5XCSiteState().(datasource.DataSourceWithConfigure), c,
		map[string]tftypes.Value{"name": strVal("ce1")})

	require.False(t, resp.Diagnostics.HasError(), "diagnostics: %v", resp.Diagnostics)
	got := siteStateModel(t, resp)
	assert.Equal(t, "ce1", got.Name.ValueString())
	assert.Equal(t, "system", got.Namespace.ValueString())
	assert.Equal(t, "CUSTOMER_EDGE", got.SiteType.ValueString())
	assert.Equal(t, "crt-20260101", got.SoftwareVersion.ValueString())
	assert.Equal(t, "9.2026", got.OperatingSystemVersion.ValueString())
}

// TestSiteState_Read_Errors verifies that API errors become a "Client Error"
// diagnostic, a response without a spec becomes a "Parsing Error", and no state
// is written in either case.
func TestSiteState_Read_Errors(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		body        string
		wantSummary string
	}{
		{name: "404 not found", status: http.StatusNotFound, body: `{"code":5,"message":"not found"}`, wantSummary: "Client Error"},
		{name: "403 forbidden", status: http.StatusForbidden, body: `{"code":7,"message":"denied"}`, wantSummary: "Client Error"},
		{name: "500 server error", status: http.StatusInternalServerError, body: `{"code":13,"message":"boom"}`, wantSummary: "Client Error"},
		{name: "invalid json", status: http.StatusOK, body: `{not json`, wantSummary: "Client Error"},
		{name: "missing spec", status: http.StatusOK, body: `{"metadata":{"name":"ce1","namespace":"system"}}`, wantSummary: "Parsing Error"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := newFakeAPIClient(t, tc.status, tc.body)

			resp := readDataSource(t, NewDataSourceF5XCSiteState().(datasource.DataSourceWithConfigure), c,
				map[string]tftypes.Value{"name": strVal("ce1")})

			require.True(t, resp.Diagnostics.HasError())
			assert.Equal(t, tc.wantSummary, resp.Diagnostics.Errors()[0].Summary())
			assert.True(t, resp.State.Raw.IsNull(), "state must not be set on error")
		})
	}
}
