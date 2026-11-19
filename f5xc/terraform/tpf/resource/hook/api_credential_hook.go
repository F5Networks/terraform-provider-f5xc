//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"gopkg.volterra.us/stdlib/codec"

	ves_io_schema_api_credential "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/api_credential"
)

// ApiCredentialOverrideUpdateHook executes api_credential update behavior that mirrors
// the legacy SDK provider logic.
//
// Why: api_credential has no general Replace/Update RPC and the update workflow is
// action-based. The legacy provider only performs backend actions when the credential:
//   - is of type API_TOKEN,
//   - is expired, and
//   - is inactive.
//
// How: the hook reads the credential using the backend-assigned credential name,
// checks type/expiry/active status, and conditionally calls Activate followed by Renew.
//
// Parameters:
//   - ctx: request-scoped context propagated from the resource Update() method.
//   - apiClient: CustomAPI-capable client used to call F5 XC endpoints.
//   - namespace: credential namespace (typically "system").
//   - credentialName: backend-assigned credential identifier (credential_name field).
//   - priorRenewTrigger: previous Terraform value of renew_trigger from state.
//   - plannedRenewTrigger: planned Terraform value of renew_trigger.
//   - expirationDays: requested expiration days from Terraform plan.
//
// Returns:
//   - nil when no action is required or all calls succeed.
//   - error when request marshaling fails, response types are unexpected, or any
//     required API call returns an error.
//
// Example:
//
//	err := ApiCredentialOverrideUpdateHook(ctx, r.client, "system", "cred-1", false, true, 30)
func ApiCredentialOverrideUpdateHook(
	ctx context.Context,
	apiClient CustomAPIClient,
	namespace string,
	credentialName string,
	priorRenewTrigger bool,
	plannedRenewTrigger bool,
	expirationDays int64,
) error {
	// renew_trigger is an edge-triggered Terraform knob. Only false->true should
	// request backend renewal. All other transitions are no-ops.
	if !plannedRenewTrigger || priorRenewTrigger {
		return nil
	}

	namespace = ResolveEffectiveNamespace(namespace, "")

	getReq := &ves_io_schema_api_credential.GetRequest{
		Name:      credentialName,
		Namespace: namespace,
	}

	yamlGetReq, err := codec.ToYAML(getReq)
	if err != nil {
		return fmt.Errorf("ApiCredentialOverrideUpdateHook: failed to marshal GetRequest: %w", err)
	}

	getRespMsg, err := apiClient.CustomAPI(
		ctx,
		"GET",
		fmt.Sprintf("/public/namespaces/%s/api_credentials/%s", namespace, credentialName),
		"ves.io.schema.api_credential.CustomAPI.Get",
		yamlGetReq,
	)
	if err != nil {
		return fmt.Errorf("ApiCredentialOverrideUpdateHook: failed to get api_credential %q: %w", credentialName, err)
	}

	getResp, ok := getRespMsg.(*ves_io_schema_api_credential.GetResponse)
	if !ok {
		return fmt.Errorf("ApiCredentialOverrideUpdateHook: unexpected Get response type %T", getRespMsg)
	}

	gcSpec := getResp.GetObject().GetSpec().GetGcSpec()
	if gcSpec == nil {
		return nil
	}

	if gcSpec.GetType() != ves_io_schema_api_credential.API_TOKEN {
		return nil
	}

	expireTS := gcSpec.GetExpirationTimestamp()
	if expireTS == nil {
		return nil
	}

	expireAt := time.Unix(expireTS.Seconds, int64(expireTS.Nanos)).UTC()
	if !time.Now().UTC().After(expireAt) {
		return nil
	}

	if gcSpec.GetActive() {
		return nil
	}

	_, err = apiClient.CustomAPI(
		ctx,
		"POST",
		fmt.Sprintf("/public/namespaces/%s/activate/api_credentials", namespace),
		"ves.io.schema.api_credential.CustomAPI.Activate",
		yamlGetReq,
	)
	if err != nil {
		return fmt.Errorf("ApiCredentialOverrideUpdateHook: failed to activate api_credential %q: %w", credentialName, err)
	}

	renewReq := &ves_io_schema_api_credential.RenewRequest{
		Name:      credentialName,
		Namespace: namespace,
	}
	if expirationDays > 0 {
		renewReq.ExpirationDays = uint32(expirationDays)
	}

	yamlRenewReq, err := codec.ToYAML(renewReq)
	if err != nil {
		return fmt.Errorf("ApiCredentialOverrideUpdateHook: failed to marshal RenewRequest: %w", err)
	}

	_, err = apiClient.CustomAPI(
		ctx,
		"POST",
		fmt.Sprintf("/public/namespaces/%s/renew/api_credentials", namespace),
		"ves.io.schema.api_credential.CustomAPI.Renew",
		yamlRenewReq,
	)
	if err != nil {
		return fmt.Errorf("ApiCredentialOverrideUpdateHook: failed to renew api_credential %q: %w", credentialName, err)
	}

	return nil
}

// ApiCredentialOverrideImportHook implements custom ImportState behavior for
// f5xc_api_credential, supporting both system-scoped and vK8s KUBE_CONFIG
// credentials.
//
// Why: Importing a vK8s KUBE_CONFIG credential requires routing the subsequent
// Read request to the vK8s namespace while keeping the top-level namespace
// attribute set to "system".
//
// How: Delegates to overrideCredentialImportHook, which parses req.ID via
// parseCredentialImportID, sets root "namespace" to
// "system" and "credential_name" to the parsed name, and when vk8sNamespace is
// non-empty populates "spec.virtual_k8s_namespace" in state so Read resolves the
// vK8s namespace.
//
// Parameters:
//   - ctx: request-scoped context from ImportState.
//   - req: Terraform ImportStateRequest containing the raw import ID.
//   - response: Terraform ImportStateResponse whose State and Diagnostics are updated in place.
//
// Returns:
//   - None. Populates response.State or appends diagnostics on invalid import ID.
//
// Example:
//
//	hook.ApiCredentialOverrideImportHook(ctx, req, response)
func ApiCredentialOverrideImportHook(ctx context.Context, req resource.ImportStateRequest, response *resource.ImportStateResponse) {
	overrideCredentialImportHook(ctx, req, response, "spec", []string{"virtual_k8s_namespace"})
}
