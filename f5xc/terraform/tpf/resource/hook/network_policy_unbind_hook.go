//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/client"
)

// isNetworkPolicyReferrerKind reports whether a referring-object kind is a parent a
// network_policy can be bound to. It filters out the policy's own object type
// ("network_policys") that the generic 409 parser also picks up from the
// "DeleteResource: ..." prefix.
func isNetworkPolicyReferrerKind(kind string) bool {
	return networkPolicyReferrerKinds.contains(kind)
}

// NetworkPolicyUnbindReferencingObjects parses an F5XC 409 conflict raised when deleting a
// network_policy, finds any referring objects (virtual networks, network policy sets,
// sites), removes the policy from their active_network_policies.network_policies list
// (switching to no_network_policy when the list empties), and PUTs them back so the policy
// can be deleted. See policyUnbindReferencingObjects for the full contract and XC-3073
// rationale.
func NetworkPolicyUnbindReferencingObjects(
	ctx context.Context,
	apiClient *client.APIClient,
	policyNamespace, policyName, errText string,
) error {
	return policyUnbindReferencingObjects(ctx, apiClient, policyNamespace, policyName, errText, policyUnbindSpec{
		logPrefix:       "NetworkPolicy",
		containerKey:    "active_network_policies",
		listKey:         "network_policies",
		emptyChoiceKeys: []string{"no_network_policy"},
		isReferrerKind:  isNetworkPolicyReferrerKind,
	})
}
