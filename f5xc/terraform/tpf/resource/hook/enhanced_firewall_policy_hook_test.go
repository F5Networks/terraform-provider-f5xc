//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"testing"
)

var enhancedFirewallPolicyWrapper = unbindWrapper{
	call:            EnhancedFirewallPolicyUnbindReferencingObjects,
	logPrefix:       "EnhancedFirewallPolicy",
	parentSchema:    "views.aws_vpc_site",
	parentKind:      "aws_vpc_sites",
	selfSchema:      "enhanced_firewall_policy",
	selfKind:        "enhanced_firewall_policys",
	selfIsFiltered:  true,
	containerKey:    "active_enhanced_firewall_policies",
	listKey:         "enhanced_firewall_policies",
	emptyChoiceKeys: []string{"no_firewall_policy", "no_enhanced_firewall_policy"},
}

// TestEnhancedFirewallPolicyUnbindReferencingObjects verifies the shared unbind
// contract for enhanced_firewall_policy. When the list empties, both
// no_firewall_policy and no_enhanced_firewall_policy must be set.
func TestEnhancedFirewallPolicyUnbindReferencingObjects(t *testing.T) {
	runUnbindWrapperSuite(t, enhancedFirewallPolicyWrapper)
}

// TestEnhancedFirewallPolicyUnbind_ReferrerKinds verifies that only site and
// virtual network parents are unbound.
func TestEnhancedFirewallPolicyUnbind_ReferrerKinds(t *testing.T) {
	accepted := map[string]string{
		"appstack_sites":   "views.appstack_site",
		"aws_vpc_sites":    "views.aws_vpc_site",
		"azure_vnet_sites": "views.azure_vnet_site",
		"gcp_vpc_sites":    "views.gcp_vpc_site",
		"aws_tgw_sites":    "views.aws_tgw_site",
		"securemesh_sites": "views.securemesh_site",
		"virtual_networks": "virtual_network",
	}
	rejected := map[string]string{
		"enhanced_firewall_policys": "enhanced_firewall_policy",
		"http_loadbalancers":        "views.http_loadbalancer",
		"network_policy_sets":       "network_policy_set",
	}

	checkUnbindReferrerKinds(t, enhancedFirewallPolicyWrapper, accepted, rejected)
}
