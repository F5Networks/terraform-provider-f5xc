//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"testing"
)

var servicePolicyWrapper = unbindWrapper{
	call:            ServicePolicyUnbindReferencingObjects,
	logPrefix:       "ServicePolicy",
	parentSchema:    "views.http_loadbalancer",
	parentKind:      "http_loadbalancers",
	selfSchema:      "service_policy",
	selfKind:        "service_policys",
	selfIsFiltered:  false,
	containerKey:    "active_service_policies",
	listKey:         "policies",
	emptyChoiceKeys: []string{"no_service_policies"},
}

// TestServicePolicyUnbindReferencingObjects verifies the shared unbind contract
// for service_policy.
func TestServicePolicyUnbindReferencingObjects(t *testing.T) {
	runUnbindWrapperSuite(t, servicePolicyWrapper)
}

// TestServicePolicyUnbind_NoReferrerFilter verifies that service_policy has no
// referrer filter, so every parsed referrer kind is fetched.
func TestServicePolicyUnbind_NoReferrerFilter(t *testing.T) {
	accepted := map[string]string{
		"http_loadbalancers": "views.http_loadbalancer",
		"cdn_loadbalancers":  "views.cdn_loadbalancer",
		"tcp_loadbalancers":  "views.tcp_loadbalancer",
		"service_policys":    "service_policy",
		"virtual_networks":   "virtual_network",
	}

	checkUnbindReferrerKinds(t, servicePolicyWrapper, accepted, nil)
}
