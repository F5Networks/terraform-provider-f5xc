//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/client"
)

// ServicePolicyUnbindReferencingObjects removes a service policy from inline parent specs.
func ServicePolicyUnbindReferencingObjects(ctx context.Context, apiClient *client.APIClient, policyNamespace, policyName, errText string) error {
	return policyUnbindReferencingObjects(ctx, apiClient, policyNamespace, policyName, errText, policyUnbindSpec{
		logPrefix:       "ServicePolicy",
		containerKey:    "active_service_policies",
		listKey:         "policies",
		emptyChoiceKeys: []string{"no_service_policies"},
	})
}
