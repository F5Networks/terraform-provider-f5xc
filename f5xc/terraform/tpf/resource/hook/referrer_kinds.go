//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

// referrerKinds contains the plural REST endpoint segments that can reference
// an inline policy. Keeping these sets together makes adding a new parent kind
// a data-only change for each policy.
type referrerKinds []string

func (kinds referrerKinds) contains(kind string) bool {
	for _, candidate := range kinds {
		if candidate == kind {
			return true
		}
	}

	return false
}

var enhancedFirewallPolicyReferrerKinds = referrerKinds{
	"appstack_sites",
	"aws_vpc_sites",
	"azure_vnet_sites",
	"gcp_vpc_sites",
	"aws_tgw_sites",
	"securemesh_sites",
	"virtual_networks",
}

var forwardProxyPolicyReferrerKinds = referrerKinds{
	"origin_pools",
	"http_loadbalancers",
	"tcp_loadbalancers",
	"proxys",
	"forward_proxy_policys",
}

var networkPolicyReferrerKinds = referrerKinds{
	"virtual_networks",
	"network_policy_sets",
	"appstack_sites",
	"aws_vpc_sites",
	"azure_vnet_sites",
	"gcp_vpc_sites",
	"aws_tgw_sites",
	"securemesh_sites",
}
