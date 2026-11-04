//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/client"
)

// isForwardProxyPolicyReferrerKind reports whether a referring-object kind is a parent a
// forward_proxy_policy can be bound to. It filters out the policy's own object type that
// the generic 409 parser also picks up from the "DeleteResource: ..." prefix.
func isForwardProxyPolicyReferrerKind(kind string) bool {
	return forwardProxyPolicyReferrerKinds.contains(kind) && kind != "forward_proxy_policys"
}

// ForwardProxyPolicyUnbindReferencingObjects parses an F5XC 409 conflict raised when
// deleting a forward_proxy_policy, finds any referring objects (origin pools, HTTP/TCP
// load balancers, proxies), removes the policy from their
// active_forward_proxy_policies.forward_proxy_policies list (switching to
// no_forward_proxy_policy when the list empties), and PUTs them back so the policy can be
// deleted. See policyUnbindReferencingObjects for the full contract and XC-3073 rationale.
func ForwardProxyPolicyUnbindReferencingObjects(
	ctx context.Context,
	apiClient *client.APIClient,
	policyNamespace, policyName, errText string,
) error {
	return policyUnbindReferencingObjects(ctx, apiClient, policyNamespace, policyName, errText, policyUnbindSpec{
		logPrefix:       "ForwardProxyPolicy",
		containerKey:    "active_forward_proxy_policies",
		listKey:         "forward_proxy_policies",
		emptyChoiceKeys: []string{"no_forward_proxy_policy"},
		isReferrerKind:  isForwardProxyPolicyReferrerKind,
	})
}
