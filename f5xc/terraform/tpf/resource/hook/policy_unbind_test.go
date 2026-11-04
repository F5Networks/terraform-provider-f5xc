//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/client"
)

// unbindTestAuthenticator is a minimal client.Authenticator that performs no auth and
// hands back a plain *http.Client so policyUnbindReferencingObjects can drive a real
// httptest server in-process.
type unbindTestAuthenticator struct{ httpClient *http.Client }

func (a *unbindTestAuthenticator) ApplyAuth(*http.Request) error { return nil }
func (a *unbindTestAuthenticator) GetClient() *http.Client       { return a.httpClient }
func (a *unbindTestAuthenticator) GetClientData() string         { return "" }

// newUnbindTestAPIClient builds a *client.APIClient whose RestClient points at srvURL.
func newUnbindTestAPIClient(srv *httptest.Server) *client.APIClient {
	auth := &unbindTestAuthenticator{httpClient: srv.Client()}
	return &client.APIClient{RestClient: client.NewAuthClient(srv.URL, auth, "test-agent")}
}

// fpUnbindSpec is the config used by ForwardProxyPolicyUnbindReferencingObjects, reused in
// tests to exercise the shared recursive walker with a realistic spec.
var fpUnbindSpec = policyUnbindSpec{
	logPrefix:       "ForwardProxyPolicy",
	containerKey:    "active_forward_proxy_policies",
	listKey:         "forward_proxy_policies",
	emptyChoiceKeys: []string{"no_forward_proxy_policy"},
	isReferrerKind:  isForwardProxyPolicyReferrerKind,
}

// policyRef builds an ObjectRefType-shaped map for a policy reference.
func policyObjRef(namespace, name string) map[string]any {
	ref := map[string]any{"name": name}
	if namespace != "" {
		ref["namespace"] = namespace
	}
	return ref
}

// policyNamesInList extracts referenced policy names from a []any of ObjectRefType-shaped
// maps for assertions.
func policyNamesInList(t *testing.T, raw any) []string {
	t.Helper()
	list, ok := raw.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", raw)
	}
	var names []string
	for _, item := range list {
		ref, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any list entry, got %T", item)
		}
		names = append(names, ref["name"].(string))
	}
	return names
}

func TestFilterPolicyRefList_RemovesMatchByName(t *testing.T) {
	list := []any{policyObjRef("", "p1"), policyObjRef("", "p2")}

	filtered, removed := filterPolicyRefList(list, "p1", "default")
	if !removed {
		t.Fatalf("expected removed=true")
	}
	got := policyNamesInList(t, filtered)
	if len(got) != 1 || got[0] != "p2" {
		t.Fatalf("expected only p2 to remain, got %v", got)
	}
}

func TestFilterPolicyRefList_NoMatch(t *testing.T) {
	list := []any{policyObjRef("", "pA"), policyObjRef("", "pB")}

	filtered, removed := filterPolicyRefList(list, "p1", "default")
	if removed {
		t.Fatalf("expected removed=false when policy is absent")
	}
	if len(filtered) != 2 {
		t.Fatalf("expected list unchanged (2 entries), got %d", len(filtered))
	}
}

func TestFilterPolicyRefList_NamespaceMismatchKeepsEntry(t *testing.T) {
	list := []any{policyObjRef("other-ns", "p1")}

	filtered, removed := filterPolicyRefList(list, "p1", "default")
	if removed {
		t.Fatalf("expected removed=false: same name but different namespace must not match")
	}
	if len(filtered) != 1 {
		t.Fatalf("expected entry preserved, got %d entries", len(filtered))
	}
}

func TestFilterPolicyRefList_NamespaceOmittedMatches(t *testing.T) {
	list := []any{policyObjRef("", "p1")}

	filtered, removed := filterPolicyRefList(list, "p1", "default")
	if !removed {
		t.Fatalf("expected removed=true when reference omits namespace")
	}
	if len(filtered) != 0 {
		t.Fatalf("expected empty list, got %d entries", len(filtered))
	}
}

// TestRemovePolicyRefFromNode_PrunesOne verifies that removing one of several policies
// leaves the container in place with the remaining refs.
func TestRemovePolicyRefFromNode_PrunesOne(t *testing.T) {
	spec := map[string]any{
		"active_forward_proxy_policies": map[string]any{
			"forward_proxy_policies": []any{
				policyObjRef("default", "p1"),
				policyObjRef("default", "p2"),
			},
		},
	}

	changed := removePolicyRefFromNode(context.Background(), spec, fpUnbindSpec, "http_loadbalancers", "default", "lb1", "p1", "default")
	if !changed {
		t.Fatalf("expected changed=true")
	}
	container := spec["active_forward_proxy_policies"].(map[string]any)
	got := policyNamesInList(t, container["forward_proxy_policies"])
	if len(got) != 1 || got[0] != "p2" {
		t.Fatalf("expected only p2 to remain, got %v", got)
	}
	if _, ok := spec["no_forward_proxy_policy"]; ok {
		t.Fatalf("did not expect no_forward_proxy_policy while list is non-empty")
	}
}

// TestRemovePolicyRefFromNode_EmptiesSwapsToNoChoice verifies that when the last policy is
// removed the container is deleted and the empty "no_*" oneof variant is set.
func TestRemovePolicyRefFromNode_EmptiesSwapsToNoChoice(t *testing.T) {
	spec := map[string]any{
		"active_forward_proxy_policies": map[string]any{
			"forward_proxy_policies": []any{
				policyObjRef("default", "p1"),
			},
		},
	}

	changed := removePolicyRefFromNode(context.Background(), spec, fpUnbindSpec, "origin_pools", "default", "op1", "p1", "default")
	if !changed {
		t.Fatalf("expected changed=true")
	}
	if _, ok := spec["active_forward_proxy_policies"]; ok {
		t.Fatalf("expected active_forward_proxy_policies to be removed when list empties")
	}
	empty, ok := spec["no_forward_proxy_policy"].(map[string]any)
	if !ok {
		t.Fatalf("expected no_forward_proxy_policy empty choice to be set, got %T", spec["no_forward_proxy_policy"])
	}
	if len(empty) != 0 {
		t.Fatalf("expected no_forward_proxy_policy to be an empty object, got %v", empty)
	}
}

// TestRemovePolicyRefFromNode_NestedContainer confirms the recursive walk finds a container
// nested under an intermediate key (as happens across the various parent kinds).
func TestRemovePolicyRefFromNode_NestedContainer(t *testing.T) {
	spec := map[string]any{
		"some_wrapper": map[string]any{
			"active_network_policies": map[string]any{
				"network_policies": []any{
					policyObjRef("default", "np1"),
					policyObjRef("default", "np2"),
				},
			},
		},
	}
	cfg := policyUnbindSpec{
		logPrefix:       "NetworkPolicy",
		containerKey:    "active_network_policies",
		listKey:         "network_policies",
		emptyChoiceKeys: []string{"no_network_policy"},
		isReferrerKind:  isNetworkPolicyReferrerKind,
	}

	changed := removePolicyRefFromNode(context.Background(), spec, cfg, "virtual_networks", "default", "vn1", "np2", "default")
	if !changed {
		t.Fatalf("expected changed=true for nested container")
	}
	wrapper := spec["some_wrapper"].(map[string]any)
	container := wrapper["active_network_policies"].(map[string]any)
	got := policyNamesInList(t, container["network_policies"])
	if len(got) != 1 || got[0] != "np1" {
		t.Fatalf("expected only np1 to remain, got %v", got)
	}
}

// TestRemovePolicyRefFromNode_MultipleEmptyChoiceKeys verifies both sibling empty choices
// are set for enhanced firewall policy (sites use no_firewall_policy).
func TestRemovePolicyRefFromNode_MultipleEmptyChoiceKeys(t *testing.T) {
	spec := map[string]any{
		"active_enhanced_firewall_policies": map[string]any{
			"enhanced_firewall_policies": []any{
				policyObjRef("default", "efp1"),
			},
		},
	}
	cfg := policyUnbindSpec{
		logPrefix:       "EnhancedFirewallPolicy",
		containerKey:    "active_enhanced_firewall_policies",
		listKey:         "enhanced_firewall_policies",
		emptyChoiceKeys: []string{"no_firewall_policy", "no_enhanced_firewall_policy"},
		isReferrerKind:  isEnhancedFirewallPolicyReferrerKind,
	}

	changed := removePolicyRefFromNode(context.Background(), spec, cfg, "aws_vpc_sites", "default", "site1", "efp1", "default")
	if !changed {
		t.Fatalf("expected changed=true")
	}
	if _, ok := spec["active_enhanced_firewall_policies"]; ok {
		t.Fatalf("expected container removed when list empties")
	}
	for _, key := range cfg.emptyChoiceKeys {
		empty, ok := spec[key].(map[string]any)
		if !ok || len(empty) != 0 {
			t.Fatalf("expected %q empty choice to be set to {}, got %v", key, spec[key])
		}
	}
}

// TestRemovePolicyRefFromNode_RoundTripJSON exercises a spec unmarshalled from JSON (as
// fetched from the API), mutates it, and re-marshals it.
func TestRemovePolicyRefFromNode_RoundTripJSON(t *testing.T) {
	raw := `{
      "spec": {
        "active_forward_proxy_policies": {
          "forward_proxy_policies": [
            {"name": "p1", "namespace": "default"},
            {"name": "p2", "namespace": "default"}
          ]
        }
      }
    }`

	var parent map[string]any
	if err := json.Unmarshal([]byte(raw), &parent); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	spec := parent["spec"].(map[string]any)

	changed := removePolicyRefFromNode(context.Background(), spec, fpUnbindSpec, "http_loadbalancers", "default", "lb1", "p1", "default")
	if !changed {
		t.Fatalf("expected changed=true")
	}
	if _, err := json.Marshal(parent); err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	container := spec["active_forward_proxy_policies"].(map[string]any)
	got := policyNamesInList(t, container["forward_proxy_policies"])
	if len(got) != 1 || got[0] != "p2" {
		t.Fatalf("expected only p2 to remain, got %v", got)
	}
}

// TestIsPolicyReferrerKind_Filtering locks in the referrer filtering for each policy kind,

// TestPolicyUnbind_FetchErrorSkipsReferrerButContinues is the regression guard for the
// XC-3073 hardening: when one referrer in a multi-referrer 409 cannot be fetched (e.g. it
// was already deleted earlier in the same apply), the unbind must skip only that referrer
// and still unbind the remaining ones, rather than aborting the whole loop with an error.
func TestPolicyUnbind_FetchErrorSkipsReferrerButContinues(t *testing.T) {
	var mu sync.Mutex
	var putBodies = map[string][]byte{}

	specWithPolicy := func() map[string]any {
		return map[string]any{
			"spec": map[string]any{
				"active_forward_proxy_policies": map[string]any{
					"forward_proxy_policies": []any{
						policyObjRef("default", "fpp1"),
					},
				},
			},
		}
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The first referrer ("op-missing") is gone: every GET 404s. Because the hook
		// tries both /config and /public URLs, 404 both so fetchErr is non-nil.
		if strings.Contains(r.URL.Path, "op-missing") {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
			return
		}
		// The second referrer ("op-ok") is reachable: GET returns its spec, PUT is captured.
		switch r.Method {
		case http.MethodGet:
			body, _ := json.Marshal(specWithPolicy())
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		case http.MethodPut:
			b := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(b)
			mu.Lock()
			putBodies[r.URL.Path] = b
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer srv.Close()

	apiClient := newUnbindTestAPIClient(srv)

	// 409 lists two origin_pool referrers; the first is unreachable, the second is fine.
	errText := "DeleteResource: ves.io.schema.forward_proxy_policy.Object tenant/default/fpp1 " +
		"is still being referred by 2 objects - [" +
		"ves.io.schema.views.origin_pool.Object default/op-missing " +
		"ves.io.schema.views.origin_pool.Object default/op-ok]"

	err := ForwardProxyPolicyUnbindReferencingObjects(context.Background(), apiClient, "default", "fpp1", errText)
	if err != nil {
		t.Fatalf("expected unbind to succeed despite one unreachable referrer, got: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// The reachable referrer must have been PUT back with the policy pruned.
	var putSeen bool
	for path, body := range putBodies {
		if !strings.Contains(path, "op-ok") {
			t.Fatalf("unexpected PUT to %q (only op-ok should be updated)", path)
		}
		putSeen = true
		var parent map[string]any
		if uerr := json.Unmarshal(body, &parent); uerr != nil {
			t.Fatalf("failed to unmarshal PUT body: %v", uerr)
		}
		spec := parent["spec"].(map[string]any)
		if _, ok := spec["active_forward_proxy_policies"]; ok {
			t.Fatalf("expected active_forward_proxy_policies to be removed when list empties")
		}
		if _, ok := spec["no_forward_proxy_policy"].(map[string]any); !ok {
			t.Fatalf("expected no_forward_proxy_policy empty choice to be set on op-ok")
		}
	}
	if !putSeen {
		t.Fatalf("expected the reachable referrer op-ok to be updated via PUT, but no PUT was captured")
	}
}

// including that the policy's own plural type (echoed by the generic 409 parser) is ignored.
func TestIsPolicyReferrerKind_Filtering(t *testing.T) {
	if !isForwardProxyPolicyReferrerKind("http_loadbalancers") {
		t.Fatalf("expected http_loadbalancers to be a forward proxy policy referrer")
	}
	if isForwardProxyPolicyReferrerKind("forward_proxy_policys") {
		t.Fatalf("expected the policy's own type to be filtered out")
	}
	if !isNetworkPolicyReferrerKind("virtual_networks") {
		t.Fatalf("expected virtual_networks to be a network policy referrer")
	}
	if isNetworkPolicyReferrerKind("network_policys") {
		t.Fatalf("expected network_policys to be filtered out")
	}
	if !isEnhancedFirewallPolicyReferrerKind("aws_vpc_sites") {
		t.Fatalf("expected aws_vpc_sites to be an enhanced firewall policy referrer")
	}
	if isEnhancedFirewallPolicyReferrerKind("enhanced_firewall_policys") {
		t.Fatalf("expected enhanced_firewall_policys to be filtered out")
	}
}
