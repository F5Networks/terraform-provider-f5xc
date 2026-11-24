//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"testing"
)

var forwardProxyPolicyWrapper = unbindWrapper{
	call:            ForwardProxyPolicyUnbindReferencingObjects,
	logPrefix:       "ForwardProxyPolicy",
	parentSchema:    "views.http_loadbalancer",
	parentKind:      "http_loadbalancers",
	selfSchema:      "forward_proxy_policy",
	selfKind:        "forward_proxy_policys",
	selfIsFiltered:  true,
	containerKey:    "active_forward_proxy_policies",
	listKey:         "forward_proxy_policies",
	emptyChoiceKeys: []string{"no_forward_proxy_policy"},
}

// TestForwardProxyPolicyUnbindReferencingObjects verifies the shared unbind
// contract for forward_proxy_policy.
func TestForwardProxyPolicyUnbindReferencingObjects(t *testing.T) {
	runUnbindWrapperSuite(t, forwardProxyPolicyWrapper)
}

// TestForwardProxyPolicyUnbind_ReferrerKinds verifies that only origin pool,
// load balancer, and proxy parents are unbound. forward_proxy_policys is in the
// kind set but is excluded explicitly, because the 409 text echoes the policy itself.
func TestForwardProxyPolicyUnbind_ReferrerKinds(t *testing.T) {
	accepted := map[string]string{
		"origin_pools":       "views.origin_pool",
		"http_loadbalancers": "views.http_loadbalancer",
		"tcp_loadbalancers":  "views.tcp_loadbalancer",
		"proxys":             "views.proxy",
	}
	rejected := map[string]string{
		"forward_proxy_policys": "forward_proxy_policy",
		"virtual_networks":      "virtual_network",
		"aws_vpc_sites":         "views.aws_vpc_site",
	}

	checkUnbindReferrerKinds(t, forwardProxyPolicyWrapper, accepted, rejected)
}
