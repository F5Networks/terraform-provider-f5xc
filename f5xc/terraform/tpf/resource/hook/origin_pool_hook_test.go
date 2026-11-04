//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"
	"encoding/json"
	"testing"
)

// poolEntry builds an OriginPoolWithWeight-shaped map with a pool reference.
func poolEntry(namespace, name string, weight int) map[string]any {
	pool := map[string]any{"name": name}
	if namespace != "" {
		pool["namespace"] = namespace
	}
	return map[string]any{
		"pool":   pool,
		"weight": weight,
	}
}

// poolNamesInList extracts the referenced pool names from a []any of
// OriginPoolWithWeight-shaped maps, for easy assertions.
func poolNamesInList(t *testing.T, raw any) []string {
	t.Helper()
	list, ok := raw.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", raw)
	}
	var names []string
	for _, item := range list {
		entry, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any list entry, got %T", item)
		}
		ref, ok := entry["pool"].(map[string]any)
		if !ok {
			t.Fatalf("expected pool ref map, got %T", entry["pool"])
		}
		names = append(names, ref["name"].(string))
	}
	return names
}

func TestFilterOriginPoolWithWeightList_RemovesMatchByName(t *testing.T) {
	pools := []any{
		poolEntry("", "pool1", 1),
		poolEntry("", "pool2", 1),
	}

	filtered, removed := filterOriginPoolWithWeightList(pools, "pool1", "default")
	if !removed {
		t.Fatalf("expected removed=true")
	}
	got := poolNamesInList(t, filtered)
	if len(got) != 1 || got[0] != "pool2" {
		t.Fatalf("expected only pool2 to remain, got %v", got)
	}
}

func TestFilterOriginPoolWithWeightList_NoMatch(t *testing.T) {
	pools := []any{
		poolEntry("", "poolA", 1),
		poolEntry("", "poolB", 1),
	}

	filtered, removed := filterOriginPoolWithWeightList(pools, "pool1", "default")
	if removed {
		t.Fatalf("expected removed=false when pool is absent")
	}
	if len(filtered) != 2 {
		t.Fatalf("expected list to be unchanged (2 entries), got %d", len(filtered))
	}
}

func TestFilterOriginPoolWithWeightList_NamespaceMismatchKeepsEntry(t *testing.T) {
	pools := []any{
		poolEntry("other-ns", "pool1", 1),
	}

	filtered, removed := filterOriginPoolWithWeightList(pools, "pool1", "default")
	if removed {
		t.Fatalf("expected removed=false: same name but different namespace must not match")
	}
	if len(filtered) != 1 {
		t.Fatalf("expected entry to be preserved, got %d entries", len(filtered))
	}
}

func TestFilterOriginPoolWithWeightList_NamespaceOmittedMatches(t *testing.T) {
	// When the reference omits namespace, name match alone is sufficient.
	pools := []any{
		poolEntry("", "pool1", 1),
	}

	filtered, removed := filterOriginPoolWithWeightList(pools, "pool1", "default")
	if !removed {
		t.Fatalf("expected removed=true when reference omits namespace")
	}
	if len(filtered) != 0 {
		t.Fatalf("expected empty list, got %d entries", len(filtered))
	}
}

func TestRemoveOriginPoolFromSpec_DefaultRoutePools(t *testing.T) {
	spec := map[string]any{
		"default_route_pools": []any{
			poolEntry("", "pool1", 1),
			poolEntry("", "pool2", 1),
		},
	}

	changed := removeOriginPoolFromSpec(context.Background(), spec, "http_loadbalancers", "default", "lb1", "pool1", "default")
	if !changed {
		t.Fatalf("expected changed=true")
	}

	got := poolNamesInList(t, spec["default_route_pools"])
	if len(got) != 1 || got[0] != "pool2" {
		t.Fatalf("expected only pool2 to remain in default_route_pools, got %v", got)
	}
}

func TestRemoveOriginPoolFromSpec_DefaultRoutePoolsRemovedWhenEmpty(t *testing.T) {
	spec := map[string]any{
		"default_route_pools": []any{
			poolEntry("", "pool1", 1),
		},
	}

	changed := removeOriginPoolFromSpec(context.Background(), spec, "http_loadbalancers", "default", "lb1", "pool1", "default")
	if !changed {
		t.Fatalf("expected changed=true")
	}
	if _, ok := spec["default_route_pools"]; ok {
		t.Fatalf("expected default_route_pools key to be deleted when it becomes empty")
	}
}

func TestRemoveOriginPoolFromSpec_RouteOriginPools(t *testing.T) {
	spec := map[string]any{
		"routes": []any{
			map[string]any{
				"simple_route": map[string]any{
					"origin_pools": []any{
						poolEntry("", "pool1", 1),
						poolEntry("", "pool3", 1),
					},
				},
			},
		},
	}

	changed := removeOriginPoolFromSpec(context.Background(), spec, "http_loadbalancers", "default", "lb1", "pool1", "default")
	if !changed {
		t.Fatalf("expected changed=true")
	}

	routes := spec["routes"].([]any)
	simpleRoute := routes[0].(map[string]any)["simple_route"].(map[string]any)
	got := poolNamesInList(t, simpleRoute["origin_pools"])
	if len(got) != 1 || got[0] != "pool3" {
		t.Fatalf("expected only pool3 to remain in route origin_pools, got %v", got)
	}
}

func TestRemoveOriginPoolFromSpec_DefaultPoolList(t *testing.T) {
	spec := map[string]any{
		"default_pool_list": map[string]any{
			"pools": []any{
				poolEntry("default", "pool1", 1),
				poolEntry("default", "pool2", 1),
			},
		},
	}

	changed := removeOriginPoolFromSpec(context.Background(), spec, "http_loadbalancers", "default", "lb1", "pool1", "default")
	if !changed {
		t.Fatalf("expected changed=true")
	}

	dpl := spec["default_pool_list"].(map[string]any)
	got := poolNamesInList(t, dpl["pools"])
	if len(got) != 1 || got[0] != "pool2" {
		t.Fatalf("expected only pool2 to remain in default_pool_list.pools, got %v", got)
	}
}

func TestRemoveOriginPoolFromSpec_DefaultPoolListKeptWhenEmpty(t *testing.T) {
	// origin_pool_choice is a required oneof; even if the pool list becomes empty we must
	// NOT delete the default_pool_list container. (The LB update from Terraform will supply
	// a valid replacement; corrupting the required choice here would break the LB.)
	spec := map[string]any{
		"default_pool_list": map[string]any{
			"pools": []any{
				poolEntry("default", "pool1", 1),
			},
		},
	}

	changed := removeOriginPoolFromSpec(context.Background(), spec, "http_loadbalancers", "default", "lb1", "pool1", "default")
	if !changed {
		t.Fatalf("expected changed=true")
	}

	dpl, ok := spec["default_pool_list"].(map[string]any)
	if !ok {
		t.Fatalf("expected default_pool_list container to be preserved (required oneof)")
	}
	pools := dpl["pools"].([]any)
	if len(pools) != 0 {
		t.Fatalf("expected pools list to be emptied, got %d", len(pools))
	}
}

func TestRemoveOriginPoolFromSpec_BothLocations(t *testing.T) {
	spec := map[string]any{
		"default_route_pools": []any{
			poolEntry("", "pool1", 1),
			poolEntry("", "pool2", 1),
		},
		"routes": []any{
			map[string]any{
				"simple_route": map[string]any{
					"origin_pools": []any{
						poolEntry("", "pool1", 1),
					},
				},
			},
		},
	}

	changed := removeOriginPoolFromSpec(context.Background(), spec, "http_loadbalancers", "default", "lb1", "pool1", "default")
	if !changed {
		t.Fatalf("expected changed=true")
	}

	drp := poolNamesInList(t, spec["default_route_pools"])
	if len(drp) != 1 || drp[0] != "pool2" {
		t.Fatalf("expected only pool2 in default_route_pools, got %v", drp)
	}

	routes := spec["routes"].([]any)
	simpleRoute := routes[0].(map[string]any)["simple_route"].(map[string]any)
	got := poolNamesInList(t, simpleRoute["origin_pools"])
	if len(got) != 0 {
		t.Fatalf("expected route origin_pools to be emptied, got %v", got)
	}
}

func TestRemoveOriginPoolFromSpec_NoMatchReturnsFalse(t *testing.T) {
	spec := map[string]any{
		"default_route_pools": []any{
			poolEntry("", "poolX", 1),
		},
		"routes": []any{
			map[string]any{
				"simple_route": map[string]any{
					"origin_pools": []any{
						poolEntry("", "poolY", 1),
					},
				},
			},
		},
	}

	changed := removeOriginPoolFromSpec(context.Background(), spec, "http_loadbalancers", "default", "lb1", "pool1", "default")
	if changed {
		t.Fatalf("expected changed=false when the pool is not referenced anywhere")
	}
}

func TestOriginPoolUnbindReferencingObjects_NilClient(t *testing.T) {
	err := OriginPoolUnbindReferencingObjects(context.Background(), nil, "default", "pool1", "some error")
	if err == nil {
		t.Fatalf("expected an error when apiClient is nil")
	}
}

// TestOriginPoolUnbind_ParsesReferringObjects verifies that the 409 error string produced
// by the F5XC API for a referenced origin_pool is parsed into the referring load balancers,
// which drives the unbind logic.
func TestOriginPoolUnbind_ParsesReferringObjects(t *testing.T) {
	errText := "DeleteResource: ves.io.schema.views.origin_pool.Object tenant/default/pool1 " +
		"is still being referred by 1 objects - [ves.io.schema.views.http_loadbalancer.Object default/lb1]"

	refs := ParseReferringObjects(errText)
	if len(refs) != 1 {
		t.Fatalf("expected 1 referring object, got %d: %+v", len(refs), refs)
	}
	if refs[0].Kind != "http_loadbalancers" {
		t.Fatalf("expected kind http_loadbalancers, got %q", refs[0].Kind)
	}
	if refs[0].Namespace != "default" || refs[0].Name != "lb1" {
		t.Fatalf("expected default/lb1, got %s/%s", refs[0].Namespace, refs[0].Name)
	}
}

// TestRemoveOriginPoolFromSpec_RoundTripJSON makes sure that a spec unmarshalled from JSON
// (as it would be fetched from the API) is correctly mutated and can be re-marshalled.
func TestRemoveOriginPoolFromSpec_RoundTripJSON(t *testing.T) {
	raw := `{
      "spec": {
        "default_route_pools": [
          {"pool": {"name": "pool1", "namespace": "default"}, "weight": 1},
          {"pool": {"name": "pool2", "namespace": "default"}, "weight": 1}
        ]
      }
    }`

	var parent map[string]any
	if err := json.Unmarshal([]byte(raw), &parent); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	spec := parent["spec"].(map[string]any)

	changed := removeOriginPoolFromSpec(context.Background(), spec, "http_loadbalancers", "default", "lb1", "pool1", "default")
	if !changed {
		t.Fatalf("expected changed=true")
	}

	out, err := json.Marshal(parent)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if got := string(out); got == "" {
		t.Fatalf("expected non-empty marshalled output")
	}

	got := poolNamesInList(t, spec["default_route_pools"])
	if len(got) != 1 || got[0] != "pool2" {
		t.Fatalf("expected only pool2 to remain, got %v", got)
	}
}

// TestIsLoadBalancerKind_FiltersReferrers locks in the referrer filtering behind the
// XC-3073 fix. The origin_pool unbind no longer gates on an in-memory managed-object
// registry (Terraform can run the pool destroy before the referring LB's
// create/update/read in the same apply, leaving such a registry empty and skipping the
// unbind). Instead it only acts on referrers that are load balancers, and the generic
// 409 parser also picks the pool itself out of the "DeleteResource: ..." prefix, which
// must be ignored.
func TestIsLoadBalancerKind_FiltersReferrers(t *testing.T) {
	loadBalancers := []string{
		"http_loadbalancers",
		"tcp_loadbalancers",
		"udp_loadbalancers",
		"cdn_loadbalancers",
	}
	for _, kind := range loadBalancers {
		if !isLoadBalancerKind(kind) {
			t.Fatalf("expected %q to be treated as a load balancer referrer", kind)
		}
	}

	notLoadBalancers := []string{
		"origin_pools", // the pool being deleted, echoed back in the 409 prefix
		"origin_pool",
		"routes",
		"",
		"http_loadbalancer", // singular form is not what ParseReferringObjects emits
	}
	for _, kind := range notLoadBalancers {
		if isLoadBalancerKind(kind) {
			t.Fatalf("expected %q NOT to be treated as a load balancer referrer", kind)
		}
	}
}
