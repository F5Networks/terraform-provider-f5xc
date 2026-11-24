//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"gopkg.volterra.us/stdlib/client/vesapi"

	ves_io_schema_namespace "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/namespace"
	"gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/views"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newObjectRef is a convenience constructor for *views.ObjectRefType.
func newObjectRef(tenant, namespace, name string) *views.ObjectRefType {
	return &views.ObjectRefType{
		Tenant:    tenant,
		Namespace: namespace,
		Name:      name,
	}
}

// setReqBody mirrors the private struct used inside ServicePolicyPreDeleteHook
// so we can unmarshal and inspect the POST body in tests.
type setReqBody struct {
	Namespace       string      `json:"namespace"`
	ServicePolicies []policyRef `json:"service_policies"`
}

type policyRef struct {
	Tenant    string `json:"tenant,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
}

type blockingServicePolicyClient struct {
	mu sync.Mutex

	getCalls         int
	firstGetStarted  chan struct{}
	releaseFirstGet  chan struct{}
	secondGetEntered chan struct{}

	firstStartOnce  sync.Once
	secondEnterOnce sync.Once
}

func (c *blockingServicePolicyClient) CustomAPI(ctx context.Context, method, uri, rpcFQN, reqYml string, opts ...vesapi.CallOpt) (proto.Message, error) {
	if method != "GET" {
		return nil, fmt.Errorf("unexpected method %s", method)
	}

	c.mu.Lock()
	c.getCalls++
	callNum := c.getCalls
	c.mu.Unlock()

	switch callNum {
	case 1:
		c.firstStartOnce.Do(func() { close(c.firstGetStarted) })
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-c.releaseFirstGet:
		}
	case 2:
		c.secondEnterOnce.Do(func() { close(c.secondGetEntered) })
	}

	return &ves_io_schema_namespace.GetActiveServicePoliciesResponse{
		ServicePolicies: []*views.ObjectRefType{},
	}, nil
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func setupActiveNamespace(t *testing.T, ns string) {
	t.Helper()
	MarkNamespaceActivePolicies(ns, true)
	t.Cleanup(func() {
		MarkNamespaceActivePolicies(ns, false)
	})
}

// TestServicePolicyPreDeleteHook_AlwaysChecksActiveList verifies that the hook
// always performs the active-list check regardless of MarkNamespaceActivePolicies
// state, since that in-memory flag doesn't survive across separate terraform
// plan/apply process invocations and can't be relied on for correctness.
func TestServicePolicyPreDeleteHook_AlwaysChecksActiveList(t *testing.T) {
	MarkNamespaceActivePolicies("default", false)

	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			if method == "GET" {
				return &ves_io_schema_namespace.GetActiveServicePoliciesResponse{
					ServicePolicies: []*views.ObjectRefType{},
				}, nil
			}
			return nil, fmt.Errorf("unexpected CustomAPI call: %s %s", method, rpcFQN)
		},
	}

	err := ServicePolicyPreDeleteHook(context.Background(), stub, "default", "my-policy")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(stub.calls) != 1 {
		t.Fatalf("expected 1 CustomAPI call (GET), got %d", len(stub.calls))
	}
}

// TestServicePolicyPreDeleteHook_PolicyNotInActiveList verifies that when the
// policy being deleted is NOT in the active list, no SET call is made.
func TestServicePolicyPreDeleteHook_PolicyNotInActiveList(t *testing.T) {
	setupActiveNamespace(t, "default")
	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			if method == "GET" {
				return &ves_io_schema_namespace.GetActiveServicePoliciesResponse{
					ServicePolicies: []*views.ObjectRefType{
						newObjectRef("acme", "default", "other-policy"),
					},
				}, nil
			}
			return nil, fmt.Errorf("unexpected CustomAPI call: %s %s", method, rpcFQN)
		},
	}

	err := ServicePolicyPreDeleteHook(context.Background(), stub, "default", "my-policy")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Only the GET call should have been made – no POST.
	if len(stub.calls) != 1 {
		t.Fatalf("expected 1 CustomAPI call (GET), got %d", len(stub.calls))
	}
	if stub.calls[0].method != "GET" {
		t.Fatalf("expected GET call, got %s", stub.calls[0].method)
	}
}

// TestServicePolicyPreDeleteHook_PolicyInActiveList_SinglePolicy verifies that
// when the policy is the only active policy, the SET call is made with an empty list.
func TestServicePolicyPreDeleteHook_PolicyInActiveList_SinglePolicy(t *testing.T) {
	const (
		ns         = "default"
		policyName = "my-policy"
	)
	setupActiveNamespace(t, ns)

	var capturedBody string

	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			switch method {
			case "GET":
				return &ves_io_schema_namespace.GetActiveServicePoliciesResponse{
					ServicePolicies: []*views.ObjectRefType{
						newObjectRef("acme", ns, policyName),
					},
				}, nil
			case "POST":
				capturedBody = reqYml
				return nil, nil
			default:
				return nil, fmt.Errorf("unexpected method %s", method)
			}
		},
	}

	err := ServicePolicyPreDeleteHook(context.Background(), stub, ns, policyName)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Expect exactly 2 calls: GET then POST.
	if len(stub.calls) != 2 {
		t.Fatalf("expected 2 CustomAPI calls (GET + POST), got %d", len(stub.calls))
	}
	if stub.calls[0].method != "GET" {
		t.Fatalf("first call should be GET, got %s", stub.calls[0].method)
	}
	if stub.calls[1].method != "POST" {
		t.Fatalf("second call should be POST, got %s", stub.calls[1].method)
	}

	// Verify the POST body contains an empty service_policies list.
	var body setReqBody
	if err := json.Unmarshal([]byte(capturedBody), &body); err != nil {
		t.Fatalf("failed to unmarshal POST body: %v", err)
	}
	if body.Namespace != ns {
		t.Errorf("POST body namespace = %q, want %q", body.Namespace, ns)
	}
	if len(body.ServicePolicies) != 0 {
		t.Errorf("POST body service_policies = %v, want empty list", body.ServicePolicies)
	}
}

// TestServicePolicyPreDeleteHook_PolicyInActiveList_MultiplePolicies verifies
// that only the target policy is removed and the remaining ones are preserved.
func TestServicePolicyPreDeleteHook_PolicyInActiveList_MultiplePolicies(t *testing.T) {
	const (
		ns         = "default"
		policyName = "policy-b"
	)
	setupActiveNamespace(t, ns)

	remaining := []*views.ObjectRefType{
		newObjectRef("acme", ns, "keep-policy-1"),
		newObjectRef("acme", ns, "keep-policy-2"),
	}

	var capturedBody string

	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			switch method {
			case "GET":
				allPolicies := append(remaining, newObjectRef("acme", ns, policyName))
				return &ves_io_schema_namespace.GetActiveServicePoliciesResponse{
					ServicePolicies: allPolicies,
				}, nil
			case "POST":
				capturedBody = reqYml
				return nil, nil
			default:
				return nil, fmt.Errorf("unexpected method %s", method)
			}
		},
	}

	err := ServicePolicyPreDeleteHook(context.Background(), stub, ns, policyName)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(stub.calls) != 2 {
		t.Fatalf("expected 2 CustomAPI calls (GET + POST), got %d", len(stub.calls))
	}

	var body setReqBody
	if err := json.Unmarshal([]byte(capturedBody), &body); err != nil {
		t.Fatalf("failed to unmarshal POST body: %v", err)
	}

	if body.Namespace != ns {
		t.Errorf("POST body namespace = %q, want %q", body.Namespace, ns)
	}
	if len(body.ServicePolicies) != 2 {
		t.Fatalf("POST body service_policies len = %d, want 2", len(body.ServicePolicies))
	}

	// Ensure the deleted policy is not present.
	for _, ref := range body.ServicePolicies {
		if ref.Name == policyName {
			t.Errorf("deleted policy %q should not be in the POST body", policyName)
		}
	}

	// Ensure the remaining policies are present.
	names := make(map[string]bool)
	for _, ref := range body.ServicePolicies {
		names[ref.Name] = true
	}
	for _, want := range remaining {
		if !names[want.Name] {
			t.Errorf("expected policy %q to be in POST body, but it was missing", want.Name)
		}
	}
}

// TestServicePolicyPreDeleteHook_EmptyActiveList verifies that when the active
// list is empty, no SET call is made (policy cannot be in an empty list).
func TestServicePolicyPreDeleteHook_EmptyActiveList(t *testing.T) {
	setupActiveNamespace(t, "default")
	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			if method == "GET" {
				return &ves_io_schema_namespace.GetActiveServicePoliciesResponse{
					ServicePolicies: []*views.ObjectRefType{},
				}, nil
			}
			return nil, fmt.Errorf("unexpected CustomAPI call: %s %s", method, rpcFQN)
		},
	}

	err := ServicePolicyPreDeleteHook(context.Background(), stub, "default", "my-policy")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(stub.calls) != 1 {
		t.Fatalf("expected 1 CustomAPI call (GET only), got %d", len(stub.calls))
	}
}

// TestServicePolicyPreDeleteHook_GetReturns404 verifies that a 404 from the GET
// call is treated as a no-op (namespace not found).
func TestServicePolicyPreDeleteHook_GetReturns404(t *testing.T) {
	setupActiveNamespace(t, "missing-ns")
	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			return nil, fmt.Errorf("request failed with status code 404")
		},
	}

	err := ServicePolicyPreDeleteHook(context.Background(), stub, "missing-ns", "my-policy")
	if err != nil {
		t.Fatalf("expected no error for 404, got: %v", err)
	}

	if len(stub.calls) != 1 {
		t.Fatalf("expected 1 CustomAPI call (GET only), got %d", len(stub.calls))
	}
}

func TestServicePolicyPreDeleteHook_GetReturns400_SharedNamespace(t *testing.T) {
	setupActiveNamespace(t, "shared")
	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			return nil, fmt.Errorf("Unsuccessful custom API GET on /public/namespaces/shared/active_service_policies, status code 400, body {\"code\":3,\"details\":[],\"message\":\"API is not valid in shared namespace\"}")
		},
	}

	err := ServicePolicyPreDeleteHook(context.Background(), stub, "shared", "my-policy")
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

// TestServicePolicyPreDeleteHook_GetReturnsNon404Error verifies that a non-404
// error from the GET call is propagated as an error.
func TestServicePolicyPreDeleteHook_GetReturnsNon404Error(t *testing.T) {
	setupActiveNamespace(t, "default")
	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			return nil, fmt.Errorf("internal server error: status code 500")
		},
	}

	err := ServicePolicyPreDeleteHook(context.Background(), stub, "default", "my-policy")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to get active service policies") {
		t.Errorf("error message = %q, want it to contain 'failed to get active service policies'", err.Error())
	}
}

// TestServicePolicyPreDeleteHook_GetReturnsUnexpectedType verifies that an
// unexpected proto.Message type from the GET call returns an error.
func TestServicePolicyPreDeleteHook_GetReturnsUnexpectedType(t *testing.T) {
	setupActiveNamespace(t, "default")
	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			// Return a wrong proto type to trigger the type-assertion failure.
			return &ves_io_schema_namespace.GetActiveServicePoliciesResponse{}, nil
		},
	}

	// Override the handler to return a completely different proto message type.
	stub.handler = func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
		// Use a different namespace proto type that is NOT GetActiveServicePoliciesResponse.
		type fakeMsg struct{ proto.Message }
		return nil, nil // nil proto.Message triggers the type assertion failure
	}

	// Provide a handler that returns a non-nil but wrong type.
	// We simulate this by returning a proto.Message that is not *GetActiveServicePoliciesResponse.
	// The easiest way is to use a different generated type from the same package.
	stub.handler = func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
		// Return a SetActiveServicePoliciesResponse (wrong type) instead.
		return &ves_io_schema_namespace.SetActiveServicePoliciesResponse{}, nil
	}

	err := ServicePolicyPreDeleteHook(context.Background(), stub, "default", "my-policy")
	if err == nil {
		t.Fatal("expected an error for unexpected response type, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected response type") {
		t.Errorf("error message = %q, want it to contain 'unexpected response type'", err.Error())
	}
}

// TestServicePolicyPreDeleteHook_PostReturnsError verifies that an error from
// the SET call is propagated.
func TestServicePolicyPreDeleteHook_PostReturnsError(t *testing.T) {
	setupActiveNamespace(t, "default")
	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			switch method {
			case "GET":
				return &ves_io_schema_namespace.GetActiveServicePoliciesResponse{
					ServicePolicies: []*views.ObjectRefType{
						newObjectRef("acme", "default", "my-policy"),
					},
				}, nil
			case "POST":
				return nil, fmt.Errorf("status code 503: service unavailable")
			default:
				return nil, fmt.Errorf("unexpected method %s", method)
			}
		},
	}

	err := ServicePolicyPreDeleteHook(context.Background(), stub, "default", "my-policy")
	if err == nil {
		t.Fatal("expected an error from the POST call, got nil")
	}
	if !strings.Contains(err.Error(), "failed to update active service policies") {
		t.Errorf("error message = %q, want it to contain 'failed to update active service policies'", err.Error())
	}
}

// TestServicePolicyPreDeleteHook_CorrectURIsAndRPCFQNs verifies that the hook
// calls the correct URIs and RPC FQNs.
func TestServicePolicyPreDeleteHook_CorrectURIsAndRPCFQNs(t *testing.T) {
	const (
		ns         = "custom-ns"
		policyName = "my-policy"
	)
	setupActiveNamespace(t, ns)

	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			switch method {
			case "GET":
				return &ves_io_schema_namespace.GetActiveServicePoliciesResponse{
					ServicePolicies: []*views.ObjectRefType{
						newObjectRef("acme", ns, policyName),
					},
				}, nil
			case "POST":
				return nil, nil
			default:
				return nil, fmt.Errorf("unexpected method %s", method)
			}
		},
	}

	err := ServicePolicyPreDeleteHook(context.Background(), stub, ns, policyName)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(stub.calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(stub.calls))
	}

	getCall := stub.calls[0]
	wantGetURI := fmt.Sprintf("/public/namespaces/%s/active_service_policies", ns)
	wantGetRPC := "ves.io.schema.namespace.NamespaceCustomAPI.GetActiveServicePolicies"
	if getCall.uri != wantGetURI {
		t.Errorf("GET uri = %q, want %q", getCall.uri, wantGetURI)
	}
	if getCall.rpcFQN != wantGetRPC {
		t.Errorf("GET rpcFQN = %q, want %q", getCall.rpcFQN, wantGetRPC)
	}

	postCall := stub.calls[1]
	wantPostURI := fmt.Sprintf("/public/namespaces/%s/active_service_policies", ns)
	wantPostRPC := "ves.io.schema.namespace.NamespaceCustomAPI.SetActiveServicePolicies"
	if postCall.uri != wantPostURI {
		t.Errorf("POST uri = %q, want %q", postCall.uri, wantPostURI)
	}
	if postCall.rpcFQN != wantPostRPC {
		t.Errorf("POST rpcFQN = %q, want %q", postCall.rpcFQN, wantPostRPC)
	}
}

// TestServicePolicyPreDeleteHook_PostBodyPreservesRefFields verifies that the
// tenant, namespace and name fields of remaining policies are correctly
// serialised into the POST body.
func TestServicePolicyPreDeleteHook_PostBodyPreservesRefFields(t *testing.T) {
	const (
		ns         = "custom-ns"
		policyName = "delete-me"
	)
	setupActiveNamespace(t, ns)

	keepRef := newObjectRef("my-tenant", "my-ns", "keep-me")

	var capturedBody string

	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			switch method {
			case "GET":
				return &ves_io_schema_namespace.GetActiveServicePoliciesResponse{
					ServicePolicies: []*views.ObjectRefType{
						keepRef,
						newObjectRef("acme", ns, policyName),
					},
				}, nil
			case "POST":
				capturedBody = reqYml
				return nil, nil
			default:
				return nil, fmt.Errorf("unexpected method %s", method)
			}
		},
	}

	err := ServicePolicyPreDeleteHook(context.Background(), stub, ns, policyName)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var body setReqBody
	if err := json.Unmarshal([]byte(capturedBody), &body); err != nil {
		t.Fatalf("failed to unmarshal POST body: %v", err)
	}

	if len(body.ServicePolicies) != 1 {
		t.Fatalf("expected 1 policy in POST body, got %d", len(body.ServicePolicies))
	}

	got := body.ServicePolicies[0]
	if got.Tenant != keepRef.Tenant {
		t.Errorf("tenant = %q, want %q", got.Tenant, keepRef.Tenant)
	}
	if got.Namespace != keepRef.Namespace {
		t.Errorf("namespace = %q, want %q", got.Namespace, keepRef.Namespace)
	}
	if got.Name != keepRef.Name {
		t.Errorf("name = %q, want %q", got.Name, keepRef.Name)
	}
}

// TestPolicyNames verifies the policyNames helper function.
func TestPolicyNames(t *testing.T) {
	refs := []*views.ObjectRefType{
		newObjectRef("t1", "ns1", "policy-a"),
		newObjectRef("t2", "ns2", "policy-b"),
		newObjectRef("t3", "ns3", "policy-c"),
	}

	names := policyNames(refs)

	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(names))
	}
	expected := []string{"policy-a", "policy-b", "policy-c"}
	for i, want := range expected {
		if names[i] != want {
			t.Errorf("names[%d] = %q, want %q", i, names[i], want)
		}
	}
}

// TestPolicyNames_Empty verifies that policyNames returns an empty slice for nil input.
func TestPolicyNames_Empty(t *testing.T) {
	names := policyNames(nil)
	if len(names) != 0 {
		t.Errorf("expected empty slice, got %v", names)
	}
}

// TestServicePolicyPreDeleteHook_NilActiveListResponse verifies that a nil
// ServicePolicies list in the response is handled gracefully (treated as empty).
func TestServicePolicyPreDeleteHook_NilActiveListResponse(t *testing.T) {
	setupActiveNamespace(t, "default")
	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			if method == "GET" {
				// ServicePolicies is nil (zero value)
				return &ves_io_schema_namespace.GetActiveServicePoliciesResponse{}, nil
			}
			return nil, fmt.Errorf("unexpected CustomAPI call: %s %s", method, rpcFQN)
		},
	}

	err := ServicePolicyPreDeleteHook(context.Background(), stub, "default", "my-policy")
	if err != nil {
		t.Fatalf("expected no error for nil active list, got: %v", err)
	}

	// Only GET should have been called – policy not found, no POST.
	if len(stub.calls) != 1 {
		t.Fatalf("expected 1 CustomAPI call (GET only), got %d", len(stub.calls))
	}
}

// TestServicePolicyPreDeleteHook_GetReqYmlIsEmpty verifies that the GET call
// is made with an empty request body (as required by the API).
func TestServicePolicyPreDeleteHook_GetReqYmlIsEmpty(t *testing.T) {
	setupActiveNamespace(t, "default")
	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			return &ves_io_schema_namespace.GetActiveServicePoliciesResponse{}, nil
		},
	}

	_ = ServicePolicyPreDeleteHook(context.Background(), stub, "default", "my-policy")

	if len(stub.calls) == 0 {
		t.Fatal("expected at least one CustomAPI call")
	}
	if stub.calls[0].reqYml != "" {
		t.Errorf("GET reqYml = %q, want empty string", stub.calls[0].reqYml)
	}
}

// TestServicePolicyPreDeleteHook_CustomAPIClientInterface verifies that
// stubCustomAPIClient satisfies the CustomAPIClient interface (compile-time check).
func TestServicePolicyPreDeleteHook_CustomAPIClientInterface(t *testing.T) {
	var _ CustomAPIClient = (*stubCustomAPIClient)(nil)
}

// TestServicePolicyPreDeleteHook_DuplicatePolicyNames verifies that if the
// same policy name appears multiple times in the active list, all occurrences
// are removed.
func TestServicePolicyPreDeleteHook_DuplicatePolicyNames(t *testing.T) {
	const (
		ns         = "default"
		policyName = "dup-policy"
	)
	setupActiveNamespace(t, ns)

	var capturedBody string

	stub := &stubCustomAPIClient{
		handler: func(method, uri, rpcFQN, reqYml string) (proto.Message, error) {
			switch method {
			case "GET":
				return &ves_io_schema_namespace.GetActiveServicePoliciesResponse{
					ServicePolicies: []*views.ObjectRefType{
						newObjectRef("acme", ns, policyName),
						newObjectRef("acme", ns, policyName), // duplicate
						newObjectRef("acme", ns, "other"),
					},
				}, nil
			case "POST":
				capturedBody = reqYml
				return nil, nil
			default:
				return nil, fmt.Errorf("unexpected method %s", method)
			}
		},
	}

	err := ServicePolicyPreDeleteHook(context.Background(), stub, ns, policyName)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var body setReqBody
	if err := json.Unmarshal([]byte(capturedBody), &body); err != nil {
		t.Fatalf("failed to unmarshal POST body: %v", err)
	}

	// Both duplicates should be removed; only "other" remains.
	if len(body.ServicePolicies) != 1 {
		t.Fatalf("expected 1 policy in POST body, got %d: %v", len(body.ServicePolicies), body.ServicePolicies)
	}
	if body.ServicePolicies[0].Name != "other" {
		t.Errorf("remaining policy = %q, want %q", body.ServicePolicies[0].Name, "other")
	}
}

func TestServicePolicyPreDeleteHook_SerializesConcurrentCallsPerNamespace(t *testing.T) {
	setupActiveNamespace(t, "default")
	stub := &blockingServicePolicyClient{
		firstGetStarted:  make(chan struct{}),
		releaseFirstGet:  make(chan struct{}),
		secondGetEntered: make(chan struct{}),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errCh := make(chan error, 2)

	go func() {
		errCh <- ServicePolicyPreDeleteHook(ctx, stub, "default", "policy-one")
	}()

	select {
	case <-stub.firstGetStarted:
	case <-time.After(1 * time.Second):
		t.Fatal("first hook call did not reach GET")
	}

	go func() {
		errCh <- ServicePolicyPreDeleteHook(ctx, stub, "default", "policy-two")
	}()

	select {
	case <-stub.secondGetEntered:
		t.Fatal("second hook call entered GET before first call released namespace lock")
	case <-time.After(250 * time.Millisecond):
	}

	stub.mu.Lock()
	if stub.getCalls != 1 {
		stub.mu.Unlock()
		t.Fatalf("expected exactly one GET call while first request is blocked, got %d", stub.getCalls)
	}
	stub.mu.Unlock()

	close(stub.releaseFirstGet)

	select {
	case <-stub.secondGetEntered:
	case <-time.After(1 * time.Second):
		t.Fatal("second hook call did not proceed after first request released namespace lock")
	}

	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			t.Fatalf("hook call failed: %v", err)
		}
	}

	stub.mu.Lock()
	defer stub.mu.Unlock()
	if stub.getCalls != 2 {
		t.Fatalf("expected two GET calls total, got %d", stub.getCalls)
	}
}

// Ensure the test file compiles even without the vesapi import being used directly.
var _ vesapi.CallOpt

// TestParseReferringObjects tests parsing of F5XC 409 error strings across various resource types.
func TestParseReferringObjects(t *testing.T) {
	tests := []struct {
		name     string
		errText  string
		expected []ReferringResource
	}{
		{
			name:    "Single referring http_loadbalancer with namespace/name",
			errText: "DeleteResource: ves.io.schema.service_policy.Object tenant/default/policy1 is still being referred by 1 objects - [ves.io.schema.views.http_loadbalancer.Object default/lb1]",
			expected: []ReferringResource{
				{Kind: "http_loadbalancers", Namespace: "default", Name: "lb1"},
			},
		},
		{
			name:    "Multiple referring objects (http_loadbalancer, cdn_loadbalancer, tcp_loadbalancer)",
			errText: "DeleteResource: ves.io.schema.service_policy.Object tenant/default/policy1 is still being referred by 3 objects - [ves.io.schema.views.http_loadbalancer.Object default/lb1 ves.io.schema.views.cdn_loadbalancer.Object prod/cdn1 ves.io.schema.views.tcp_loadbalancer.Object staging/tcp1]",
			expected: []ReferringResource{
				{Kind: "http_loadbalancers", Namespace: "default", Name: "lb1"},
				{Kind: "cdn_loadbalancers", Namespace: "prod", Name: "cdn1"},
				{Kind: "tcp_loadbalancers", Namespace: "staging", Name: "tcp1"},
			},
		},
		{
			name:    "Three part path tenant/namespace/name",
			errText: "is still being referred by 1 objects - [ves.io.schema.views.http_loadbalancer.Object my-tenant/user-ns/my-lb]",
			expected: []ReferringResource{
				{Kind: "http_loadbalancers", Namespace: "user-ns", Name: "my-lb"},
			},
		},
		{
			name:     "No referring objects",
			errText:  "some random error message",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseReferringObjects(tt.errText)
			if len(got) != len(tt.expected) {
				t.Fatalf("expected %d referring objects, got %d", len(tt.expected), len(got))
			}
			for i, want := range tt.expected {
				if got[i].Kind != want.Kind {
					t.Errorf("[%d] kind = %q, want %q", i, got[i].Kind, want.Kind)
				}
				if got[i].Namespace != want.Namespace {
					t.Errorf("[%d] namespace = %q, want %q", i, got[i].Namespace, want.Namespace)
				}
				if got[i].Name != want.Name {
					t.Errorf("[%d] name = %q, want %q", i, got[i].Name, want.Name)
				}
			}
		})
	}
}

// A blank namespace maps to the shared "_default" key in the active-policy registry.
func TestHasActivePolicies(t *testing.T) {
	assert.False(t, HasActivePolicies("cov-unmarked"))

	MarkNamespaceActivePolicies(" ", true)
	t.Cleanup(func() { MarkNamespaceActivePolicies("", false) })
	assert.True(t, HasActivePolicies(""))
	assert.True(t, HasActivePolicies("_default"))
}

// A blank namespace shares one lock per policy kind.
func TestPolicyNamespaceLocks_BlankNamespace(t *testing.T) {
	lockServicePolicyNamespace(" ")()
	lockNetworkPolicyNamespace("")()
	_, ok := servicePolicyNamespaceLocks.Load("_default")
	assert.True(t, ok)
	_, ok = networkPolicyNamespaceLocks.Load("_default")
	assert.True(t, ok)
}
