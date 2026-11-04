//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/client"
)

// policyUnbindSpec describes how a particular policy kind (forward_proxy_policy,
// network_policy, enhanced_firewall_policy) is referenced inside a parent resource's
// spec, so the generic reactive unbind logic knows what to prune.
//
// A policy reference always lives inside an "active" container that holds a repeated
// list of ObjectRefType entries, and that container is one variant of a required oneof
// whose "empty" variant is a "no_*" block. For example:
//
//	spec...active_forward_proxy_policies.forward_proxy_policies[] (+ sibling no_forward_proxy_policy)
//	spec...active_network_policies.network_policies[]             (+ sibling no_network_policy)
//	spec...active_enhanced_firewall_policies.enhanced_firewall_policies[] (+ sibling no_enhanced_firewall_policy)
//
// These containers appear at varying depths across the different parent kinds (origin
// pools, load balancers, sites, virtual networks, network policy sets), so we locate them
// with a recursive walk instead of hard-coding every path.
type policyUnbindSpec struct {
	// logPrefix is the short human-readable policy name used in log lines, e.g. "ForwardProxyPolicy".
	logPrefix string
	// containerKey is the "active_*_policies" map key that wraps the policy list.
	containerKey string
	// listKey is the repeated ObjectRefType field inside the container, e.g. "forward_proxy_policies".
	listKey string
	// emptyChoiceKeys are the sibling "no_*" oneof variants to set (to {}) when the policy
	// list becomes empty. Multiple keys are supported because different parents name the
	// empty choice differently (e.g. sites use "no_firewall_policy" for enhanced firewall
	// policies while the standalone container uses "no_enhanced_firewall_policy").
	emptyChoiceKeys []string
	// isReferrerKind reports whether a referring-object kind (plural REST segment, e.g.
	// "http_loadbalancers") is a parent this policy can actually be bound to. It filters
	// out the policy's own object type that the generic 409 parser also extracts from the
	// "DeleteResource: ..." prefix.
	isReferrerKind func(kind string) bool
}

// policyUnbindReferencingObjects implements the shared reactive pre-delete unbind flow
// for policy resources. It parses an F5XC 409 "is still being referred by" error, then for
// each referring parent object it GETs the object, removes the (policyNamespace/policyName)
// reference from the policy list described by cfg, PUTs the object back, and returns. The
// caller then retries the policy DELETE.
//
// Like the origin_pool unbind (XC-3073), this deliberately does NOT gate on an in-memory
// "managed object" registry: Terraform's apply graph can run the policy destroy before it
// runs the referring parent's create/update/read in the same process, which would leave
// such a registry empty and cause the unbind to be skipped and the delete to fail with 409.
// The operation is safe by construction: it only ever removes the single policy reference
// Terraform is actively deleting, and only from an object the F5XC API itself reported as
// referencing that exact policy. It never deletes the parent object.
func policyUnbindReferencingObjects(
	ctx context.Context,
	apiClient *client.APIClient,
	policyNamespace, policyName, errText string,
	cfg policyUnbindSpec,
) error {
	if apiClient == nil || apiClient.RestClient == nil {
		return fmt.Errorf("%sUnbindReferencingObjects: apiClient or RestClient is nil", cfg.logPrefix)
	}

	referringObjs := ParseReferringObjects(errText)
	if len(referringObjs) == 0 {
		tflog.Info(ctx, fmt.Sprintf("[hook] %s: no referring objects found in error text: %s", cfg.logPrefix, errText))
		return nil
	}

	baseURL := strings.TrimRight(apiClient.RestClient.URL, "/")

	for _, obj := range referringObjs {
		ns := obj.Namespace
		if ns == "" {
			ns = policyNamespace
		}

		if cfg.isReferrerKind != nil && !cfg.isReferrerKind(obj.Kind) {
			tflog.Info(ctx, fmt.Sprintf("[hook] %s: skipping non-referrer %s %s/%s", cfg.logPrefix, obj.Kind, ns, obj.Name))
			continue
		}

		tflog.Info(ctx, fmt.Sprintf("[hook] %s: unbinding policy %q from %s %q in namespace %q", cfg.logPrefix, policyName, obj.Kind, obj.Name, ns))

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
			// A single referrer that cannot be fetched (e.g. it was already deleted in
			// the same apply, or is temporarily unreachable) must not abort unbinding the
			// remaining referrers: the whole point of this loop is to clear every reference
			// that is blocking the policy delete. Skip this one and continue; the caller's
			// delete retry will surface a fresh 409 if any real reference still remains.
			tflog.Warn(ctx, fmt.Sprintf("[hook] %s: failed to fetch referrer %s %s/%s, skipping: %v", cfg.logPrefix, obj.Kind, ns, obj.Name, fetchErr))
			continue
		}

		var parentObj map[string]any
		if err := json.Unmarshal(respBytes, &parentObj); err != nil {
			tflog.Warn(ctx, fmt.Sprintf("[hook] %s: failed to parse referrer %s %s/%s JSON, skipping: %v", cfg.logPrefix, obj.Kind, ns, obj.Name, err))
			continue
		}

		spec, ok := parentObj["spec"].(map[string]any)
		if !ok || spec == nil {
			continue
		}

		changed := removePolicyRefFromNode(ctx, spec, cfg, obj.Kind, ns, obj.Name, policyName, policyNamespace)
		if !changed {
			tflog.Info(ctx, fmt.Sprintf("[hook] %s: policy %q not found in %s %s/%s spec, nothing to unbind", cfg.logPrefix, policyName, obj.Kind, ns, obj.Name))
			continue
		}

		updatedBody, err := json.Marshal(parentObj)
		if err != nil {
			return fmt.Errorf("failed to marshal updated %s %s/%s JSON: %w", obj.Kind, ns, obj.Name, err)
		}

		if _, updateErr := apiClient.RestClient.Execute(ctx, http.MethodPut, targetURL, updatedBody); updateErr != nil {
			return fmt.Errorf("failed to update %s %s/%s: %w", obj.Kind, ns, obj.Name, updateErr)
		}

		tflog.Info(ctx, fmt.Sprintf("[hook] %s: successfully unbound policy %q from %s %s/%s", cfg.logPrefix, policyName, obj.Kind, ns, obj.Name))
	}

	return nil
}

// removePolicyRefFromNode recursively walks node looking for maps that contain
// cfg.containerKey (the "active_*_policies" block). Wherever it finds one, it prunes the
// (policyNamespace/policyName) reference from the policy list; if the list becomes empty it
// removes the container and sets the sibling "no_*" empty choice on the same parent map so
// the required oneof stays satisfied. It returns true if it changed anything.
func removePolicyRefFromNode(
	ctx context.Context,
	node any,
	cfg policyUnbindSpec,
	kind, parentNamespace, parentName string,
	policyName, policyNamespace string,
) bool {
	changed := false

	switch typed := node.(type) {
	case map[string]any:
		for key, val := range typed {
			if key == cfg.containerKey {
				if container, ok := val.(map[string]any); ok && container != nil {
					if removePolicyRefFromContainer(ctx, typed, container, cfg, kind, parentNamespace, parentName, policyName, policyNamespace) {
						changed = true
					}
					continue
				}
			}
			if removePolicyRefFromNode(ctx, val, cfg, kind, parentNamespace, parentName, policyName, policyNamespace) {
				changed = true
			}
		}
	case []any:
		for _, item := range typed {
			if removePolicyRefFromNode(ctx, item, cfg, kind, parentNamespace, parentName, policyName, policyNamespace) {
				changed = true
			}
		}
	}

	return changed
}

// removePolicyRefFromContainer prunes the matching policy reference from container[listKey].
// parent is the map that holds container under cfg.containerKey, so that when the list
// empties we can swap the container for the empty "no_*" choice on parent.
func removePolicyRefFromContainer(
	ctx context.Context,
	parent, container map[string]any,
	cfg policyUnbindSpec,
	kind, parentNamespace, parentName string,
	policyName, policyNamespace string,
) bool {
	rawList, ok := container[cfg.listKey].([]any)
	if !ok || rawList == nil {
		return false
	}

	filtered, removed := filterPolicyRefList(rawList, policyName, policyNamespace)
	if !removed {
		return false
	}

	tflog.Info(ctx, fmt.Sprintf("[hook] %s: removing policy %q from %s.%s on %s %s/%s", cfg.logPrefix, policyName, cfg.containerKey, cfg.listKey, kind, parentNamespace, parentName))

	if len(filtered) == 0 {
		// The oneof is required and its list variant would now be empty; switch to the
		// empty "no_*" variant instead.
		delete(parent, cfg.containerKey)
		for _, emptyKey := range cfg.emptyChoiceKeys {
			parent[emptyKey] = map[string]any{}
		}
	} else {
		container[cfg.listKey] = filtered
	}
	return true
}

// filterPolicyRefList returns a copy of a []ObjectRefType-shaped list with any entry whose
// name (and namespace, when present on the reference) matches (policyName/policyNamespace)
// removed, plus a bool indicating whether anything was removed. References frequently omit
// the namespace when the policy lives in the same namespace as the parent, so a missing or
// empty namespace on the reference is treated as a match.
func filterPolicyRefList(list []any, policyName, policyNamespace string) ([]any, bool) {
	removed := false
	filtered := make([]any, 0, len(list))
	for _, raw := range list {
		ref, ok := raw.(map[string]any)
		if !ok {
			filtered = append(filtered, raw)
			continue
		}
		refName, _ := ref["name"].(string)
		if refName != policyName {
			filtered = append(filtered, raw)
			continue
		}
		if refNS, ok := ref["namespace"].(string); ok && refNS != "" && policyNamespace != "" && refNS != policyNamespace {
			filtered = append(filtered, raw)
			continue
		}
		removed = true
	}
	return filtered, removed
}
