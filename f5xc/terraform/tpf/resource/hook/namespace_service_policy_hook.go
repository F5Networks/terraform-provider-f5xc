//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

// Package hook contains custom lifecycle hooks for F5XC Terraform resources.
// These hooks are called at specific points in the resource lifecycle (e.g. pre-delete)
// and are NOT auto-generated – they are maintained manually.
package hook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gopkg.volterra.us/stdlib/client/vesapi"

	ves_io_schema_namespace "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/namespace"
	"gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/views"
	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/client"
)

// CustomAPIClient is the minimal interface required by the hooks to call F5XC Custom APIs.
// It is satisfied by *client.APIClient.
type CustomAPIClient interface {
	CustomAPI(ctx context.Context, method, uri, rpcFQN, reqYml string, opts ...vesapi.CallOpt) (proto.Message, error)
}

var (
	servicePolicyNamespaceLocks sync.Map
	activePolicyNamespaces      sync.Map
)

// MarkNamespaceActivePolicies enables or disables active service policy checking for a namespace.
// When active is true, ServicePolicyPreDeleteHook will query and clean up active service policies
// for the given namespace before deleting a policy.
func MarkNamespaceActivePolicies(namespace string, active bool) {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = "_default"
	}
	if active {
		activePolicyNamespaces.Store(ns, true)
	} else {
		activePolicyNamespaces.Delete(ns)
	}
}

// HasActivePolicies reports whether active service policies or service_policies_from_namespace
// are marked in use for a namespace.
func HasActivePolicies(namespace string) bool {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = "_default"
	}
	v, ok := activePolicyNamespaces.Load(ns)
	return ok && v.(bool)
}

// lockServicePolicyNamespace serializes active list mutation and delete operations
// for service policies within a namespace.
func lockServicePolicyNamespace(namespace string) func() {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = "_default"
	}

	muAny, _ := servicePolicyNamespaceLocks.LoadOrStore(ns, &sync.Mutex{})
	mu := muAny.(*sync.Mutex)
	mu.Lock()

	return mu.Unlock
}

// ServicePolicyPreDeleteHook is a pre-delete hook for the f5xc_service_policy resource.
//
// Before a service policy can be deleted it must be removed from the list of active
// service policies in its namespace. If active service policies / service_policies_from_namespace
// are not in use for the namespace, or if the policy is not in the active list, the hook is a no-op.
//
// Flow:
//  1. Check HasActivePolicies(namespace) -> if false, return nil immediately.
//  2. GET /public/namespaces/<ns>/active_service_policies -> list current active policies
//  3. Check whether <policyName> is in the list
//  4. If yes: POST /public/namespaces/<ns>/active_service_policies with the filtered list
//  5. Return - the caller can then proceed with the actual DeleteObject call
func ServicePolicyPreDeleteHook(ctx context.Context, apiClient CustomAPIClient, namespace, policyName string) error {
	if !HasActivePolicies(namespace) {
		tflog.Info(ctx, fmt.Sprintf("[hook] ServicePolicyPreDeleteHook: service_policies_from_namespace / active policies not enabled for namespace %q, skipping active policy check", namespace))
		return nil
	}

	unlock := lockServicePolicyNamespace(namespace)
	defer unlock()

	return servicePolicyPreDeleteHookUnlocked(ctx, apiClient, namespace, policyName)
}

// servicePolicyPreDeleteHookUnlocked performs active list cleanup without taking
// a namespace lock. Callers that require serialization must lock externally.
func servicePolicyPreDeleteHookUnlocked(ctx context.Context, apiClient CustomAPIClient, namespace, policyName string) error {
	tflog.Info(ctx, fmt.Sprintf("[hook] ServicePolicyPreDeleteHook: checking if %q is active in namespace %q", policyName, namespace))

	// -------------------------------------------------------------------------
	// Step 1: Get current active service policies
	// -------------------------------------------------------------------------
	getRPCFQN := "ves.io.schema.namespace.NamespaceCustomAPI.GetActiveServicePolicies"
	getURI := fmt.Sprintf("/public/namespaces/%s/active_service_policies", namespace)

	getResp, err := apiClient.CustomAPI(ctx, "GET", getURI, getRPCFQN, "")
	if err != nil {
		if strings.Contains(err.Error(), "status code 404") {
			// Namespace not found – nothing to clean up
			tflog.Info(ctx, fmt.Sprintf("[hook] namespace %q not found (404), skipping active policy cleanup", namespace))
			return nil
		}
		if strings.Contains(err.Error(), "status code 400") {
			tflog.Info(ctx, fmt.Sprintf("[hook] namespace %q does not support active service policies (400), skipping cleanup", namespace))
			return nil
		}
		return fmt.Errorf("ServicePolicyPreDeleteHook: failed to get active service policies: %w", err)
	}

	// Cast the proto.Message to the concrete response type
	activePoliciesResp, ok := getResp.(*ves_io_schema_namespace.GetActiveServicePoliciesResponse)
	if !ok {
		return fmt.Errorf("ServicePolicyPreDeleteHook: unexpected response type %T, expected *GetActiveServicePoliciesResponse", getResp)
	}

	// -------------------------------------------------------------------------
	// Step 2: Check if the policy is in the active list
	// -------------------------------------------------------------------------
	currentPolicies := activePoliciesResp.GetServicePolicies()
	tflog.Info(ctx, fmt.Sprintf("[hook] active service policies in namespace %q: %v", namespace, policyNames(currentPolicies)))

	filteredPolicies := make([]*views.ObjectRefType, 0, len(currentPolicies))
	found := false
	for _, ref := range currentPolicies {
		if ref.GetName() == policyName {
			found = true
			tflog.Info(ctx, fmt.Sprintf("[hook] policy %q found in active list – will remove before delete", policyName))
			continue // skip – this is the one being deleted
		}
		filteredPolicies = append(filteredPolicies, ref)
	}

	if !found {
		tflog.Info(ctx, fmt.Sprintf("[hook] policy %q is NOT in the active list – no cleanup needed", policyName))
		return nil
	}

	// -------------------------------------------------------------------------
	// Step 3: Update active service policies (remove the deleted policy)
	// -------------------------------------------------------------------------
	setRPCFQN := "ves.io.schema.namespace.NamespaceCustomAPI.SetActiveServicePolicies"
	setURI := fmt.Sprintf("/public/namespaces/%s/active_service_policies", namespace)

	// Build the JSON body for SetActiveServicePolicies.
	// Shape: { "namespace": "<ns>", "service_policies": [ { "tenant": "...", "namespace": "...", "name": "..." }, ... ] }
	type policyRef struct {
		Tenant    string `json:"tenant,omitempty"`
		Namespace string `json:"namespace,omitempty"`
		Name      string `json:"name"`
	}
	type setReqBody struct {
		Namespace       string      `json:"namespace"`
		ServicePolicies []policyRef `json:"service_policies"`
	}

	refs := make([]policyRef, 0, len(filteredPolicies))
	for _, ref := range filteredPolicies {
		refs = append(refs, policyRef{
			Tenant:    ref.GetTenant(),
			Namespace: ref.GetNamespace(),
			Name:      ref.GetName(),
		})
	}

	bodyBytes, err := json.Marshal(setReqBody{
		Namespace:       namespace,
		ServicePolicies: refs,
	})
	if err != nil {
		return fmt.Errorf("ServicePolicyPreDeleteHook: failed to marshal SetActiveServicePolicies request: %w", err)
	}

	tflog.Info(ctx, fmt.Sprintf("[hook] updating active service policies for namespace %q: %s", namespace, string(bodyBytes)))

	_, err = apiClient.CustomAPI(ctx, "POST", setURI, setRPCFQN, string(bodyBytes))
	if err != nil {
		return fmt.Errorf("ServicePolicyPreDeleteHook: failed to update active service policies: %w", err)
	}

	tflog.Info(ctx, fmt.Sprintf("[hook] successfully removed %q from active service policies in namespace %q", policyName, namespace))
	return nil
}

// policyNames is a helper that extracts the names from a slice of ObjectRefType for logging.
func policyNames(refs []*views.ObjectRefType) []string {
	names := make([]string, 0, len(refs))
	for _, ref := range refs {
		names = append(names, ref.GetName())
	}
	return names
}

// ReferringResource represents a resource referencing a service policy.
type ReferringResource struct {
	Kind      string // REST endpoint plural e.g. "http_loadbalancers", "cdn_loadbalancers", "tcp_loadbalancers"
	Namespace string
	Name      string
}

// ReferringHTTPLoadBalancer is an alias for ReferringResource for backward compatibility.
type ReferringHTTPLoadBalancer = ReferringResource

// ParseReferringObjects extracts referring resources (e.g. http_loadbalancer, cdn_loadbalancer, tcp_loadbalancer)
// from an F5XC API 409 Conflict error string.
//
// Example error string:
// "DeleteResource: ves.io.schema.service_policy.Object tenant/default/policy1 is still being referred by 2 objects - [ves.io.schema.views.http_loadbalancer.Object default/lb1 ves.io.schema.views.cdn_loadbalancer.Object default/cdn1]"
func ParseReferringObjects(errText string) []ReferringResource {
	var result []ReferringResource

	// Ignore target object in prefix (e.g. "DeleteResource: ves.io.schema.service_policy.Object...")
	textToParse := errText
	if idx := strings.Index(errText, "is still being referred by"); idx != -1 {
		textToParse = errText[idx:]
	}

	re := regexp.MustCompile(`ves\.io\.schema\.([a-zA-Z0-9_\.]+)\.Object\s+([^\s\]]+)`)
	matches := re.FindAllStringSubmatch(textToParse, -1)
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		rawKind := match[1] // e.g. "views.http_loadbalancer", "views.cdn_loadbalancer", "views.tcp_loadbalancer"
		path := match[2]    // e.g. "default/lb1" or "tenant/default/lb1"

		parts := strings.Split(path, "/")
		if len(parts) < 2 {
			continue
		}
		name := parts[len(parts)-1]
		ns := parts[len(parts)-2]

		kindParts := strings.Split(rawKind, ".")
		shortKind := kindParts[len(kindParts)-1] // e.g. "http_loadbalancer", "cdn_loadbalancer"

		// Map to plural REST API endpoint segment
		pluralKind := shortKind + "s"

		result = append(result, ReferringResource{
			Kind:      pluralKind,
			Namespace: ns,
			Name:      name,
		})
	}
	return result
}

// ParseReferringHTTPLoadBalancers is an alias for ParseReferringObjects for backward compatibility.
func ParseReferringHTTPLoadBalancers(errText string) []ReferringHTTPLoadBalancer {
	return ParseReferringObjects(errText)
}

// ServicePolicyUnbindReferencingObjects parses an F5XC 409 conflict error, finds any referring resources
// (http_loadbalancer, cdn_loadbalancer, tcp_loadbalancer, etc.), and removes policyName from their
// active_service_policies specification on F5XC API.
func ServicePolicyUnbindReferencingObjects(
	ctx context.Context,
	apiClient *client.APIClient,
	policyNamespace, policyName string,
	errText string,
) error {
	if apiClient == nil || apiClient.RestClient == nil {
		return fmt.Errorf("ServicePolicyUnbindReferencingObjects: apiClient or RestClient is nil")
	}

	referringObjs := ParseReferringObjects(errText)
	if len(referringObjs) == 0 {
		tflog.Info(ctx, fmt.Sprintf("[hook] no referring objects found in error text: %s", errText))
		return nil
	}

	baseURL := strings.TrimRight(apiClient.RestClient.URL, "/")

	for _, obj := range referringObjs {
		ns := obj.Namespace
		if ns == "" {
			ns = policyNamespace
		}
		tflog.Info(ctx, fmt.Sprintf("[hook] unbinding policy %q from %s %q in namespace %q", policyName, obj.Kind, obj.Name, ns))

		urls := []string{
			fmt.Sprintf("%s/config/namespaces/%s/%s/%s", baseURL, ns, obj.Kind, obj.Name),
			fmt.Sprintf("%s/public/namespaces/%s/%s/%s", baseURL, ns, obj.Kind, obj.Name),
		}

		var respBytes []byte
		var targetURL string
		var fetchErr error

		for _, u := range urls {
			respBytes, fetchErr = apiClient.RestClient.Execute(ctx, http.MethodGet, u, nil)
			if fetchErr == nil {
				targetURL = u
				break
			}
		}

		if fetchErr != nil {
			return fmt.Errorf("failed to fetch %s %s/%s: %w", obj.Kind, ns, obj.Name, fetchErr)
		}

		var parentObj map[string]any
		if err := json.Unmarshal(respBytes, &parentObj); err != nil {
			return fmt.Errorf("failed to parse %s %s/%s JSON: %w", obj.Kind, ns, obj.Name, err)
		}

		spec, ok := parentObj["spec"].(map[string]any)
		if !ok || spec == nil {
			continue
		}

		activePolicies, ok := spec["active_service_policies"].(map[string]any)
		if !ok || activePolicies == nil {
			continue
		}

		rawPolicies, ok := activePolicies["policies"].([]any)
		if !ok || rawPolicies == nil {
			continue
		}

		var filteredPolicies []any
		for _, rawP := range rawPolicies {
			pMap, ok := rawP.(map[string]any)
			if !ok {
				continue
			}
			if pName, ok := pMap["name"].(string); ok && pName == policyName {
				tflog.Info(ctx, fmt.Sprintf("[hook] removing policy %q from active_service_policies on %s %s/%s", policyName, obj.Kind, ns, obj.Name))
				continue
			}
			filteredPolicies = append(filteredPolicies, rawP)
		}

		if len(filteredPolicies) == 0 {
			delete(spec, "active_service_policies")
			spec["no_service_policies"] = map[string]any{}
		} else {
			activePolicies["policies"] = filteredPolicies
		}

		updatedBody, err := json.Marshal(parentObj)
		if err != nil {
			return fmt.Errorf("failed to marshal updated %s %s/%s JSON: %w", obj.Kind, ns, obj.Name, err)
		}

		_, updateErr := apiClient.RestClient.Execute(ctx, http.MethodPut, targetURL, updatedBody)
		if updateErr != nil {
			return fmt.Errorf("failed to update %s %s/%s: %w", obj.Kind, ns, obj.Name, updateErr)
		}

		tflog.Info(ctx, fmt.Sprintf("[hook] successfully unbound policy %q from %s %s/%s", policyName, obj.Kind, ns, obj.Name))
	}

	return nil
}

// ServicePolicyUnbindReferencingLoadBalancers is an alias for ServicePolicyUnbindReferencingObjects for backward compatibility.
func ServicePolicyUnbindReferencingLoadBalancers(
	ctx context.Context,
	apiClient *client.APIClient,
	policyNamespace, policyName string,
	errText string,
) error {
	return ServicePolicyUnbindReferencingObjects(ctx, apiClient, policyNamespace, policyName, errText)
}

