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
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gopkg.volterra.us/stdlib/client/vesapi"

	ves_io_schema_namespace "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/namespace"
	"gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/views"
)

// CustomAPIClient is the minimal interface required by the hooks to call F5XC Custom APIs.
// It is satisfied by *client.APIClient.
type CustomAPIClient interface {
	CustomAPI(ctx context.Context, method, uri, rpcFQN, reqYml string, opts ...vesapi.CallOpt) (proto.Message, error)
}

// ServicePolicyPreDeleteHook is a pre-delete hook for the f5xc_service_policy resource.
//
// Before a service policy can be deleted it must be removed from the list of active
// service policies in its namespace. If the policy is not in the active list the hook
// is a no-op.
//
// Flow:
//  1. GET  /public/namespaces/<ns>/active_service_policies  → list current active policies
//  2. Check whether <policyName> is in the list
//  3. If yes: POST /public/namespaces/<ns>/active_service_policies with the filtered list
//     (i.e. the original list minus the policy being deleted)
//  4. Return – the caller can then proceed with the actual DeleteObject call
func ServicePolicyPreDeleteHook(ctx context.Context, apiClient CustomAPIClient, namespace, policyName string) error {
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
