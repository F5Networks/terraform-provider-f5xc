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
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ves_io_schema "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema"
	ves_io_schema_api_credential "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/api_credential"
)

func namespaceRoleObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"namespace": types.StringType,
		"role":      types.StringType,
	}}
}

func buildNamespaceRolesList(t *testing.T, pairs [][2]string) types.List {
	t.Helper()

	objType := namespaceRoleObjectType()
	elements := make([]attr.Value, 0, len(pairs))
	for _, pair := range pairs {
		obj, diags := types.ObjectValue(
			objType.AttrTypes,
			map[string]attr.Value{
				"namespace": types.StringValue(pair[0]),
				"role":      types.StringValue(pair[1]),
			},
		)
		if diags.HasError() {
			t.Fatalf("failed to create namespace role object: %v", diags)
		}
		elements = append(elements, obj)
	}

	listVal, diags := types.ListValue(objType, elements)
	if diags.HasError() {
		t.Fatalf("failed to create namespace roles list: %v", diags)
	}

	return listVal
}

func TestServiceCredentialPreUpdateHook_NoopWhenRenewEdgeNotRequested(t *testing.T) {
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
			req := &ves_io_schema_api_credential.ReplaceServiceCredentialsRequest{}

			err := ServiceCredentialPreUpdateHook(
				context.Background(),
				stub,
				req,
				types.ListNull(namespaceRoleObjectType()),
				"system",
				"svc-cred-1",
				tc.priorRenewTrigger,
				tc.planRenewTrigger,
				10,
			)
			if err != nil {
				t.Fatalf("ServiceCredentialPreUpdateHook returned error: %v", err)
			}
			if len(stub.calls) != 0 {
				t.Fatalf("expected no CustomAPI calls, got %d", len(stub.calls))
			}
			if req.NamespaceAccess != nil {
				t.Fatalf("expected req.NamespaceAccess to stay nil for null namespace_roles")
			}
		})
	}
}

func TestServiceCredentialPreUpdateHook_RenewsOnFalseToTrue(t *testing.T) {
	expiredAt := time.Now().UTC().Add(-1 * time.Hour)

	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			switch rpcFQN {
			case "ves.io.schema.api_credential.CustomAPI.GetServiceCredentials":
				return &ves_io_schema_api_credential.GetServiceCredentialsResponse{
					Type: ves_io_schema_api_credential.SERVICE_API_TOKEN,
					ExpiryTimestamp: &types_pb.Timestamp{
						Seconds: expiredAt.Unix(),
						Nanos:   int32(expiredAt.Nanosecond()),
					},
					Active: false,
				}, nil
			case "ves.io.schema.api_credential.CustomAPI.ActivateServiceCredentials":
				return nil, nil
			case "ves.io.schema.api_credential.CustomAPI.RenewServiceCredentials":
				return nil, nil
			default:
				return nil, fmt.Errorf("unexpected rpc %s", rpcFQN)
			}
		},
	}
	req := &ves_io_schema_api_credential.ReplaceServiceCredentialsRequest{}

	err := ServiceCredentialPreUpdateHook(
		context.Background(),
		stub,
		req,
		types.ListNull(namespaceRoleObjectType()),
		"system",
		"svc-cred-1",
		false,
		true,
		30,
	)
	if err != nil {
		t.Fatalf("ServiceCredentialPreUpdateHook returned error: %v", err)
	}

	if len(stub.calls) != 3 {
		t.Fatalf("expected 3 CustomAPI calls (GetServiceCredentials, ActivateServiceCredentials, RenewServiceCredentials), got %d", len(stub.calls))
	}
	if stub.calls[0].rpcFQN != "ves.io.schema.api_credential.CustomAPI.GetServiceCredentials" {
		t.Fatalf("first rpc = %q, want GetServiceCredentials", stub.calls[0].rpcFQN)
	}
	if stub.calls[1].rpcFQN != "ves.io.schema.api_credential.CustomAPI.ActivateServiceCredentials" {
		t.Fatalf("second rpc = %q, want ActivateServiceCredentials", stub.calls[1].rpcFQN)
	}
	if stub.calls[2].rpcFQN != "ves.io.schema.api_credential.CustomAPI.RenewServiceCredentials" {
		t.Fatalf("third rpc = %q, want RenewServiceCredentials", stub.calls[2].rpcFQN)
	}
}

func TestServiceCredentialPreUpdateHook_MapsNamespaceRolesToNamespaceAccess(t *testing.T) {
	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			return nil, fmt.Errorf("unexpected CustomAPI call: %s %s %s", method, uri, rpcFQN)
		},
	}
	req := &ves_io_schema_api_credential.ReplaceServiceCredentialsRequest{}
	namespaceRoles := buildNamespaceRolesList(t, [][2]string{
		{"ns-a", "admin"},
		{"ns-a", "viewer"},
		{"ns-a", "admin"},
		{"ns-b", "operator"},
	})

	err := ServiceCredentialPreUpdateHook(
		context.Background(),
		stub,
		req,
		namespaceRoles,
		"system",
		"svc-cred-1",
		false,
		false,
		30,
	)
	if err != nil {
		t.Fatalf("ServiceCredentialPreUpdateHook returned error: %v", err)
	}
	if len(stub.calls) != 0 {
		t.Fatalf("expected no CustomAPI calls, got %d", len(stub.calls))
	}

	if req.NamespaceAccess == nil {
		t.Fatal("expected NamespaceAccess to be populated from namespace_roles")
	}
	if req.NamespaceAccess.NamespaceRoleMap == nil {
		t.Fatal("expected NamespaceRoleMap to be initialized")
	}
	if got := req.NamespaceAccess.NamespaceRoleMap["ns-a"]; got == nil {
		t.Fatal("expected ns-a entry in NamespaceRoleMap")
	} else if len(got.Names) != 2 {
		t.Fatalf("expected ns-a roles to be deduplicated to 2 entries, got %v", got.Names)
	}
	if got := req.NamespaceAccess.NamespaceRoleMap["ns-b"]; got == nil || len(got.Names) != 1 || got.Names[0] != "operator" {
		t.Fatalf("unexpected ns-b roles: %#v", got)
	}
}

func TestServiceCredentialPreUpdateHook_EmptyNamespaceRolesClearsNamespaceAccess(t *testing.T) {
	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			return nil, fmt.Errorf("unexpected CustomAPI call: %s %s %s", method, uri, rpcFQN)
		},
	}
	req := &ves_io_schema_api_credential.ReplaceServiceCredentialsRequest{
		NamespaceAccess: &ves_io_schema.NamespaceAccessType{NamespaceRoleMap: map[string]*ves_io_schema.RoleListType{
			"stale": {Names: []string{"old-role"}},
		}},
	}
	emptyList, diags := types.ListValue(namespaceRoleObjectType(), []attr.Value{})
	if diags.HasError() {
		t.Fatalf("failed to create empty namespace_roles list: %v", diags)
	}

	err := ServiceCredentialPreUpdateHook(
		context.Background(),
		stub,
		req,
		emptyList,
		"system",
		"svc-cred-1",
		false,
		false,
		30,
	)
	if err != nil {
		t.Fatalf("ServiceCredentialPreUpdateHook returned error: %v", err)
	}
	if req.NamespaceAccess == nil {
		t.Fatal("expected NamespaceAccess to be set for explicit empty namespace_roles")
	}
	if req.NamespaceAccess.NamespaceRoleMap == nil {
		t.Fatal("expected NamespaceRoleMap to be initialized for clear request")
	}
	if len(req.NamespaceAccess.NamespaceRoleMap) != 0 {
		t.Fatalf("expected empty NamespaceRoleMap to clear namespace access, got %#v", req.NamespaceAccess.NamespaceRoleMap)
	}
}

func TestServiceCredentialPreUpdateHook_PropagatesEffectiveNamespace(t *testing.T) {
	t.Run("sets_req_namespace_when_renew_not_triggered", func(t *testing.T) {
		stub := &stubCustomAPIClient{
			handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
				return nil, fmt.Errorf("unexpected CustomAPI call: %s %s %s", method, uri, rpcFQN)
			},
		}
		req := &ves_io_schema_api_credential.ReplaceServiceCredentialsRequest{
			Name:      "tf-name",
			Namespace: "system",
		}

		err := ServiceCredentialPreUpdateHook(
			context.Background(),
			stub,
			req,
			types.ListNull(namespaceRoleObjectType()),
			"custom-vk8s",
			"svc-cred-1",
			false,
			false,
			30,
		)
		if err != nil {
			t.Fatalf("ServiceCredentialPreUpdateHook returned error: %v", err)
		}
		if req.Namespace != "custom-vk8s" {
			t.Fatalf("req.Namespace = %q, want %q", req.Namespace, "custom-vk8s")
		}
		if req.Name != "svc-cred-1" {
			t.Fatalf("req.Name = %q, want %q", req.Name, "svc-cred-1")
		}
	})

	t.Run("uses_effective_namespace_for_renew_calls_and_replace_request", func(t *testing.T) {
		expiredAt := time.Now().UTC().Add(-1 * time.Hour)

		stub := &stubCustomAPIClient{
			handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
				switch rpcFQN {
				case "ves.io.schema.api_credential.CustomAPI.GetServiceCredentials":
					return &ves_io_schema_api_credential.GetServiceCredentialsResponse{
						Type: ves_io_schema_api_credential.SERVICE_API_TOKEN,
						ExpiryTimestamp: &types_pb.Timestamp{
							Seconds: expiredAt.Unix(),
							Nanos:   int32(expiredAt.Nanosecond()),
						},
						Active: false,
					}, nil
				case "ves.io.schema.api_credential.CustomAPI.ActivateServiceCredentials":
					return nil, nil
				case "ves.io.schema.api_credential.CustomAPI.RenewServiceCredentials":
					return nil, nil
				default:
					return nil, fmt.Errorf("unexpected rpc %s", rpcFQN)
				}
			},
		}
		req := &ves_io_schema_api_credential.ReplaceServiceCredentialsRequest{
			Namespace: "system",
		}

		err := ServiceCredentialPreUpdateHook(
			context.Background(),
			stub,
			req,
			types.ListNull(namespaceRoleObjectType()),
			"custom-vk8s",
			"svc-cred-1",
			false,
			true,
			30,
		)
		if err != nil {
			t.Fatalf("ServiceCredentialPreUpdateHook returned error: %v", err)
		}
		if req.Namespace != "custom-vk8s" {
			t.Fatalf("req.Namespace = %q, want %q", req.Namespace, "custom-vk8s")
		}
		if len(stub.calls) != 3 {
			t.Fatalf("expected 3 CustomAPI calls, got %d", len(stub.calls))
		}
		if got, want := stub.calls[0].uri, "/public/namespaces/custom-vk8s/service_credentials/svc-cred-1"; got != want {
			t.Fatalf("GetServiceCredentials uri = %q, want %q", got, want)
		}
		if got, want := stub.calls[1].uri, "/public/namespaces/custom-vk8s/activate/service_credentials"; got != want {
			t.Fatalf("ActivateServiceCredentials uri = %q, want %q", got, want)
		}
		if got, want := stub.calls[2].uri, "/public/namespaces/custom-vk8s/renew/service_credentials"; got != want {
			t.Fatalf("RenewServiceCredentials uri = %q, want %q", got, want)
		}
	})
}

func TestServiceCredentialOverrideImportHook(t *testing.T) {
	testSchema := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"namespace":       schema.StringAttribute{Required: true},
			"credential_name": schema.StringAttribute{Computed: true},
			"service_credential_choice": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"api_certificate": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"password": schema.StringAttribute{Optional: true},
						},
					},
					"api_token": schema.BoolAttribute{Optional: true},
					"site_kubeconfig": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"site": schema.StringAttribute{Optional: true},
						},
					},
					"vk8s_kubeconfig": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"vk8s_cluster_name": schema.StringAttribute{Optional: true},
							"vk8s_namespace":    schema.StringAttribute{Optional: true},
						},
					},
				},
			},
		},
	}

	t.Run("system_two_part", func(t *testing.T) {
		ctx := context.Background()
		resp := &resource.ImportStateResponse{State: newEmptyStateForSchema(testSchema)}
		ServiceCredentialOverrideImportHook(ctx, resource.ImportStateRequest{ID: "system/svc-123"}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
		}
		var ns, credName types.String
		var choice types.Object
		mustGetAttribute(t, ctx, resp.State, path.Root("namespace"), &ns)
		mustGetAttribute(t, ctx, resp.State, path.Root("credential_name"), &credName)
		mustGetAttribute(t, ctx, resp.State, path.Root("service_credential_choice"), &choice)
		if ns.ValueString() != "system" || credName.ValueString() != "svc-123" {
			t.Fatalf("got (%q, %q), want (system, svc-123)", ns.ValueString(), credName.ValueString())
		}
		if !choice.IsNull() {
			t.Fatalf("expected service_credential_choice to remain null for system credential, got %v", choice)
		}
	})

	t.Run("vk8s_two_part", func(t *testing.T) {
		ctx := context.Background()
		resp := &resource.ImportStateResponse{State: newEmptyStateForSchema(testSchema)}
		ServiceCredentialOverrideImportHook(ctx, resource.ImportStateRequest{ID: "my-vk8s-ns/svc-456"}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
		}
		var ns, credName, vk8sNs types.String
		mustGetAttribute(t, ctx, resp.State, path.Root("namespace"), &ns)
		mustGetAttribute(t, ctx, resp.State, path.Root("credential_name"), &credName)
		mustGetAttribute(t, ctx, resp.State, path.Root("service_credential_choice").AtName("vk8s_kubeconfig").AtName("vk8s_namespace"), &vk8sNs)
		if ns.ValueString() != "system" || credName.ValueString() != "svc-456" || vk8sNs.ValueString() != "my-vk8s-ns" {
			t.Fatalf("got (%q, %q, %q), want (system, svc-456, my-vk8s-ns)", ns.ValueString(), credName.ValueString(), vk8sNs.ValueString())
		}
	})

	t.Run("vk8s_three_part", func(t *testing.T) {
		ctx := context.Background()
		resp := &resource.ImportStateResponse{State: newEmptyStateForSchema(testSchema)}
		ServiceCredentialOverrideImportHook(ctx, resource.ImportStateRequest{ID: "system/my-vk8s-ns/svc-789"}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
		}
		var ns, credName, vk8sNs types.String
		mustGetAttribute(t, ctx, resp.State, path.Root("namespace"), &ns)
		mustGetAttribute(t, ctx, resp.State, path.Root("credential_name"), &credName)
		mustGetAttribute(t, ctx, resp.State, path.Root("service_credential_choice").AtName("vk8s_kubeconfig").AtName("vk8s_namespace"), &vk8sNs)
		if ns.ValueString() != "system" || credName.ValueString() != "svc-789" || vk8sNs.ValueString() != "my-vk8s-ns" {
			t.Fatalf("got (%q, %q, %q), want (system, svc-789, my-vk8s-ns)", ns.ValueString(), credName.ValueString(), vk8sNs.ValueString())
		}
	})

	t.Run("invalid_ids", func(t *testing.T) {
		for _, id := range []string{"", "only-one", "my-vk8s-ns/", "system//cred", "/ns/cred", "system/ns/", "other/ns/cred", "a/b/c/d"} {
			resp := &resource.ImportStateResponse{State: newEmptyStateForSchema(testSchema)}
			ServiceCredentialOverrideImportHook(context.Background(), resource.ImportStateRequest{ID: id}, resp)
			if !resp.Diagnostics.HasError() {
				t.Fatalf("expected error for invalid import ID %q", id)
			}
		}
	})
}

func TestServiceCredentialPreUpdateHook_InputGuards(t *testing.T) {
	noCalls := func(t *testing.T) *stubCustomAPIClient {
		return &stubCustomAPIClient{
			handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
				t.Fatalf("unexpected CustomAPI call: %s %s %s", method, uri, rpcFQN)
				return nil, nil
			},
		}
	}

	t.Run("nil_request_returns_error", func(t *testing.T) {
		err := ServiceCredentialPreUpdateHook(context.Background(), noCalls(t), nil, types.ListNull(namespaceRoleObjectType()), "my-vk8s-ns", "svc-cred-1", false, false, 30)
		if err == nil {
			t.Fatal("expected an error for a nil request")
		}
	})

	t.Run("empty_credential_name_keeps_request_name", func(t *testing.T) {
		req := &ves_io_schema_api_credential.ReplaceServiceCredentialsRequest{Name: "tf-name", Namespace: "system"}
		err := ServiceCredentialPreUpdateHook(context.Background(), noCalls(t), req, types.ListNull(namespaceRoleObjectType()), "my-vk8s-ns", "", false, false, 30)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if req.Name != "tf-name" {
			t.Fatalf("req.Name = %q, want tf-name to be kept when credentialName is empty", req.Name)
		}
		if req.Namespace != "my-vk8s-ns" {
			t.Fatalf("req.Namespace = %q, want my-vk8s-ns", req.Namespace)
		}
	})

	t.Run("empty_namespace_defaults_to_system", func(t *testing.T) {
		req := &ves_io_schema_api_credential.ReplaceServiceCredentialsRequest{Name: "tf-name", Namespace: "stale"}
		err := ServiceCredentialPreUpdateHook(context.Background(), noCalls(t), req, types.ListNull(namespaceRoleObjectType()), "", "svc-cred-1", false, false, 30)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if req.Namespace != "system" {
			t.Fatalf("req.Namespace = %q, want system", req.Namespace)
		}
		if req.Name != "svc-cred-1" {
			t.Fatalf("req.Name = %q, want svc-cred-1", req.Name)
		}
	})
}

const (
	svcCredGet      = "ves.io.schema.api_credential.CustomAPI.GetServiceCredentials"
	svcCredActivate = "ves.io.schema.api_credential.CustomAPI.ActivateServiceCredentials"
	svcCredRenew    = "ves.io.schema.api_credential.CustomAPI.RenewServiceCredentials"
)

// The service_credential renew flow stops at each guard and returns each API error.
func TestServiceCredentialPreUpdateHook_Branches(t *testing.T) {
	expired := tsFrom(time.Now().Add(-time.Hour))
	future := tsFrom(time.Now().Add(time.Hour))
	boom := errors.New("boom")
	expiredInactive := &ves_io_schema_api_credential.GetServiceCredentialsResponse{
		Type: ves_io_schema_api_credential.SERVICE_API_TOKEN, ExpiryTimestamp: expired,
	}

	cases := []struct {
		name      string
		resps     map[string]proto.Message
		errs      map[string]error
		wantErr   string
		wantCalls int
	}{
		{"get error", nil, map[string]error{svcCredGet: boom}, "failed to get service_credential", 1},
		{"wrong get type", map[string]proto.Message{svcCredGet: &ves_io_schema_api_credential.RenewRequest{}}, nil, "unexpected GetServiceCredentials response type", 1},
		{"not api token", map[string]proto.Message{svcCredGet: &ves_io_schema_api_credential.GetServiceCredentialsResponse{
			Type: ves_io_schema_api_credential.SERVICE_KUBE_CONFIG,
		}}, nil, "", 1},
		{"no expiry", map[string]proto.Message{svcCredGet: &ves_io_schema_api_credential.GetServiceCredentialsResponse{
			Type: ves_io_schema_api_credential.SERVICE_API_TOKEN,
		}}, nil, "", 1},
		{"not expired", map[string]proto.Message{svcCredGet: &ves_io_schema_api_credential.GetServiceCredentialsResponse{
			Type: ves_io_schema_api_credential.SERVICE_API_TOKEN, ExpiryTimestamp: future,
		}}, nil, "", 1},
		{"still active", map[string]proto.Message{svcCredGet: &ves_io_schema_api_credential.GetServiceCredentialsResponse{
			Type: ves_io_schema_api_credential.SERVICE_API_TOKEN, ExpiryTimestamp: expired, Active: true,
		}}, nil, "", 1},
		{"activate error", map[string]proto.Message{svcCredGet: expiredInactive}, map[string]error{svcCredActivate: boom}, "failed to activate", 2},
		{"renew error", map[string]proto.Message{svcCredGet: expiredInactive}, map[string]error{svcCredRenew: boom}, "failed to renew", 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := rpcStub(tc.resps, tc.errs)
			req := &ves_io_schema_api_credential.ReplaceServiceCredentialsRequest{}
			err := ServiceCredentialPreUpdateHook(context.Background(), stub, req, types.ListNull(namespaceRoleObjectType()), "system", "svc-1", false, true, 0)
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantErr)
			}
			assert.Len(t, stub.calls, tc.wantCalls)
		})
	}
}

// A namespace_roles list that cannot decode stops the hook before any API call.
func TestServiceCredentialPreUpdateHook_BadNamespaceRoles(t *testing.T) {
	bad, diags := types.ListValue(types.StringType, []attr.Value{types.StringValue("x")})
	require.False(t, diags.HasError())
	stub := &stubCustomAPIClient{}
	err := ServiceCredentialPreUpdateHook(context.Background(), stub, &ves_io_schema_api_credential.ReplaceServiceCredentialsRequest{}, bad, "system", "svc-1", false, true, 0)
	require.ErrorContains(t, err, "failed to decode namespace_roles")
	assert.Empty(t, stub.calls)
}

// Role entries with null, unknown, or blank fields are dropped.
func TestBuildNamespaceAccessFromRoles_SkipsIncompleteEntries(t *testing.T) {
	objType := namespaceRoleObjectType()
	mk := func(ns, role attr.Value) attr.Value {
		v, d := types.ObjectValue(objType.AttrTypes, map[string]attr.Value{"namespace": ns, "role": role})
		require.False(t, d.HasError())
		return v
	}
	list, diags := types.ListValue(objType, []attr.Value{
		mk(types.StringNull(), types.StringValue("r")),
		mk(types.StringValue("ns"), types.StringUnknown()),
		mk(types.StringValue(" "), types.StringValue("r")),
		mk(types.StringValue("ns"), types.StringValue("admin")),
	})
	require.False(t, diags.HasError())

	access, ok, err := buildNamespaceAccessFromRoles(context.Background(), list)
	require.NoError(t, err)
	require.True(t, ok)
	require.Len(t, access.GetNamespaceRoleMap(), 1)
	assert.Equal(t, []string{"admin"}, access.GetNamespaceRoleMap()["ns"].GetNames())
}
