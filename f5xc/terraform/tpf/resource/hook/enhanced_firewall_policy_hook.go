//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/client"
)

// isEnhancedFirewallPolicyReferrerKind reports whether a referring-object kind is a parent
// an enhanced_firewall_policy can be bound to. It filters out the policy's own object type
// ("enhanced_firewall_policys") that the generic 409 parser also picks up from the
// "DeleteResource: ..." prefix.
func isEnhancedFirewallPolicyReferrerKind(kind string) bool {
	return enhancedFirewallPolicyReferrerKinds.contains(kind)
}

// EnhancedFirewallPolicyUnbindReferencingObjects parses an F5XC 409 conflict raised when
// deleting an enhanced_firewall_policy, finds any referring site objects, removes the
// policy from their active_enhanced_firewall_policies.enhanced_firewall_policies list
// (switching to no_firewall_policy / no_enhanced_firewall_policy when the list empties),
// and PUTs them back so the policy can be deleted. See policyUnbindReferencingObjects for
// the full contract and XC-3073 rationale.
func EnhancedFirewallPolicyUnbindReferencingObjects(
	ctx context.Context,
	apiClient *client.APIClient,
	policyNamespace, policyName, errText string,
) error {
	return policyUnbindReferencingObjects(ctx, apiClient, policyNamespace, policyName, errText, policyUnbindSpec{
		logPrefix:       "EnhancedFirewallPolicy",
		containerKey:    "active_enhanced_firewall_policies",
		listKey:         "enhanced_firewall_policies",
		emptyChoiceKeys: []string{"no_firewall_policy", "no_enhanced_firewall_policy"},
		isReferrerKind:  isEnhancedFirewallPolicyReferrerKind,
	})
}
