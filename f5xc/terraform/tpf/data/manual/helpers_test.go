//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package manual

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"
	"gopkg.volterra.us/stdlib/client/vesapi"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/client"
	ves_io_schema_combined "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/combined"
)

// fakeObjectServer is a local TLS API gateway that returns one canned reply
// for every request and records the request paths.
type fakeObjectServer struct {
	mu     sync.Mutex
	status int
	body   string
	paths  []string
}

func (f *fakeObjectServer) reply() (int, string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.status, f.body
}

func (f *fakeObjectServer) recordPath(path string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.paths = append(f.paths, path)
}

func (f *fakeObjectServer) requestPaths() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.paths...)
}

func (f *fakeObjectServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.recordPath(r.URL.Path)
	status, body := f.reply()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

// newFakeAPIClient starts a TLS server that replies with status and body, and
// returns a *client.APIClient whose vesapi client targets it.
//
// The provider's own URI remapper is unexported, so the test round tripper
// does the same /public -> /config rewrite. The vesapi client trusts only the
// test server certificate.
func newFakeAPIClient(t *testing.T, status int, body string) (*client.APIClient, *fakeObjectServer) {
	t.Helper()
	fake := &fakeObjectServer{status: status, body: body}
	srv := httptest.NewTLSServer(fake)
	t.Cleanup(srv.Close)

	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	rewrite := func(req *http.Request, next http.RoundTripper) (*http.Response, error) {
		req.URL.Path = strings.Replace(req.URL.Path, "/api/public/", "/api/config/", 1)
		return next.RoundTrip(req)
	}

	vc, err := vesapi.NewAPIClient(srv.URL+"/api", ves_io_schema_combined.MDR,
		vesapi.WithTLSCfg(&tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}),
		vesapi.WithCfgRoundTripperFn(rewrite),
	)
	require.NoError(t, err)
	return &client.APIClient{APIClient: vc}, fake
}

// dataSourceSchema returns the schema of ds and fails on error diagnostics.
func dataSourceSchema(t *testing.T, ds datasource.DataSource) schema.Schema {
	t.Helper()
	var resp datasource.SchemaResponse
	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	require.False(t, resp.Diagnostics.HasError(), "Schema() diagnostics: %v", resp.Diagnostics)
	return resp.Schema
}

// configFor builds a tfsdk.Config for s where every attribute is null except
// the values given in set.
func configFor(t *testing.T, s schema.Schema, set map[string]tftypes.Value) tfsdk.Config {
	t.Helper()
	objType, ok := s.Type().TerraformType(context.Background()).(tftypes.Object)
	require.True(t, ok)

	vals := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, typ := range objType.AttributeTypes {
		if v, found := set[name]; found {
			vals[name] = v
			continue
		}
		vals[name] = tftypes.NewValue(typ, nil)
	}
	return tfsdk.Config{Schema: s, Raw: tftypes.NewValue(objType, vals)}
}

// readDataSource configures ds with c, runs Read with the given config values,
// and returns the response.
func readDataSource(t *testing.T, ds datasource.DataSourceWithConfigure, c *client.APIClient, set map[string]tftypes.Value) datasource.ReadResponse {
	t.Helper()
	ctx := context.Background()
	s := dataSourceSchema(t, ds)

	var cfgResp datasource.ConfigureResponse
	ds.Configure(ctx, datasource.ConfigureRequest{ProviderData: c}, &cfgResp)
	require.False(t, cfgResp.Diagnostics.HasError(), "Configure() diagnostics: %v", cfgResp.Diagnostics)

	resp := datasource.ReadResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}}
	ds.Read(ctx, datasource.ReadRequest{Config: configFor(t, s, set)}, &resp)
	return resp
}

// configureCases are the Configure inputs shared by both data sources.
var configureCases = []struct {
	name         string
	providerData any
	wantErr      bool
}{
	{name: "nil provider data", providerData: nil},
	{name: "api client pointer", providerData: &client.APIClient{}},
	{name: "wrong type", providerData: "not-a-client", wantErr: true},
	{name: "client value instead of pointer", providerData: client.APIClient{}, wantErr: true},
}

func strVal(s string) tftypes.Value { return tftypes.NewValue(tftypes.String, s) }

// strType returns a known types.String with value v.
func strType(v string) types.String { return types.StringValue(v) }
