//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package manual

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.volterra.us/stdlib/client/vesapi"

	ves_io_schema "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema"
	ves_io_schema_combined "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/combined"
	ves_io_schema_registration "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/registration"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/client"
)

// fakeRegistrationServer is a local TLS API gateway for the registration
// custom APIs. Each endpoint returns a configurable status and body.
type fakeRegistrationServer struct {
	mu sync.Mutex

	pendingBody   string
	approvedBody  string
	listStatus    int
	getStatus     int
	approveStatus int

	listCalls    int
	getCalls     int
	approveCalls []string
}

func (f *fakeRegistrationServer) handle(r *http.Request) (int, string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	body, _ := io.ReadAll(r.Body)
	switch {
	case strings.HasSuffix(r.URL.Path, "/listregistrationsbystate"):
		f.listCalls++
		if f.listStatus != 0 {
			return f.listStatus, `{}`
		}
		if strings.Contains(string(body), "APPROVED") {
			return http.StatusOK, orEmptyList(f.approvedBody)
		}
		return http.StatusOK, orEmptyList(f.pendingBody)
	case strings.HasSuffix(r.URL.Path, "/approve"):
		f.approveCalls = append(f.approveCalls, string(body))
		return statusOrOK(f.approveStatus), `{}`
	case strings.Contains(r.URL.Path, "/registrations/"):
		f.getCalls++
		return statusOrOK(f.getStatus), `{}`
	}
	return http.StatusNotImplemented, `{}`
}

func (f *fakeRegistrationServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	status, body := f.handle(r)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func (f *fakeRegistrationServer) approvals() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.approveCalls...)
}

func (f *fakeRegistrationServer) counts() (list, get int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.listCalls, f.getCalls
}

func orEmptyList(body string) string {
	if body == "" {
		return `{"items":[]}`
	}
	return body
}

func statusOrOK(status int) int {
	if status == 0 {
		return http.StatusOK
	}
	return status
}

// newRegistrationAPIClient starts fake and returns a *client.APIClient whose
// vesapi client trusts only the test server certificate.
func newRegistrationAPIClient(t *testing.T, fake *fakeRegistrationServer) *client.APIClient {
	t.Helper()
	return newRegistrationAPIClientWithHook(t, fake, nil)
}

// newRegistrationAPIClientWithHook is newRegistrationAPIClient with afterList,
// which runs after each list response is fully read. The hook runs once the
// HTTP call is complete, so it can cancel the caller's context without
// failing that call.
func newRegistrationAPIClientWithHook(t *testing.T, fake *fakeRegistrationServer, afterList func(call int)) *client.APIClient {
	t.Helper()
	srv := httptest.NewTLSServer(fake)
	t.Cleanup(srv.Close)

	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	passthrough := func(req *http.Request, next http.RoundTripper) (*http.Response, error) {
		resp, err := next.RoundTrip(req)
		if err != nil || afterList == nil || !strings.HasSuffix(req.URL.Path, "/listregistrationsbystate") {
			return resp, err
		}
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		resp.Body = io.NopCloser(bytes.NewReader(body))
		list, _ := fake.counts()
		afterList(list)
		return resp, nil
	}
	vc, err := vesapi.NewAPIClient(srv.URL+"/api", ves_io_schema_combined.MDR,
		vesapi.WithTLSCfg(&tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}),
		vesapi.WithCfgRoundTripperFn(passthrough),
	)
	require.NoError(t, err)
	return &client.APIClient{APIClient: vc}
}

// listBody returns a ListResponse with one item that uses get_spec.
func listBody(name, cluster, hostname string) string {
	return `{"items":[{"name":"` + name + `","get_spec":{"passport":{"cluster_name":"` + cluster +
		`","cluster_size":1},"infra":{"hostname":"` + hostname + `"}}}]}`
}

func registrationSchema(t *testing.T) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	NewResourceF5XCRegistrationApproval().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	require.False(t, resp.Diagnostics.HasError())
	return resp.Schema
}

// baseRegistrationModel returns a valid model with all optional values null.
func baseRegistrationModel() registrationApprovalModel {
	return registrationApprovalModel{
		Namespace:          types.StringValue("system"),
		ClusterName:        types.StringValue("ce1"),
		ClusterSize:        types.Int64Value(3),
		Hostname:           types.StringValue("node-0"),
		Retry:              types.Int64Value(1),
		WaitTime:           types.Int64Value(1),
		ConnectedRegion:    types.StringNull(),
		Latitude:           types.Float64Null(),
		Longitude:          types.Float64Null(),
		TunnelType:         types.ListNull(types.StringType),
		PrivateNetworkName: types.StringNull(),
		RegistrationName:   types.StringNull(),
		Timeouts:           testTimeoutsValue(),
	}
}

// badTimeoutsValue returns a timeouts value that fails duration parsing for
// every operation.
func badTimeoutsValue() timeouts.Value {
	attrTypes := map[string]attr.Type{
		"create": types.StringType,
		"read":   types.StringType,
		"update": types.StringType,
		"delete": types.StringType,
	}
	return timeouts.Value{Object: types.ObjectValueMust(attrTypes, map[string]attr.Value{
		"create": types.StringValue("bad"),
		"read":   types.StringValue("bad"),
		"update": types.StringValue("bad"),
		"delete": types.StringValue("bad"),
	})}
}

func registrationState(t *testing.T, m registrationApprovalModel) tfsdk.State {
	t.Helper()
	s := tfsdk.State{Schema: registrationSchema(t)}
	require.False(t, s.Set(context.Background(), &m).HasError())
	return s
}

// invalidRaw is a value whose type does not match any resource schema, so
// Get on a Plan, State, or Config that holds it returns an error diagnostic.
var invalidRaw = tftypes.NewValue(tftypes.String, "not-an-object")

func firstErrorSummary(d diag.Diagnostics) string {
	if !d.HasError() {
		return ""
	}
	return d.Errors()[0].Summary()
}

// TestRegistrationApproval_Metadata verifies the type name suffix.
func TestRegistrationApproval_Metadata(t *testing.T) {
	var resp resource.MetadataResponse
	NewResourceF5XCRegistrationApproval().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "f5xc"}, &resp)
	assert.Equal(t, "f5xc_registration_approval", resp.TypeName)
}

// TestRegistrationApproval_Schema verifies required inputs and the computed
// registration_name.
func TestRegistrationApproval_Schema(t *testing.T) {
	attrs := registrationSchema(t).Attributes
	for _, k := range []string{"cluster_name", "cluster_size", "hostname"} {
		assert.True(t, attrs[k].IsRequired(), k)
	}
	assert.True(t, attrs["registration_name"].IsComputed())
	assert.Contains(t, attrs, "timeouts")
}

// TestRegistrationApproval_Configure verifies that Configure accepts nil and
// *client.APIClient, and rejects other types.
func TestRegistrationApproval_Configure(t *testing.T) {
	tests := []struct {
		name    string
		data    any
		wantErr bool
	}{
		{name: "nil", data: nil},
		{name: "api client", data: &client.APIClient{}},
		{name: "wrong type", data: "x", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := &ResourceF5XCRegistrationApproval{}
			var resp resource.ConfigureResponse
			r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: tc.data}, &resp)
			assert.Equal(t, tc.wantErr, resp.Diagnostics.HasError())
			if !tc.wantErr && tc.data != nil {
				assert.Same(t, tc.data, r.client)
			}
		})
	}
}

// TestRegistrationApproval_ValidateConfig verifies that a valid config has no
// diagnostics and an undecodable config reports an error.
func TestRegistrationApproval_ValidateConfig(t *testing.T) {
	s := registrationSchema(t)
	r := &ResourceF5XCRegistrationApproval{}

	valid := registrationState(t, baseRegistrationModel())
	var ok resource.ValidateConfigResponse
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: s, Raw: valid.Raw}}, &ok)
	assert.False(t, ok.Diagnostics.HasError())

	var bad resource.ValidateConfigResponse
	r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: s, Raw: invalidRaw}}, &bad)
	assert.True(t, bad.Diagnostics.HasError())
}

// TestRegistrationApproval_ModifyPlan verifies the destroy warning, plan
// decoding on create, and no change on update.
func TestRegistrationApproval_ModifyPlan(t *testing.T) {
	ctx := context.Background()
	s := registrationSchema(t)
	r := &ResourceF5XCRegistrationApproval{}
	nullObj := tftypes.NewValue(s.Type().TerraformType(ctx), nil)
	valid := registrationState(t, baseRegistrationModel()).Raw

	t.Run("destroy warns", func(t *testing.T) {
		var resp resource.ModifyPlanResponse
		r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: tfsdk.Plan{Schema: s, Raw: nullObj}}, &resp)
		require.Len(t, resp.Diagnostics.Warnings(), 1)
		assert.Equal(t, "Resource Destruction", resp.Diagnostics.Warnings()[0].Summary())
	})
	t.Run("create with valid plan", func(t *testing.T) {
		var resp resource.ModifyPlanResponse
		r.ModifyPlan(ctx, resource.ModifyPlanRequest{
			Plan:  tfsdk.Plan{Schema: s, Raw: valid},
			State: tfsdk.State{Schema: s, Raw: nullObj},
		}, &resp)
		assert.Empty(t, resp.Diagnostics)
	})
	t.Run("create with undecodable plan", func(t *testing.T) {
		var resp resource.ModifyPlanResponse
		r.ModifyPlan(ctx, resource.ModifyPlanRequest{
			Plan:  tfsdk.Plan{Schema: s, Raw: invalidRaw},
			State: tfsdk.State{Schema: s, Raw: nullObj},
		}, &resp)
		assert.True(t, resp.Diagnostics.HasError())
	})
	t.Run("update", func(t *testing.T) {
		var resp resource.ModifyPlanResponse
		r.ModifyPlan(ctx, resource.ModifyPlanRequest{
			Plan:  tfsdk.Plan{Schema: s, Raw: valid},
			State: tfsdk.State{Schema: s, Raw: valid},
		}, &resp)
		assert.Empty(t, resp.Diagnostics)
	})
}

// applyFn runs Create or Update with plan and returns the diagnostics and the
// resulting state.
type applyFn func(t *testing.T, r *ResourceF5XCRegistrationApproval, plan tfsdk.Plan) (diag.Diagnostics, tfsdk.State)

func runCreate(t *testing.T, r *ResourceF5XCRegistrationApproval, plan tfsdk.Plan) (diag.Diagnostics, tfsdk.State) {
	t.Helper()
	resp := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &resp)
	return resp.Diagnostics, resp.State
}

func runUpdate(t *testing.T, r *ResourceF5XCRegistrationApproval, plan tfsdk.Plan) (diag.Diagnostics, tfsdk.State) {
	t.Helper()
	resp := resource.UpdateResponse{State: tfsdk.State{Schema: plan.Schema}}
	r.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: tfsdk.State(plan)}, &resp)
	return resp.Diagnostics, resp.State
}

// TestRegistrationApproval_CreateUpdate verifies that Create and Update
// resolve the registration, approve it only when it is not yet APPROVED, and
// report each failure with its own diagnostic summary.
func TestRegistrationApproval_CreateUpdate(t *testing.T) {
	ops := map[string]applyFn{"create": runCreate, "update": runUpdate}

	for opName, op := range ops {
		t.Run(opName, func(t *testing.T) {
			t.Run("pending registration is approved", func(t *testing.T) {
				fake := &fakeRegistrationServer{pendingBody: listBody("reg-1", "ce1", "node-0.example.com")}
				r := &ResourceF5XCRegistrationApproval{client: newRegistrationAPIClient(t, fake)}
				m := baseRegistrationModel()
				m.Namespace = types.StringValue("  ")

				diags, state := op(t, r, tfsdk.Plan(registrationState(t, m)))
				require.False(t, diags.HasError(), "%v", diags)

				var got registrationApprovalModel
				require.False(t, state.Get(context.Background(), &got).HasError())
				assert.Equal(t, "reg-1", got.RegistrationName.ValueString())
				assert.Equal(t, "system", got.Namespace.ValueString())
				require.Len(t, fake.approvals(), 1)
				assert.Contains(t, fake.approvals()[0], "APPROVED")
			})

			t.Run("approved registration is not approved again", func(t *testing.T) {
				fake := &fakeRegistrationServer{approvedBody: listBody("reg-2", "ce1", "node-0")}
				r := &ResourceF5XCRegistrationApproval{client: newRegistrationAPIClient(t, fake)}

				diags, _ := op(t, r, tfsdk.Plan(registrationState(t, baseRegistrationModel())))
				require.False(t, diags.HasError(), "%v", diags)
				assert.Empty(t, fake.approvals())
			})

			t.Run("undecodable plan", func(t *testing.T) {
				r := &ResourceF5XCRegistrationApproval{}
				diags, _ := op(t, r, tfsdk.Plan{Schema: registrationSchema(t), Raw: invalidRaw})
				assert.True(t, diags.HasError())
			})

			t.Run("invalid timeout", func(t *testing.T) {
				m := baseRegistrationModel()
				m.Timeouts = badTimeoutsValue()
				diags, _ := op(t, &ResourceF5XCRegistrationApproval{}, tfsdk.Plan(registrationState(t, m)))
				assert.Equal(t, "Timeout Cannot Be Parsed", firstErrorSummary(diags))
			})

			t.Run("resolve error", func(t *testing.T) {
				diags, _ := op(t, &ResourceF5XCRegistrationApproval{}, tfsdk.Plan(registrationState(t, baseRegistrationModel())))
				assert.Equal(t, "Resolve Registration Error", firstErrorSummary(diags))
			})

			t.Run("invalid tunnel type", func(t *testing.T) {
				fake := &fakeRegistrationServer{pendingBody: listBody("reg-1", "ce1", "node-0")}
				r := &ResourceF5XCRegistrationApproval{client: newRegistrationAPIClient(t, fake)}
				m := baseRegistrationModel()
				m.TunnelType = types.ListValueMust(types.StringType, []attr.Value{types.StringValue("BAD")})

				diags, _ := op(t, r, tfsdk.Plan(registrationState(t, m)))
				assert.Equal(t, "Build Approval Request Error", firstErrorSummary(diags))
			})

			t.Run("approve call fails", func(t *testing.T) {
				fake := &fakeRegistrationServer{
					pendingBody:   listBody("reg-1", "ce1", "node-0"),
					approveStatus: http.StatusInternalServerError,
				}
				r := &ResourceF5XCRegistrationApproval{client: newRegistrationAPIClient(t, fake)}

				diags, _ := op(t, r, tfsdk.Plan(registrationState(t, baseRegistrationModel())))
				assert.Equal(t, "Registration Approve Error", firstErrorSummary(diags))
			})
		})
	}
}

func runRead(t *testing.T, r *ResourceF5XCRegistrationApproval, state tfsdk.State) resource.ReadResponse {
	t.Helper()
	resp := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	return resp
}

// TestRegistrationApproval_Read verifies that Read keeps state when the
// registration exists, removes it when the name is empty or the API returns
// 404, and reports other errors.
func TestRegistrationApproval_Read(t *testing.T) {
	named := baseRegistrationModel()
	named.RegistrationName = types.StringValue("reg-1")
	named.Namespace = types.StringValue("")

	t.Run("exists", func(t *testing.T) {
		fake := &fakeRegistrationServer{}
		r := &ResourceF5XCRegistrationApproval{client: newRegistrationAPIClient(t, fake)}
		resp := runRead(t, r, registrationState(t, named))
		require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

		var got registrationApprovalModel
		require.False(t, resp.State.Get(context.Background(), &got).HasError())
		assert.Equal(t, "system", got.Namespace.ValueString())
		_, gets := fake.counts()
		assert.Equal(t, 1, gets)
	})
	t.Run("empty registration name removes state", func(t *testing.T) {
		resp := runRead(t, &ResourceF5XCRegistrationApproval{}, registrationState(t, baseRegistrationModel()))
		assert.False(t, resp.Diagnostics.HasError())
		assert.True(t, resp.State.Raw.IsNull())
	})
	t.Run("404 removes state", func(t *testing.T) {
		fake := &fakeRegistrationServer{getStatus: http.StatusNotFound}
		r := &ResourceF5XCRegistrationApproval{client: newRegistrationAPIClient(t, fake)}
		resp := runRead(t, r, registrationState(t, named))
		assert.False(t, resp.Diagnostics.HasError())
		assert.True(t, resp.State.Raw.IsNull())
	})
	t.Run("server error", func(t *testing.T) {
		fake := &fakeRegistrationServer{getStatus: http.StatusInternalServerError}
		r := &ResourceF5XCRegistrationApproval{client: newRegistrationAPIClient(t, fake)}
		resp := runRead(t, r, registrationState(t, named))
		assert.Equal(t, "Client Error", firstErrorSummary(resp.Diagnostics))
	})
	t.Run("undecodable state", func(t *testing.T) {
		resp := runRead(t, &ResourceF5XCRegistrationApproval{}, tfsdk.State{Schema: registrationSchema(t), Raw: invalidRaw})
		assert.True(t, resp.Diagnostics.HasError())
	})
	t.Run("invalid timeout", func(t *testing.T) {
		m := named
		m.Timeouts = badTimeoutsValue()
		resp := runRead(t, &ResourceF5XCRegistrationApproval{}, registrationState(t, m))
		assert.Equal(t, "Timeout Cannot Be Parsed", firstErrorSummary(resp.Diagnostics))
	})
}

func runDelete(t *testing.T, r *ResourceF5XCRegistrationApproval, state tfsdk.State) diag.Diagnostics {
	t.Helper()
	var resp resource.DeleteResponse
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
	return resp.Diagnostics
}

// TestRegistrationApproval_Delete verifies that Delete retires the
// registration, ignores 404, skips the call when the name is empty, and
// reports other errors.
func TestRegistrationApproval_Delete(t *testing.T) {
	named := baseRegistrationModel()
	named.RegistrationName = types.StringValue("reg-1")

	t.Run("retires registration", func(t *testing.T) {
		fake := &fakeRegistrationServer{}
		r := &ResourceF5XCRegistrationApproval{client: newRegistrationAPIClient(t, fake)}
		require.False(t, runDelete(t, r, registrationState(t, named)).HasError())
		require.Len(t, fake.approvals(), 1)
		assert.Contains(t, fake.approvals()[0], "RETIRED")
	})
	t.Run("404 is ignored", func(t *testing.T) {
		fake := &fakeRegistrationServer{approveStatus: http.StatusNotFound}
		r := &ResourceF5XCRegistrationApproval{client: newRegistrationAPIClient(t, fake)}
		assert.False(t, runDelete(t, r, registrationState(t, named)).HasError())
	})
	t.Run("server error", func(t *testing.T) {
		fake := &fakeRegistrationServer{approveStatus: http.StatusInternalServerError}
		r := &ResourceF5XCRegistrationApproval{client: newRegistrationAPIClient(t, fake)}
		assert.Equal(t, "Registration Retire Error", firstErrorSummary(runDelete(t, r, registrationState(t, named))))
	})
	t.Run("empty registration name", func(t *testing.T) {
		assert.False(t, runDelete(t, &ResourceF5XCRegistrationApproval{}, registrationState(t, baseRegistrationModel())).HasError())
	})
	t.Run("undecodable state", func(t *testing.T) {
		assert.True(t, runDelete(t, &ResourceF5XCRegistrationApproval{}, tfsdk.State{Schema: registrationSchema(t), Raw: invalidRaw}).HasError())
	})
	t.Run("invalid timeout", func(t *testing.T) {
		m := named
		m.Timeouts = badTimeoutsValue()
		assert.Equal(t, "Timeout Cannot Be Parsed", firstErrorSummary(runDelete(t, &ResourceF5XCRegistrationApproval{}, registrationState(t, m))))
	})
}

// TestRegistrationApproval_ImportState verifies that the ID must be
// namespace/registration_name and that both parts are written to state.
func TestRegistrationApproval_ImportState(t *testing.T) {
	ctx := context.Background()
	s := registrationSchema(t)
	newResp := func() *resource.ImportStateResponse {
		return &resource.ImportStateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}}
	}

	for _, id := range []string{"only-one", "a/b/c", " /reg", "system/ "} {
		resp := newResp()
		(&ResourceF5XCRegistrationApproval{}).ImportState(ctx, resource.ImportStateRequest{ID: id}, resp)
		assert.Equal(t, "Invalid Import ID", firstErrorSummary(resp.Diagnostics), id)
	}

	resp := newResp()
	(&ResourceF5XCRegistrationApproval{}).ImportState(ctx, resource.ImportStateRequest{ID: "system/reg-1"}, resp)
	require.False(t, resp.Diagnostics.HasError())
	var name types.String
	require.False(t, resp.State.GetAttribute(ctx, path.Root("registration_name"), &name).HasError())
	assert.Equal(t, "reg-1", name.ValueString())
}

// TestRegistrationApproval_Normalizers verifies defaults and bounds for
// namespace, retry, wait interval, and hostname.
func TestRegistrationApproval_Normalizers(t *testing.T) {
	assert.Equal(t, "system", normalizeNamespace(" "))
	assert.Equal(t, "ns", normalizeNamespace(" ns "))

	assert.Equal(t, int64(1), normalizeRetryAttempts(0))
	assert.Equal(t, int64(5), normalizeRetryAttempts(5))

	assert.Equal(t, 60*time.Second, normalizeWaitInterval(29))
	assert.Equal(t, 60*time.Second, normalizeWaitInterval(121))
	assert.Equal(t, 45*time.Second, normalizeWaitInterval(45))

	assert.Equal(t, "node-0", normalizeHostname(" Node-0.Example.com "))
	assert.Equal(t, ".hidden", normalizeHostname(".hidden"))
	assert.Equal(t, "plain", normalizeHostname("plain"))
}

// TestResolveRegistrationForApproval verifies input checks, the search order
// PENDING then APPROVED, the name fallback to metadata, and the not-found and
// context-cancel errors.
func TestResolveRegistrationForApproval(t *testing.T) {
	ctx := context.Background()

	t.Run("input checks", func(t *testing.T) {
		_, _, _, err := resolveRegistrationForApproval(ctx, nil, "", "c", "h", 1, 1)
		assert.ErrorContains(t, err, "api client is required")

		c := &client.APIClient{}
		_, _, _, err = resolveRegistrationForApproval(ctx, c, "", " ", "h", 1, 1)
		assert.ErrorContains(t, err, "cluster_name is required")
		_, _, _, err = resolveRegistrationForApproval(ctx, c, "", "c", " ", 1, 1)
		assert.ErrorContains(t, err, "hostname is required")
	})

	t.Run("pending match", func(t *testing.T) {
		fake := &fakeRegistrationServer{pendingBody: listBody("reg-1", "ce1", "node-0")}
		name, passport, state, err := resolveRegistrationForApproval(ctx, newRegistrationAPIClient(t, fake), "", "ce1", "node-0", 1, 1)
		require.NoError(t, err)
		assert.Equal(t, "reg-1", name)
		assert.Equal(t, "ce1", passport.GetClusterName())
		assert.Equal(t, ves_io_schema_registration.PENDING, state)
	})

	t.Run("approved match", func(t *testing.T) {
		fake := &fakeRegistrationServer{approvedBody: listBody("reg-2", "ce1", "node-0")}
		_, _, state, err := resolveRegistrationForApproval(ctx, newRegistrationAPIClient(t, fake), "", "ce1", "node-0", 1, 1)
		require.NoError(t, err)
		assert.Equal(t, ves_io_schema_registration.APPROVED, state)
	})

	t.Run("name from metadata", func(t *testing.T) {
		body := `{"items":[{"metadata":{"name":"meta-reg"},"get_spec":{"passport":{"cluster_name":"ce1"},"infra":{"hostname":"node-0"}}}]}`
		fake := &fakeRegistrationServer{pendingBody: body}
		name, _, _, err := resolveRegistrationForApproval(ctx, newRegistrationAPIClient(t, fake), "", "ce1", "node-0", 1, 1)
		require.NoError(t, err)
		assert.Equal(t, "meta-reg", name)
	})

	t.Run("empty name", func(t *testing.T) {
		fake := &fakeRegistrationServer{pendingBody: listBody("", "ce1", "node-0")}
		_, _, _, err := resolveRegistrationForApproval(ctx, newRegistrationAPIClient(t, fake), "", "ce1", "node-0", 1, 1)
		assert.ErrorContains(t, err, "matched registration has empty name")
	})

	t.Run("list error", func(t *testing.T) {
		fake := &fakeRegistrationServer{listStatus: http.StatusInternalServerError}
		_, _, _, err := resolveRegistrationForApproval(ctx, newRegistrationAPIClient(t, fake), "", "ce1", "node-0", 1, 1)
		assert.ErrorContains(t, err, "list by state call failed")
	})

	t.Run("not found after attempts", func(t *testing.T) {
		fake := &fakeRegistrationServer{}
		_, _, _, err := resolveRegistrationForApproval(ctx, newRegistrationAPIClient(t, fake), "", "ce1", "node-0", 0, 1)
		assert.ErrorContains(t, err, "registration not found")
		lists, _ := fake.counts()
		assert.Equal(t, 2, lists, "one attempt searches both states")
	})

	t.Run("context done while waiting", func(t *testing.T) {
		cctx, cancel := context.WithCancel(ctx)
		defer cancel()
		// Cancel after attempt 0 has searched both states, so the context is
		// done only when the poll wait starts.
		fake := &fakeRegistrationServer{}
		c := newRegistrationAPIClientWithHook(t, fake, func(call int) {
			if call == 2 {
				cancel()
			}
		})
		_, _, _, err := resolveRegistrationForApproval(cctx, c, "", "ce1", "node-0", 2, 30)
		assert.ErrorContains(t, err, "context done while waiting")
	})
}

// TestFindRegistrationByClusterAndHostname verifies cluster and hostname
// matching: an exact hostname wins, a single cluster match is accepted, and
// several cluster matches without a hostname match are an error.
func TestFindRegistrationByClusterAndHostname(t *testing.T) {
	ctx := context.Background()
	find := func(t *testing.T, body string) (*ves_io_schema_registration.ListResponseItem, error) {
		fake := &fakeRegistrationServer{pendingBody: body}
		return findRegistrationByClusterAndHostname(ctx, newRegistrationAPIClient(t, fake), "system", ves_io_schema_registration.PENDING, "CE1", "node-1.example.com")
	}

	t.Run("hostname match among cluster matches", func(t *testing.T) {
		item, err := find(t, `{"items":[
			{"name":"other","get_spec":{"passport":{"cluster_name":"x"},"infra":{"hostname":"node-1"}}},
			{"name":"a","get_spec":{"passport":{"cluster_name":"ce1"},"infra":{"hostname":"node-0"}}},
			{"name":"b","get_spec":{"passport":{"cluster_name":"ce1"},"infra":{"hostname":"NODE-1"}}}]}`)
		require.NoError(t, err)
		assert.Equal(t, "b", item.GetName())
	})
	t.Run("single cluster match", func(t *testing.T) {
		item, err := find(t, listBody("a", "ce1", "node-9"))
		require.NoError(t, err)
		assert.Equal(t, "a", item.GetName())
	})
	t.Run("ambiguous cluster matches", func(t *testing.T) {
		_, err := find(t, `{"items":[
			{"name":"a","get_spec":{"passport":{"cluster_name":"ce1"},"infra":{"hostname":"node-7"}}},
			{"name":"b","get_spec":{"passport":{"cluster_name":"ce1"},"infra":{"hostname":"node-8"}}}]}`)
		assert.ErrorContains(t, err, "found 2 registrations")
	})
	t.Run("no match", func(t *testing.T) {
		item, err := find(t, `{"items":[null]}`)
		require.NoError(t, err)
		assert.Nil(t, item)
	})
}

// TestExtractItemFields verifies that cluster, hostname, and passport come
// from get_spec first and fall back to object.spec.gc_spec.
func TestExtractItemFields(t *testing.T) {
	c, h := extractItemClusterAndHostname(nil)
	assert.Empty(t, c)
	assert.Empty(t, h)
	assert.Nil(t, extractItemPassport(nil))

	gcPassport := &ves_io_schema_registration.Passport{ClusterName: "gc-cluster"}
	fallback := &ves_io_schema_registration.ListResponseItem{
		GetSpec: &ves_io_schema_registration.GetSpecType{},
		Object: &ves_io_schema_registration.Object{Spec: &ves_io_schema_registration.SpecType{
			GcSpec: &ves_io_schema_registration.GlobalSpecType{
				Passport: gcPassport,
				Infra:    &ves_io_schema_registration.Infra{Hostname: "gc-host"},
			},
		}},
	}
	c, h = extractItemClusterAndHostname(fallback)
	assert.Equal(t, "gc-cluster", c)
	assert.Equal(t, "gc-host", h)
	assert.Same(t, gcPassport, extractItemPassport(fallback))

	partial := &ves_io_schema_registration.ListResponseItem{
		GetSpec: &ves_io_schema_registration.GetSpecType{
			Passport: &ves_io_schema_registration.Passport{ClusterName: "spec-cluster"},
		},
		Object: fallback.Object,
	}
	c, h = extractItemClusterAndHostname(partial)
	assert.Equal(t, "spec-cluster", c)
	assert.Equal(t, "gc-host", h)

	assert.Nil(t, extractItemPassport(&ves_io_schema_registration.ListResponseItem{}))
}

// TestBuildApprovalRequest verifies that plan overrides replace passport
// values and that tunnel_type is validated against SiteToSiteTunnelType.
func TestBuildApprovalRequest(t *testing.T) {
	base := &ves_io_schema_registration.Passport{ClusterName: "ce1", ClusterSize: 1, PrivateNetworkName: "base-net"}

	_, err := buildApprovalRequest(baseRegistrationModel(), "reg", nil, ves_io_schema_registration.APPROVED)
	assert.ErrorContains(t, err, "base passport is required")

	t.Run("overrides", func(t *testing.T) {
		m := baseRegistrationModel()
		m.PrivateNetworkName = types.StringValue(" net-1 ")
		m.Latitude = types.Float64Value(1.5)
		m.Longitude = types.Float64Value(-2.5)
		m.ConnectedRegion = types.StringValue(" region-1 ")
		m.TunnelType = types.ListValueMust(types.StringType, []attr.Value{types.StringValue("SITE_TO_SITE_TUNNEL_IPSEC")})

		req, err := buildApprovalRequest(m, "reg", base, ves_io_schema_registration.APPROVED)
		require.NoError(t, err)
		assert.Equal(t, int32(3), req.Passport.ClusterSize)
		assert.Equal(t, "net-1", req.Passport.PrivateNetworkName)
		assert.Equal(t, float32(1.5), req.Passport.Latitude)
		assert.Equal(t, float32(-2.5), req.Passport.Longitude)
		assert.Equal(t, "region-1", req.ConnectedRegion)
		assert.Equal(t, ves_io_schema.SITE_TO_SITE_TUNNEL_IPSEC, req.TunnelType)
	})

	t.Run("blank values keep base passport", func(t *testing.T) {
		m := baseRegistrationModel()
		m.ClusterSize = types.Int64Value(0)
		m.PrivateNetworkName = types.StringValue(" ")
		m.TunnelType = types.ListValueMust(types.StringType, []attr.Value{types.StringValue(" ")})

		req, err := buildApprovalRequest(m, "reg", base, ves_io_schema_registration.APPROVED)
		require.NoError(t, err)
		assert.Equal(t, int32(1), req.Passport.ClusterSize)
		assert.Equal(t, "base-net", req.Passport.PrivateNetworkName)
		assert.Equal(t, ves_io_schema.SiteToSiteTunnelType(0), req.TunnelType)
	})

	t.Run("null or unknown tunnel element is ignored", func(t *testing.T) {
		for _, v := range []attr.Value{types.StringNull(), types.StringUnknown()} {
			m := baseRegistrationModel()
			m.TunnelType = types.ListValueMust(types.StringType, []attr.Value{v})
			_, err := buildApprovalRequest(m, "reg", base, ves_io_schema_registration.APPROVED)
			assert.NoError(t, err)
		}
	})

	t.Run("invalid tunnel type", func(t *testing.T) {
		m := baseRegistrationModel()
		m.TunnelType = types.ListValueMust(types.StringType, []attr.Value{types.StringValue("BAD")})
		_, err := buildApprovalRequest(m, "reg", base, ves_io_schema_registration.APPROVED)
		assert.ErrorContains(t, err, "invalid tunnel_type")
	})

	t.Run("undecodable tunnel list", func(t *testing.T) {
		m := baseRegistrationModel()
		m.TunnelType = types.ListValueMust(types.Int64Type, []attr.Value{types.Int64Value(1)})
		_, err := buildApprovalRequest(m, "reg", base, ves_io_schema_registration.APPROVED)
		assert.ErrorContains(t, err, "unable to decode tunnel_type")
	})
}

// TestRunRegistrationApproval verifies input checks and that the namespace
// defaults to "system" before the call.
func TestRunRegistrationApproval(t *testing.T) {
	ctx := context.Background()
	assert.ErrorContains(t, runRegistrationApproval(ctx, nil, nil), "api client is required")
	c := &client.APIClient{}
	assert.ErrorContains(t, runRegistrationApproval(ctx, c, nil), "approval request is required")
	assert.ErrorContains(t, runRegistrationApproval(ctx, c, &ves_io_schema_registration.ApprovalReq{Name: " "}), "registration name is required")

	fake := &fakeRegistrationServer{}
	req := &ves_io_schema_registration.ApprovalReq{Name: "reg-1"}
	require.NoError(t, runRegistrationApproval(ctx, newRegistrationAPIClient(t, fake), req))
	assert.Equal(t, "system", req.Namespace)

	fake = &fakeRegistrationServer{approveStatus: http.StatusBadGateway}
	err := runRegistrationApproval(ctx, newRegistrationAPIClient(t, fake), &ves_io_schema_registration.ApprovalReq{Name: "reg-1"})
	assert.ErrorContains(t, err, "custom API call failed")
}

// TestResolveTimeout verifies each selector, the configured value, and the
// error paths for an invalid fallback, an invalid configured value, and an
// unknown selector.
func TestResolveTimeout(t *testing.T) {
	ctx := context.Background()
	for _, sel := range []timeoutSelector{timeoutSelectorCreate, timeoutSelectorRead, timeoutSelectorUpdate, timeoutSelectorDelete} {
		var diags diag.Diagnostics
		d, ok := resolveTimeout(ctx, testTimeoutsValue(), sel, "2m", &diags)
		assert.True(t, ok)
		assert.Equal(t, 2*time.Minute, d)
	}

	var diags diag.Diagnostics
	_, ok := resolveTimeout(ctx, testTimeoutsValue(), timeoutSelectorCreate, "bad", &diags)
	assert.False(t, ok)
	assert.Equal(t, "Invalid Duration Format", firstErrorSummary(diags))

	diags = nil
	_, ok = resolveTimeout(ctx, testTimeoutsValue(), timeoutSelector(99), "1m", &diags)
	assert.False(t, ok)
	assert.Equal(t, "Timeout Error", firstErrorSummary(diags))

	diags = nil
	_, ok = resolveTimeout(ctx, badTimeoutsValue(), timeoutSelectorRead, "1m", &diags)
	assert.False(t, ok)
}

// TestSleepWithContext verifies that the sleep ends on the timer or on
// context cancellation, whichever comes first.
func TestSleepWithContext(t *testing.T) {
	assert.NoError(t, sleepWithContext(context.Background(), time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.ErrorIs(t, sleepWithContext(ctx, time.Hour), context.Canceled)
}

// TestIsStatusNotFound verifies that only errors that carry "status code 404"
// count as not found.
func TestIsStatusNotFound(t *testing.T) {
	assert.False(t, isStatusNotFound(nil))
	assert.False(t, isStatusNotFound(io.EOF))
	assert.True(t, isStatusNotFound(errString("call failed, status code 404, body {}")))
}

type errString string

func (e errString) Error() string { return string(e) }
