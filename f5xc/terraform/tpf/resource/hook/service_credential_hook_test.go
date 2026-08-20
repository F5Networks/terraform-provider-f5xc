//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	types_pb "github.com/gogo/protobuf/types"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

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
