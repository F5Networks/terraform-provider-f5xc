//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"testing"
)

var networkPolicyWrapper = unbindWrapper{
	call:            NetworkPolicyUnbindReferencingObjects,
	logPrefix:       "NetworkPolicy",
	parentSchema:    "virtual_network",
	parentKind:      "virtual_networks",
	selfSchema:      "network_policy",
	selfKind:        "network_policys",
	selfIsFiltered:  true,
	containerKey:    "active_network_policies",
	listKey:         "network_policies",
	emptyChoiceKeys: []string{"no_network_policy"},
}

// TestNetworkPolicyUnbindReferencingObjects verifies the shared unbind contract
// for network_policy.
func TestNetworkPolicyUnbindReferencingObjects(t *testing.T) {
	runUnbindWrapperSuite(t, networkPolicyWrapper)
}

// TestNetworkPolicyUnbind_ReferrerKinds verifies that only virtual network,
// network policy set, and site parents are unbound.
func TestNetworkPolicyUnbind_ReferrerKinds(t *testing.T) {
	accepted := map[string]string{
		"virtual_networks":    "virtual_network",
		"network_policy_sets": "network_policy_set",
		"appstack_sites":      "views.appstack_site",
		"aws_vpc_sites":       "views.aws_vpc_site",
		"azure_vnet_sites":    "views.azure_vnet_site",
		"gcp_vpc_sites":       "views.gcp_vpc_site",
		"aws_tgw_sites":       "views.aws_tgw_site",
		"securemesh_sites":    "views.securemesh_site",
	}
	rejected := map[string]string{
		"network_policys":    "network_policy",
		"http_loadbalancers": "views.http_loadbalancer",
		"origin_pools":       "views.origin_pool",
	}

	checkUnbindReferrerKinds(t, networkPolicyWrapper, accepted, rejected)
}
