//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	"gopkg.volterra.us/stdlib/client/vesapi"

	ves_io_schema_namespace "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/namespace"
	"gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/views"
)

type setNetworkPolicyReqBody struct {
	Namespace       string             `json:"namespace"`
	NetworkPolicies []networkPolicyRef `json:"network_policies"`
}

type networkPolicyRef struct {
	Tenant    string `json:"tenant,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
}

func newNetworkObjectRef(tenant, namespace, name string) *views.ObjectRefType {
	return &views.ObjectRefType{
		Tenant:    tenant,
		Namespace: namespace,
		Name:      name,
	}
}

func TestNetworkPolicyPreDeleteHook_PolicyNotInActiveList(t *testing.T) {
	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			if method == "GET" {
				return &ves_io_schema_namespace.GetActiveNetworkPoliciesResponse{
					NetworkPolicies: []*views.ObjectRefType{
						newNetworkObjectRef("acme", "default", "other-policy"),
					},
				}, nil
			}
			return nil, fmt.Errorf("unexpected CustomAPI call: %s %s", method, rpcFQN)
		},
	}

	err := NetworkPolicyPreDeleteHook(context.Background(), stub, "default", "my-policy")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(stub.calls) != 1 {
		t.Fatalf("expected 1 CustomAPI call (GET), got %d", len(stub.calls))
	}
	if stub.calls[0].method != "GET" {
		t.Fatalf("expected GET call, got %s", stub.calls[0].method)
	}
}

func TestNetworkPolicyPreDeleteHook_PolicyInActiveList_MultiplePolicies(t *testing.T) {
	const (
		ns         = "production"
		policyName = "delete-me"
	)

	remaining := []*views.ObjectRefType{
		newNetworkObjectRef("acme", ns, "keep-policy-1"),
		newNetworkObjectRef("acme", ns, "keep-policy-2"),
	}

	var capturedBody string

	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			switch method {
			case "GET":
				allPolicies := append(remaining, newNetworkObjectRef("acme", ns, policyName))
				return &ves_io_schema_namespace.GetActiveNetworkPoliciesResponse{
					NetworkPolicies: allPolicies,
				}, nil
			case "POST":
				capturedBody = reqYml
				return nil, nil
			default:
				return nil, fmt.Errorf("unexpected method %s", method)
			}
		},
	}

	err := NetworkPolicyPreDeleteHook(context.Background(), stub, ns, policyName)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(stub.calls) != 2 {
		t.Fatalf("expected 2 CustomAPI calls (GET + POST), got %d", len(stub.calls))
	}

	var body setNetworkPolicyReqBody
	if err := json.Unmarshal([]byte(capturedBody), &body); err != nil {
		t.Fatalf("failed to unmarshal POST body: %v", err)
	}

	if body.Namespace != ns {
		t.Errorf("POST body namespace = %q, want %q", body.Namespace, ns)
	}
	if len(body.NetworkPolicies) != 2 {
		t.Fatalf("POST body network_policies len = %d, want 2", len(body.NetworkPolicies))
	}

	for _, ref := range body.NetworkPolicies {
		if ref.Name == policyName {
			t.Errorf("deleted policy %q should not be in the POST body", policyName)
		}
	}
}

func TestNetworkPolicyPreDeleteHook_GetReturns404(t *testing.T) {
	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			return nil, fmt.Errorf("request failed with status code 404")
		},
	}

	err := NetworkPolicyPreDeleteHook(context.Background(), stub, "missing-ns", "my-policy")
	if err != nil {
		t.Fatalf("expected no error for 404, got: %v", err)
	}

	if len(stub.calls) != 1 {
		t.Fatalf("expected 1 CustomAPI call (GET only), got %d", len(stub.calls))
	}
}

func TestNetworkPolicyPreDeleteHook_GetReturns400_SharedNamespace(t *testing.T) {
	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			return nil, fmt.Errorf("Unsuccessful custom API GET on /public/namespaces/shared/active_network_policies, status code 400, body {\"code\":3,\"details\":[],\"message\":\"API is not valid in shared namespace\"}")
		},
	}

	err := NetworkPolicyPreDeleteHook(context.Background(), stub, "shared", "my-policy")
	if err != nil {
		t.Fatalf("expected no error for 400 (shared namespace), got: %v", err)
	}

	if len(stub.calls) != 1 {
		t.Fatalf("expected 1 CustomAPI call (GET only), got %d", len(stub.calls))
	}
	if stub.calls[0].method != "GET" {
		t.Fatalf("expected GET call, got %s", stub.calls[0].method)
	}
}

func TestNetworkPolicyPreDeleteHook_PostReturnsError(t *testing.T) {
	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			switch method {
			case "GET":
				return &ves_io_schema_namespace.GetActiveNetworkPoliciesResponse{
					NetworkPolicies: []*views.ObjectRefType{
						newNetworkObjectRef("acme", "default", "my-policy"),
					},
				}, nil
			case "POST":
				return nil, fmt.Errorf("status code 503: service unavailable")
			default:
				return nil, fmt.Errorf("unexpected method %s", method)
			}
		},
	}

	err := NetworkPolicyPreDeleteHook(context.Background(), stub, "default", "my-policy")
	if err == nil {
		t.Fatal("expected an error from the POST call, got nil")
	}
	if !strings.Contains(err.Error(), "failed to update active network policies") {
		t.Errorf("error message = %q, want it to contain 'failed to update active network policies'", err.Error())
	}
}

func TestNetworkPolicyPreDeleteHook_CorrectURIsAndRPCFQNs(t *testing.T) {
	const (
		ns         = "my-namespace"
		policyName = "my-policy"
	)

	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			switch method {
			case "GET":
				return &ves_io_schema_namespace.GetActiveNetworkPoliciesResponse{
					NetworkPolicies: []*views.ObjectRefType{
						newNetworkObjectRef("acme", ns, policyName),
					},
				}, nil
			case "POST":
				return nil, nil
			default:
				return nil, fmt.Errorf("unexpected method %s", method)
			}
		},
	}

	err := NetworkPolicyPreDeleteHook(context.Background(), stub, ns, policyName)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(stub.calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(stub.calls))
	}

	getCall := stub.calls[0]
	wantGetURI := fmt.Sprintf("/public/namespaces/%s/active_network_policies", ns)
	wantGetRPC := "ves.io.schema.namespace.NamespaceCustomAPI.GetActiveNetworkPolicies"
	if getCall.uri != wantGetURI {
		t.Errorf("GET uri = %q, want %q", getCall.uri, wantGetURI)
	}
	if getCall.rpcFQN != wantGetRPC {
		t.Errorf("GET rpcFQN = %q, want %q", getCall.rpcFQN, wantGetRPC)
	}

	postCall := stub.calls[1]
	wantPostURI := fmt.Sprintf("/public/namespaces/%s/active_network_policies", ns)
	wantPostRPC := "ves.io.schema.namespace.NamespaceCustomAPI.SetActiveNetworkPolicies"
	if postCall.uri != wantPostURI {
		t.Errorf("POST uri = %q, want %q", postCall.uri, wantPostURI)
	}
	if postCall.rpcFQN != wantPostRPC {
		t.Errorf("POST rpcFQN = %q, want %q", postCall.rpcFQN, wantPostRPC)
	}
}

// Ensure the test file compiles even without the vesapi import being used directly.
var _ vesapi.CallOpt
