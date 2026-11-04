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

// OriginPoolUnbindReferencingObjects parses an F5XC 409 conflict error raised when
// deleting an origin_pool, finds any referring load balancers (http_loadbalancer,
// tcp_loadbalancer, cdn_loadbalancer, etc.), and removes the origin pool from their
// spec on the F5XC API so the pool can subsequently be deleted.
//
// This fixes the case where, in a single `terraform apply`, a user deletes an
// origin_pool while updating a load balancer to stop referencing it. Terraform may
// attempt the origin_pool delete before/while the LB still references it, and the
// API rejects the delete with a 409 "is still being referred by" error. This reactive
// unbind hook detaches the pool reference from the LBs Terraform manages, allowing the
// delete to be retried successfully.
//
// An origin pool can be referenced from a load balancer spec in three places:
//   - spec.default_route_pools[].pool                 (default route pools)
//   - spec.default_pool_list.pools[].pool             (origin_pool_choice = default_pool_list)
//   - spec.routes[].simple_route.origin_pools[].pool  (per-route origin pools)
//
// The unbind is safe by construction: it only removes the single pool reference
// (poolNamespace/poolName) that Terraform is actively deleting, and only from a load
// balancer that the F5XC API itself reported as referencing that exact pool. It never
// deletes a load balancer and never removes any other pool reference. It deliberately
// does NOT gate on an in-memory "managed object" registry — Terraform's apply graph can
// run this origin_pool destroy before it runs the referring LB's create/update/read in
// the same process (XC-3073), which would leave such a registry empty and cause the
// unbind to be skipped and the delete to fail with a 409.
func OriginPoolUnbindReferencingObjects(
	ctx context.Context,
	apiClient *client.APIClient,
	poolNamespace, poolName string,
	errText string,
) error {
	if apiClient == nil || apiClient.RestClient == nil {
		return fmt.Errorf("OriginPoolUnbindReferencingObjects: apiClient or RestClient is nil")
	}

	referringObjs := ParseReferringObjects(errText)
	if len(referringObjs) == 0 {
		tflog.Info(ctx, fmt.Sprintf("[hook] OriginPool: no referring objects found in error text: %s", errText))
		return nil
	}

	baseURL := strings.TrimRight(apiClient.RestClient.URL, "/")

	for _, obj := range referringObjs {
		ns := obj.Namespace
		if ns == "" {
			ns = poolNamespace
		}

		// The 409 error text repeats the pool being deleted in its "DeleteResource: ..."
		// prefix; the generic parser also picks that up. Never treat the origin pool itself
		// (or anything in the same namespace/name that is not a load balancer) as a referrer.
		if !isLoadBalancerKind(obj.Kind) {
			tflog.Info(ctx, fmt.Sprintf("[hook] OriginPool: skipping non-load-balancer referrer %s %s/%s", obj.Kind, ns, obj.Name))
			continue
		}

		// NOTE: This intentionally does NOT gate on an in-memory "managed object" set.
		// Terraform's apply graph frequently runs the origin_pool destroy BEFORE it runs
		// the referring load balancer's create/update/read in the same process (see
		// XC-3073), so such a gate silently skips the unbind and the delete fails with 409.
		//
		// The operation is safe by construction instead: we only ever remove the ONE pool
		// reference (poolNamespace/poolName) that Terraform is actively deleting, and only
		// from a load balancer that the F5XC API itself reported as referencing that exact
		// pool. We never delete the load balancer and never touch any other pool reference.
		tflog.Info(ctx, fmt.Sprintf("[hook] OriginPool: unbinding pool %q from %s %q in namespace %q", poolName, obj.Kind, obj.Name, ns))

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

		changed := removeOriginPoolFromSpec(ctx, spec, obj.Kind, ns, obj.Name, poolName, poolNamespace)
		if !changed {
			tflog.Info(ctx, fmt.Sprintf("[hook] OriginPool: pool %q not found in %s %s/%s spec, nothing to unbind", poolName, obj.Kind, ns, obj.Name))
			continue
		}

		updatedBody, err := json.Marshal(parentObj)
		if err != nil {
			return fmt.Errorf("failed to marshal updated %s %s/%s JSON: %w", obj.Kind, ns, obj.Name, err)
		}

		_, updateErr := apiClient.RestClient.Execute(ctx, http.MethodPut, targetURL, updatedBody)
		if updateErr != nil {
			return fmt.Errorf("failed to update %s %s/%s: %w", obj.Kind, ns, obj.Name, updateErr)
		}

		tflog.Info(ctx, fmt.Sprintf("[hook] OriginPool: successfully unbound pool %q from %s %s/%s", poolName, obj.Kind, ns, obj.Name))
	}

	return nil
}

// isLoadBalancerKind reports whether a referring-object kind (as produced by
// ParseReferringObjects, e.g. "http_loadbalancers") is a load balancer view that can
// carry origin pool references. This filters out the origin pool's own object type that
// the generic 409 error parser also picks up from the "DeleteResource: ..." prefix.
func isLoadBalancerKind(kind string) bool {
	switch kind {
	case "http_loadbalancers", "tcp_loadbalancers", "udp_loadbalancers", "cdn_loadbalancers":
		return true
	default:
		return false
	}
}

// removeOriginPoolFromSpec removes any reference to (poolNamespace/poolName) from a load
// balancer spec map, covering spec.default_route_pools, spec.default_pool_list.pools and
// each route's simple_route.origin_pools list. It returns true when at least one reference
// was removed.
//
// The comparison matches on pool name, and additionally on namespace when the reference
// carries one (references frequently omit namespace when they live in the same namespace
// as the load balancer).
func removeOriginPoolFromSpec(
	ctx context.Context,
	spec map[string]any,
	kind, lbNamespace, lbName string,
	poolName, poolNamespace string,
) bool {
	changed := false

	// 1) spec.default_route_pools[].pool
	if rawPools, ok := spec["default_route_pools"].([]any); ok && rawPools != nil {
		filtered, removed := filterOriginPoolWithWeightList(rawPools, poolName, poolNamespace)
		if removed {
			changed = true
			tflog.Info(ctx, fmt.Sprintf("[hook] OriginPool: removing pool %q from default_route_pools on %s %s/%s", poolName, kind, lbNamespace, lbName))
			if len(filtered) == 0 {
				delete(spec, "default_route_pools")
			} else {
				spec["default_route_pools"] = filtered
			}
		}
	}

	// 2) spec.default_pool_list.pools[].pool
	//    default_pool_list is one variant of the required origin_pool_choice oneof and is a
	//    very common way an LB references origin pools by name. We only prune the matching
	//    pool from the list; we never remove the default_pool_list container itself, because
	//    origin_pool_choice is required and has no empty variant.
	if defaultPoolList, ok := spec["default_pool_list"].(map[string]any); ok && defaultPoolList != nil {
		if rawPools, ok := defaultPoolList["pools"].([]any); ok && rawPools != nil {
			filtered, removed := filterOriginPoolWithWeightList(rawPools, poolName, poolNamespace)
			if removed {
				changed = true
				tflog.Info(ctx, fmt.Sprintf("[hook] OriginPool: removing pool %q from default_pool_list on %s %s/%s", poolName, kind, lbNamespace, lbName))
				defaultPoolList["pools"] = filtered
			}
		}
	}

	// 3) spec.routes[].simple_route.origin_pools[].pool
	if rawRoutes, ok := spec["routes"].([]any); ok && rawRoutes != nil {
		for _, rawRoute := range rawRoutes {
			route, ok := rawRoute.(map[string]any)
			if !ok {
				continue
			}
			simpleRoute, ok := route["simple_route"].(map[string]any)
			if !ok || simpleRoute == nil {
				continue
			}
			rawOriginPools, ok := simpleRoute["origin_pools"].([]any)
			if !ok || rawOriginPools == nil {
				continue
			}
			filtered, removed := filterOriginPoolWithWeightList(rawOriginPools, poolName, poolNamespace)
			if removed {
				changed = true
				tflog.Info(ctx, fmt.Sprintf("[hook] OriginPool: removing pool %q from a route's origin_pools on %s %s/%s", poolName, kind, lbNamespace, lbName))
				simpleRoute["origin_pools"] = filtered
			}
		}
	}

	return changed
}

// filterOriginPoolWithWeightList returns a copy of pools with any entry whose
// .pool reference matches (poolName[/poolNamespace]) removed, plus a bool indicating
// whether anything was removed.
func filterOriginPoolWithWeightList(pools []any, poolName, poolNamespace string) ([]any, bool) {
	removed := false
	filtered := make([]any, 0, len(pools))
	for _, raw := range pools {
		entry, ok := raw.(map[string]any)
		if !ok {
			filtered = append(filtered, raw)
			continue
		}
		poolRef, ok := entry["pool"].(map[string]any)
		if !ok || poolRef == nil {
			filtered = append(filtered, raw)
			continue
		}
		refName, _ := poolRef["name"].(string)
		if refName != poolName {
			filtered = append(filtered, raw)
			continue
		}
		// Name matches. If the reference carries a namespace, require it to match too.
		if refNS, ok := poolRef["namespace"].(string); ok && refNS != "" && poolNamespace != "" && refNS != poolNamespace {
			filtered = append(filtered, raw)
			continue
		}
		removed = true
	}
	return filtered, removed
}
