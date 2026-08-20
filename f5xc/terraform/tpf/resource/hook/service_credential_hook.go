//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"gopkg.volterra.us/stdlib/codec"

	ves_io_schema "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema"
	ves_io_schema_api_credential "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/api_credential"
)

type serviceCredentialNamespaceRoleModel struct {
	Namespace types.String `tfsdk:"namespace"`
	Role      types.String `tfsdk:"role"`
}

// ServiceCredentialPreUpdateHook executes pre-update renew behavior for
// service_credential resources.
//
// Why: service_credential keeps standard Replace behavior for mutable fields,
// while renewal remains an action-driven flow. The hook is invoked before Update
// and performs renewal only on renew_trigger false->true transitions.
//
// How: the hook reads the service credential, checks token type/expiry/activity,
// then conditionally calls ActivateServiceCredentials followed by
// RenewServiceCredentials.
//
// Parameters:
//   - ctx: request-scoped context propagated from resource Update().
//   - apiClient: CustomAPI-capable client used to call F5 XC endpoints.
//   - req: in-flight ReplaceServiceCredentialsRequest that will be sent by the
//     generated Update() path.
//   - namespaceRoles: Terraform plan value for namespace_roles.
//   - namespace: credential namespace.
//   - credentialName: backend-assigned credential identifier.
//   - priorRenewTrigger: previous Terraform value of renew_trigger from state.
//   - plannedRenewTrigger: planned Terraform value of renew_trigger.
//   - expirationDays: requested expiration days from Terraform plan.
//
// Returns:
//   - nil when no action is required or all calls succeed.
//   - error when request marshaling fails, response types are unexpected, or a
//     required API call returns an error.
func ServiceCredentialPreUpdateHook(
	ctx context.Context,
	apiClient CustomAPIClient,
	req *ves_io_schema_api_credential.ReplaceServiceCredentialsRequest,
	namespaceRoles types.List,
	namespace string,
	credentialName string,
	priorRenewTrigger bool,
	plannedRenewTrigger bool,
	expirationDays int64,
) error {
	if req == nil {
		return fmt.Errorf("ServiceCredentialPreUpdateHook: replace request is nil")
	}

	namespaceAccess, hasNamespaceAccess, err := buildNamespaceAccessFromRoles(ctx, namespaceRoles)
	if err != nil {
		return err
	}
	if hasNamespaceAccess {
		req.NamespaceAccess = namespaceAccess
	}

	if !plannedRenewTrigger || priorRenewTrigger {
		return nil
	}

	getReq := &ves_io_schema_api_credential.GetRequest{
		Name:      credentialName,
		Namespace: namespace,
	}

	yamlGetReq, err := codec.ToYAML(getReq)
	if err != nil {
		return fmt.Errorf("ServiceCredentialPreUpdateHook: failed to marshal GetRequest: %w", err)
	}

	getRespMsg, err := apiClient.CustomAPI(
		ctx,
		"GET",
		fmt.Sprintf("/public/namespaces/%s/service_credentials/%s", namespace, credentialName),
		"ves.io.schema.api_credential.CustomAPI.GetServiceCredentials",
		yamlGetReq,
	)
	if err != nil {
		return fmt.Errorf("ServiceCredentialPreUpdateHook: failed to get service_credential %q: %w", credentialName, err)
	}

	getResp, ok := getRespMsg.(*ves_io_schema_api_credential.GetServiceCredentialsResponse)
	if !ok {
		return fmt.Errorf("ServiceCredentialPreUpdateHook: unexpected GetServiceCredentials response type %T", getRespMsg)
	}

	if getResp.GetType() != ves_io_schema_api_credential.SERVICE_API_TOKEN {
		return nil
	}

	expireTS := getResp.GetExpiryTimestamp()
	if expireTS == nil {
		return nil
	}

	expireAt := time.Unix(expireTS.Seconds, int64(expireTS.Nanos)).UTC()
	if !time.Now().UTC().After(expireAt) {
		return nil
	}

	if getResp.GetActive() {
		return nil
	}

	_, err = apiClient.CustomAPI(
		ctx,
		"POST",
		fmt.Sprintf("/public/namespaces/%s/activate/service_credentials", namespace),
		"ves.io.schema.api_credential.CustomAPI.ActivateServiceCredentials",
		yamlGetReq,
	)
	if err != nil {
		return fmt.Errorf("ServiceCredentialPreUpdateHook: failed to activate service_credential %q: %w", credentialName, err)
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
		return fmt.Errorf("ServiceCredentialPreUpdateHook: failed to marshal RenewRequest: %w", err)
	}

	_, err = apiClient.CustomAPI(
		ctx,
		"POST",
		fmt.Sprintf("/public/namespaces/%s/renew/service_credentials", namespace),
		"ves.io.schema.api_credential.CustomAPI.RenewServiceCredentials",
		yamlRenewReq,
	)
	if err != nil {
		return fmt.Errorf("ServiceCredentialPreUpdateHook: failed to renew service_credential %q: %w", credentialName, err)
	}

	return nil
}

// buildNamespaceAccessFromRoles converts Terraform namespace_roles into the
// ReplaceServiceCredentialsRequest namespace_access payload.
//
// Why: CreateServiceCredentialsRequest uses namespace_roles, while
// ReplaceServiceCredentialsRequest expects namespace_access.namespace_role_map.
//
// How: each role entry {namespace, role} is grouped by namespace and appended to
// RoleListType.Names, with duplicate roles removed per namespace.
//
// Parameters:
//   - ctx: request-scoped context for Terraform value conversion.
//   - namespaceRoles: Terraform plan list for namespace_roles.
//
// Returns:
//   - *ves_io_schema.NamespaceAccessType: converted namespace access payload.
//   - bool: true when namespace_roles is known (including explicit empty list).
//   - error: non-nil if Terraform list decoding fails.
func buildNamespaceAccessFromRoles(ctx context.Context, namespaceRoles types.List) (*ves_io_schema.NamespaceAccessType, bool, error) {
	if namespaceRoles.IsNull() || namespaceRoles.IsUnknown() {
		return nil, false, nil
	}

	var elements []serviceCredentialNamespaceRoleModel
	if diags := namespaceRoles.ElementsAs(ctx, &elements, false); diags.HasError() {
		return nil, false, fmt.Errorf("ServiceCredentialPreUpdateHook: failed to decode namespace_roles: %s", diags.Errors()[0].Detail())
	}

	namespaceRoleMap := make(map[string]*ves_io_schema.RoleListType)
	seenRoles := make(map[string]map[string]struct{})

	for _, element := range elements {
		if element.Namespace.IsNull() || element.Namespace.IsUnknown() || element.Role.IsNull() || element.Role.IsUnknown() {
			continue
		}

		namespace := strings.TrimSpace(element.Namespace.ValueString())
		role := strings.TrimSpace(element.Role.ValueString())
		if namespace == "" || role == "" {
			continue
		}

		if _, ok := namespaceRoleMap[namespace]; !ok {
			namespaceRoleMap[namespace] = &ves_io_schema.RoleListType{Names: []string{}}
		}
		if _, ok := seenRoles[namespace]; !ok {
			seenRoles[namespace] = make(map[string]struct{})
		}
		if _, exists := seenRoles[namespace][role]; exists {
			continue
		}
		seenRoles[namespace][role] = struct{}{}
		namespaceRoleMap[namespace].Names = append(namespaceRoleMap[namespace].Names, role)
	}

	return &ves_io_schema.NamespaceAccessType{NamespaceRoleMap: namespaceRoleMap}, true, nil
}
