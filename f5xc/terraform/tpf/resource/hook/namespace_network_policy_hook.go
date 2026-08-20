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

	"github.com/hashicorp/terraform-plugin-log/tflog"

	ves_io_schema_namespace "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/namespace"
	"gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/views"
)

var networkPolicyNamespaceLocks sync.Map

// lockNetworkPolicyNamespace serializes active list mutation and delete operations
// for network policies within a namespace.
func lockNetworkPolicyNamespace(namespace string) func() {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = "_default"
	}

	muAny, _ := networkPolicyNamespaceLocks.LoadOrStore(ns, &sync.Mutex{})
	mu := muAny.(*sync.Mutex)
	mu.Lock()

	return mu.Unlock
}

// NetworkPolicyPreDeleteHook is a pre-delete hook for the f5xc_network_policy resource.
//
// Before a network policy can be deleted it must be removed from the list of active
// network policies in its namespace. If the policy is not in the active list the hook
// is a no-op.
func NetworkPolicyPreDeleteHook(ctx context.Context, apiClient CustomAPIClient, namespace, policyName string) error {
	unlock := lockNetworkPolicyNamespace(namespace)
	defer unlock()

	return networkPolicyPreDeleteHookUnlocked(ctx, apiClient, namespace, policyName)
}

// networkPolicyPreDeleteHookUnlocked performs active list cleanup without taking
// a namespace lock. Callers that require serialization must lock externally.
func networkPolicyPreDeleteHookUnlocked(ctx context.Context, apiClient CustomAPIClient, namespace, policyName string) error {
	tflog.Info(ctx, fmt.Sprintf("[hook] NetworkPolicyPreDeleteHook: checking if %q is active in namespace %q", policyName, namespace))

	getRPCFQN := "ves.io.schema.namespace.NamespaceCustomAPI.GetActiveNetworkPolicies"
	getURI := fmt.Sprintf("/public/namespaces/%s/active_network_policies", namespace)

	getResp, err := apiClient.CustomAPI(ctx, "GET", getURI, getRPCFQN, "")
	if err != nil {
		if strings.Contains(err.Error(), "status code 404") {
			tflog.Info(ctx, fmt.Sprintf("[hook] namespace %q not found (404), skipping active policy cleanup", namespace))
			return nil
		}
		if strings.Contains(err.Error(), "status code 400") {
			tflog.Info(ctx, fmt.Sprintf("[hook] namespace %q does not support active network policies (400), skipping cleanup", namespace))
			return nil
		}
		return fmt.Errorf("NetworkPolicyPreDeleteHook: failed to get active network policies: %w", err)
	}

	activePoliciesResp, ok := getResp.(*ves_io_schema_namespace.GetActiveNetworkPoliciesResponse)
	if !ok {
		return fmt.Errorf("NetworkPolicyPreDeleteHook: unexpected response type %T, expected *GetActiveNetworkPoliciesResponse", getResp)
	}

	currentPolicies := activePoliciesResp.GetNetworkPolicies()
	tflog.Info(ctx, fmt.Sprintf("[hook] active network policies in namespace %q: %v", namespace, policyNames(currentPolicies)))

	filteredPolicies := make([]*views.ObjectRefType, 0, len(currentPolicies))
	found := false
	for _, ref := range currentPolicies {
		if ref.GetName() == policyName {
			found = true
			tflog.Info(ctx, fmt.Sprintf("[hook] network policy %q found in active list - will remove before delete", policyName))
			continue
		}
		filteredPolicies = append(filteredPolicies, ref)
	}

	if !found {
		tflog.Info(ctx, fmt.Sprintf("[hook] network policy %q is NOT in the active list - no cleanup needed", policyName))
		return nil
	}

	setRPCFQN := "ves.io.schema.namespace.NamespaceCustomAPI.SetActiveNetworkPolicies"
	setURI := fmt.Sprintf("/public/namespaces/%s/active_network_policies", namespace)

	type networkPolicyRef struct {
		Tenant    string `json:"tenant,omitempty"`
		Namespace string `json:"namespace,omitempty"`
		Name      string `json:"name"`
	}
	type setReqBody struct {
		Namespace       string             `json:"namespace"`
		NetworkPolicies []networkPolicyRef `json:"network_policies"`
	}

	refs := make([]networkPolicyRef, 0, len(filteredPolicies))
	for _, ref := range filteredPolicies {
		refs = append(refs, networkPolicyRef{
			Tenant:    ref.GetTenant(),
			Namespace: ref.GetNamespace(),
			Name:      ref.GetName(),
		})
	}

	bodyBytes, err := json.Marshal(setReqBody{
		Namespace:       namespace,
		NetworkPolicies: refs,
	})
	if err != nil {
		return fmt.Errorf("NetworkPolicyPreDeleteHook: failed to marshal SetActiveNetworkPolicies request: %w", err)
	}

	tflog.Info(ctx, fmt.Sprintf("[hook] updating active network policies for namespace %q: %s", namespace, string(bodyBytes)))

	_, err = apiClient.CustomAPI(ctx, "POST", setURI, setRPCFQN, string(bodyBytes))
	if err != nil {
		return fmt.Errorf("NetworkPolicyPreDeleteHook: failed to update active network policies: %w", err)
	}

	tflog.Info(ctx, fmt.Sprintf("[hook] successfully removed %q from active network policies in namespace %q", policyName, namespace))
	return nil
}
