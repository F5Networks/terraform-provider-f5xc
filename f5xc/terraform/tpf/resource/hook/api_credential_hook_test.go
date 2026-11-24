//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	types_pb "github.com/gogo/protobuf/types"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.volterra.us/stdlib/client/vesapi"

	ves_io_schema_api_credential "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/api_credential"
)

type customAPICall struct {
	method string
	uri    string
	rpcFQN string
	reqYml string
}

type stubCustomAPIClient struct {
	calls   []customAPICall
	handler func(method, uri, rpcFQN, reqYml string) (proto.Message, error)
}

func (s *stubCustomAPIClient) CustomAPI(ctx context.Context, method, uri, rpcFQN, reqYml string, opts ...vesapi.CallOpt) (proto.Message, error) {
	s.calls = append(s.calls, customAPICall{method: method, uri: uri, rpcFQN: rpcFQN, reqYml: reqYml})
	if s.handler == nil {
		return nil, nil
	}
	return s.handler(method, uri, rpcFQN, reqYml)
}

func TestApiCredentialOverrideUpdateHook_NoopWhenRenewEdgeNotRequested(t *testing.T) {
	testCases := []struct {
		name              string
		priorRenewTrigger bool
		planRenewTrigger  bool
	}{
		{name: "false_to_false", priorRenewTrigger: false, planRenewTrigger: false},
		{name: "true_to_true", priorRenewTrigger: true, planRenewTrigger: true},
		{name: "true_to_false", priorRenewTrigger: true, planRenewTrigger: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &stubCustomAPIClient{
				handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
					return nil, fmt.Errorf("unexpected CustomAPI call: %s %s %s", method, uri, rpcFQN)
				},
			}

			err := ApiCredentialOverrideUpdateHook(
				context.Background(),
				stub,
				"system",
				"cred-1",
				tc.priorRenewTrigger,
				tc.planRenewTrigger,
				10,
			)
			if err != nil {
				t.Fatalf("ApiCredentialOverrideUpdateHook returned error: %v", err)
			}
			if len(stub.calls) != 0 {
				t.Fatalf("expected no CustomAPI calls, got %d", len(stub.calls))
			}
		})
	}
}

func TestApiCredentialOverrideUpdateHook_RenewsOnFalseToTrue(t *testing.T) {
	expiredAt := time.Now().UTC().Add(-1 * time.Hour)

	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			switch rpcFQN {
			case "ves.io.schema.api_credential.CustomAPI.Get":
				return &ves_io_schema_api_credential.GetResponse{
					Object: &ves_io_schema_api_credential.Object{
						Spec: &ves_io_schema_api_credential.SpecType{
							GcSpec: &ves_io_schema_api_credential.GlobalSpecType{
								Type: ves_io_schema_api_credential.API_TOKEN,
								ExpirationTimestamp: &types_pb.Timestamp{
									Seconds: expiredAt.Unix(),
									Nanos:   int32(expiredAt.Nanosecond()),
								},
								Active: false,
							},
						},
					},
				}, nil
			case "ves.io.schema.api_credential.CustomAPI.Activate":
				return nil, nil
			case "ves.io.schema.api_credential.CustomAPI.Renew":
				return nil, nil
			default:
				return nil, fmt.Errorf("unexpected rpc %s", rpcFQN)
			}
		},
	}

	err := ApiCredentialOverrideUpdateHook(
		context.Background(),
		stub,
		"system",
		"cred-1",
		false,
		true,
		30,
	)
	if err != nil {
		t.Fatalf("ApiCredentialOverrideUpdateHook returned error: %v", err)
	}

	if len(stub.calls) != 3 {
		t.Fatalf("expected 3 CustomAPI calls (Get, Activate, Renew), got %d", len(stub.calls))
	}
	if stub.calls[0].rpcFQN != "ves.io.schema.api_credential.CustomAPI.Get" {
		t.Fatalf("first rpc = %q, want Get", stub.calls[0].rpcFQN)
	}
	if stub.calls[1].rpcFQN != "ves.io.schema.api_credential.CustomAPI.Activate" {
		t.Fatalf("second rpc = %q, want Activate", stub.calls[1].rpcFQN)
	}
	if stub.calls[2].rpcFQN != "ves.io.schema.api_credential.CustomAPI.Renew" {
		t.Fatalf("third rpc = %q, want Renew", stub.calls[2].rpcFQN)
	}
}

func TestApiCredentialOverrideUpdateHook_UsesEffectiveNamespace(t *testing.T) {
	expiredAt := time.Now().UTC().Add(-1 * time.Hour)

	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			switch rpcFQN {
			case "ves.io.schema.api_credential.CustomAPI.Get":
				return &ves_io_schema_api_credential.GetResponse{
					Object: &ves_io_schema_api_credential.Object{
						Spec: &ves_io_schema_api_credential.SpecType{
							GcSpec: &ves_io_schema_api_credential.GlobalSpecType{
								Type: ves_io_schema_api_credential.API_TOKEN,
								ExpirationTimestamp: &types_pb.Timestamp{
									Seconds: expiredAt.Unix(),
									Nanos:   int32(expiredAt.Nanosecond()),
								},
								Active: false,
							},
						},
					},
				}, nil
			case "ves.io.schema.api_credential.CustomAPI.Activate":
				return nil, nil
			case "ves.io.schema.api_credential.CustomAPI.Renew":
				return nil, nil
			default:
				return nil, fmt.Errorf("unexpected rpc %s", rpcFQN)
			}
		},
	}

	err := ApiCredentialOverrideUpdateHook(
		context.Background(),
		stub,
		"custom-vk8s",
		"cred-1",
		false,
		true,
		30,
	)
	if err != nil {
		t.Fatalf("ApiCredentialOverrideUpdateHook returned error: %v", err)
	}

	if len(stub.calls) != 3 {
		t.Fatalf("expected 3 CustomAPI calls, got %d", len(stub.calls))
	}
	if got, want := stub.calls[0].uri, "/public/namespaces/custom-vk8s/api_credentials/cred-1"; got != want {
		t.Fatalf("Get uri = %q, want %q", got, want)
	}
	if got, want := stub.calls[1].uri, "/public/namespaces/custom-vk8s/activate/api_credentials"; got != want {
		t.Fatalf("Activate uri = %q, want %q", got, want)
	}
	if got, want := stub.calls[2].uri, "/public/namespaces/custom-vk8s/renew/api_credentials"; got != want {
		t.Fatalf("Renew uri = %q, want %q", got, want)
	}
}

// mustGetAttribute reads the attribute at p from state into target and fails the
// test when the lookup returns error diagnostics (for example, a wrong path or type).
func mustGetAttribute(t *testing.T, ctx context.Context, state tfsdk.State, p path.Path, target interface{}) {
	t.Helper()
	if diags := state.GetAttribute(ctx, p, target); diags.HasError() {
		t.Fatalf("GetAttribute(%s) diagnostics: %v", p, diags)
	}
}

func newEmptyStateForSchema(s schema.Schema) tfsdk.State {
	objType := s.Type().TerraformType(context.Background()).(tftypes.Object)
	vals := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for k, t := range objType.AttributeTypes {
		vals[k] = tftypes.NewValue(t, nil)
	}
	return tfsdk.State{
		Schema: s,
		Raw:    tftypes.NewValue(objType, vals),
	}
}

func TestApiCredentialOverrideImportHook(t *testing.T) {
	testSchema := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"namespace":       schema.StringAttribute{Required: true},
			"credential_name": schema.StringAttribute{Computed: true},
			"spec": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"password":              schema.StringAttribute{Optional: true},
					"type":                  schema.ListAttribute{ElementType: types.StringType, Optional: true},
					"virtual_k8s_name":      schema.StringAttribute{Optional: true},
					"virtual_k8s_namespace": schema.StringAttribute{Optional: true},
				},
			},
		},
	}

	t.Run("system_two_part", func(t *testing.T) {
		ctx := context.Background()
		resp := &resource.ImportStateResponse{State: newEmptyStateForSchema(testSchema)}
		ApiCredentialOverrideImportHook(ctx, resource.ImportStateRequest{ID: "system/cred-123"}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
		}
		var ns, credName types.String
		var spec types.Object
		mustGetAttribute(t, ctx, resp.State, path.Root("namespace"), &ns)
		mustGetAttribute(t, ctx, resp.State, path.Root("credential_name"), &credName)
		mustGetAttribute(t, ctx, resp.State, path.Root("spec"), &spec)
		if ns.ValueString() != "system" {
			t.Fatalf("namespace = %q, want system", ns.ValueString())
		}
		if credName.ValueString() != "cred-123" {
			t.Fatalf("credential_name = %q, want cred-123", credName.ValueString())
		}
		if !spec.IsNull() {
			t.Fatalf("expected spec to remain null for system credential, got %v", spec)
		}
	})

	t.Run("vk8s_two_part", func(t *testing.T) {
		ctx := context.Background()
		resp := &resource.ImportStateResponse{State: newEmptyStateForSchema(testSchema)}
		ApiCredentialOverrideImportHook(ctx, resource.ImportStateRequest{ID: "my-vk8s-ns/cred-456"}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
		}
		var ns, credName, vk8sNs types.String
		mustGetAttribute(t, ctx, resp.State, path.Root("namespace"), &ns)
		mustGetAttribute(t, ctx, resp.State, path.Root("credential_name"), &credName)
		mustGetAttribute(t, ctx, resp.State, path.Root("spec").AtName("virtual_k8s_namespace"), &vk8sNs)
		if ns.ValueString() != "system" {
			t.Fatalf("namespace = %q, want system", ns.ValueString())
		}
		if credName.ValueString() != "cred-456" {
			t.Fatalf("credential_name = %q, want cred-456", credName.ValueString())
		}
		if vk8sNs.ValueString() != "my-vk8s-ns" {
			t.Fatalf("spec.virtual_k8s_namespace = %q, want my-vk8s-ns", vk8sNs.ValueString())
		}
	})

	t.Run("vk8s_three_part", func(t *testing.T) {
		ctx := context.Background()
		resp := &resource.ImportStateResponse{State: newEmptyStateForSchema(testSchema)}
		ApiCredentialOverrideImportHook(ctx, resource.ImportStateRequest{ID: "system/my-vk8s-ns/cred-789"}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
		}
		var ns, credName, vk8sNs types.String
		mustGetAttribute(t, ctx, resp.State, path.Root("namespace"), &ns)
		mustGetAttribute(t, ctx, resp.State, path.Root("credential_name"), &credName)
		mustGetAttribute(t, ctx, resp.State, path.Root("spec").AtName("virtual_k8s_namespace"), &vk8sNs)
		if ns.ValueString() != "system" || credName.ValueString() != "cred-789" || vk8sNs.ValueString() != "my-vk8s-ns" {
			t.Fatalf("got (%q, %q, %q), want (system, cred-789, my-vk8s-ns)", ns.ValueString(), credName.ValueString(), vk8sNs.ValueString())
		}
	})

	t.Run("invalid_ids", func(t *testing.T) {
		for _, id := range []string{"", "only-one", "/cred", "system/", "other/ns/cred", "a/b/c/d"} {
			resp := &resource.ImportStateResponse{State: newEmptyStateForSchema(testSchema)}
			ApiCredentialOverrideImportHook(context.Background(), resource.ImportStateRequest{ID: id}, resp)
			if !resp.Diagnostics.HasError() {
				t.Fatalf("expected error for invalid import ID %q", id)
			}
		}
	})
}

// TestCredentialImportHooks_FollowSchema verifies that the import hooks derive nested
// attribute types from the resource schema instead of a hard-coded copy, so a schema
// that gains an extra nested attribute still imports without a type mismatch.
func TestCredentialImportHooks_FollowSchema(t *testing.T) {
	ctx := context.Background()

	t.Run("api_credential_spec_with_extra_attribute", func(t *testing.T) {
		s := schema.Schema{
			Attributes: map[string]schema.Attribute{
				"namespace":       schema.StringAttribute{Required: true},
				"credential_name": schema.StringAttribute{Computed: true},
				"spec": schema.SingleNestedAttribute{
					Optional: true,
					Attributes: map[string]schema.Attribute{
						"virtual_k8s_namespace": schema.StringAttribute{Optional: true},
						"type":                  schema.ListAttribute{ElementType: types.StringType, Optional: true},
						"new_field":             schema.Int64Attribute{Optional: true},
					},
				},
			},
		}
		resp := &resource.ImportStateResponse{State: newEmptyStateForSchema(s)}
		ApiCredentialOverrideImportHook(ctx, resource.ImportStateRequest{ID: "my-vk8s-ns/cred-1"}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
		}
		var vk8sNs types.String
		var newField types.Int64
		mustGetAttribute(t, ctx, resp.State, path.Root("spec").AtName("virtual_k8s_namespace"), &vk8sNs)
		mustGetAttribute(t, ctx, resp.State, path.Root("spec").AtName("new_field"), &newField)
		if vk8sNs.ValueString() != "my-vk8s-ns" {
			t.Fatalf("spec.virtual_k8s_namespace = %q, want my-vk8s-ns", vk8sNs.ValueString())
		}
		if !newField.IsNull() {
			t.Fatalf("spec.new_field = %v, want null", newField)
		}
	})

	t.Run("service_credential_choice_with_extra_attribute", func(t *testing.T) {
		s := schema.Schema{
			Attributes: map[string]schema.Attribute{
				"namespace":       schema.StringAttribute{Required: true},
				"credential_name": schema.StringAttribute{Computed: true},
				"service_credential_choice": schema.SingleNestedAttribute{
					Optional: true,
					Attributes: map[string]schema.Attribute{
						"api_token": schema.BoolAttribute{Optional: true},
						"vk8s_kubeconfig": schema.SingleNestedAttribute{
							Optional: true,
							Attributes: map[string]schema.Attribute{
								"vk8s_namespace": schema.StringAttribute{Optional: true},
								"new_field":      schema.StringAttribute{Optional: true},
							},
						},
						"new_choice": schema.SetAttribute{ElementType: types.StringType, Optional: true},
					},
				},
			},
		}
		resp := &resource.ImportStateResponse{State: newEmptyStateForSchema(s)}
		ServiceCredentialOverrideImportHook(ctx, resource.ImportStateRequest{ID: "system/my-vk8s-ns/svc-1"}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
		}
		var vk8sNs types.String
		mustGetAttribute(t, ctx, resp.State, path.Root("service_credential_choice").AtName("vk8s_kubeconfig").AtName("vk8s_namespace"), &vk8sNs)
		if vk8sNs.ValueString() != "my-vk8s-ns" {
			t.Fatalf("vk8s_namespace = %q, want my-vk8s-ns", vk8sNs.ValueString())
		}
	})

	t.Run("missing_leaf_returns_error", func(t *testing.T) {
		s := schema.Schema{
			Attributes: map[string]schema.Attribute{
				"namespace":       schema.StringAttribute{Required: true},
				"credential_name": schema.StringAttribute{Computed: true},
				"spec": schema.SingleNestedAttribute{
					Optional:   true,
					Attributes: map[string]schema.Attribute{"type": schema.ListAttribute{ElementType: types.StringType, Optional: true}},
				},
			},
		}
		resp := &resource.ImportStateResponse{State: newEmptyStateForSchema(s)}
		ApiCredentialOverrideImportHook(ctx, resource.ImportStateRequest{ID: "my-vk8s-ns/cred-1"}, resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected an error when spec.virtual_k8s_namespace is missing from the schema")
		}
	})
}

func tsFrom(t time.Time) *types_pb.Timestamp {
	return &types_pb.Timestamp{Seconds: t.Unix(), Nanos: int32(t.Nanosecond())}
}

// rpcStub answers each rpcFQN with the response or error in the maps.
func rpcStub(resps map[string]proto.Message, errs map[string]error) *stubCustomAPIClient {
	return &stubCustomAPIClient{handler: func(_, _, rpcFQN, _ string) (proto.Message, error) {
		return resps[rpcFQN], errs[rpcFQN]
	}}
}

const (
	apiCredGet      = "ves.io.schema.api_credential.CustomAPI.Get"
	apiCredActivate = "ves.io.schema.api_credential.CustomAPI.Activate"
	apiCredRenew    = "ves.io.schema.api_credential.CustomAPI.Renew"
)

func apiCredGetResp(spec *ves_io_schema_api_credential.GlobalSpecType) *ves_io_schema_api_credential.GetResponse {
	return &ves_io_schema_api_credential.GetResponse{
		Object: &ves_io_schema_api_credential.Object{Spec: &ves_io_schema_api_credential.SpecType{GcSpec: spec}},
	}
}

// The api_credential renew flow stops at each guard and returns each API error.
func TestApiCredentialOverrideUpdateHook_Branches(t *testing.T) {
	expired := tsFrom(time.Now().Add(-time.Hour))
	future := tsFrom(time.Now().Add(time.Hour))
	boom := errors.New("boom")
	expiredInactive := apiCredGetResp(&ves_io_schema_api_credential.GlobalSpecType{
		Type: ves_io_schema_api_credential.API_TOKEN, ExpirationTimestamp: expired,
	})

	cases := []struct {
		name      string
		resps     map[string]proto.Message
		errs      map[string]error
		wantErr   string
		wantCalls int
	}{
		{"get error", nil, map[string]error{apiCredGet: boom}, "failed to get api_credential", 1},
		{"wrong get type", map[string]proto.Message{apiCredGet: &ves_io_schema_api_credential.RenewRequest{}}, nil, "unexpected Get response type", 1},
		{"no gc spec", map[string]proto.Message{apiCredGet: apiCredGetResp(nil)}, nil, "", 1},
		{"not api token", map[string]proto.Message{apiCredGet: apiCredGetResp(&ves_io_schema_api_credential.GlobalSpecType{
			Type: ves_io_schema_api_credential.KUBE_CONFIG,
		})}, nil, "", 1},
		{"no expiry", map[string]proto.Message{apiCredGet: apiCredGetResp(&ves_io_schema_api_credential.GlobalSpecType{
			Type: ves_io_schema_api_credential.API_TOKEN,
		})}, nil, "", 1},
		{"not expired", map[string]proto.Message{apiCredGet: apiCredGetResp(&ves_io_schema_api_credential.GlobalSpecType{
			Type: ves_io_schema_api_credential.API_TOKEN, ExpirationTimestamp: future,
		})}, nil, "", 1},
		{"still active", map[string]proto.Message{apiCredGet: apiCredGetResp(&ves_io_schema_api_credential.GlobalSpecType{
			Type: ves_io_schema_api_credential.API_TOKEN, ExpirationTimestamp: expired, Active: true,
		})}, nil, "", 1},
		{"activate error", map[string]proto.Message{apiCredGet: expiredInactive}, map[string]error{apiCredActivate: boom}, "failed to activate", 2},
		{"renew error", map[string]proto.Message{apiCredGet: expiredInactive}, map[string]error{apiCredRenew: boom}, "failed to renew", 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := rpcStub(tc.resps, tc.errs)
			err := ApiCredentialOverrideUpdateHook(context.Background(), stub, "system", "cred-1", false, true, 0)
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantErr)
			}
			assert.Len(t, stub.calls, tc.wantCalls)
		})
	}
}
