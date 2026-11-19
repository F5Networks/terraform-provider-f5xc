//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const (
	defaultCredentialNamespace  = "system"
	apiCredentialTypeKubeConfig = "KUBE_CONFIG"
)

// ResolveEffectiveNamespace determines the backend namespace where an API or
// service credential is stored for post-create operations (Read, Update, Revoke).
//
// Why: starting in XC 10.0, vK8s KubeConfig credentials are created via the
// "system" namespace endpoint, but the backend persists the credential object
// under the target vK8s namespace for RBAC enforcement. Non-vK8s credentials
// remain in the configured namespace (defaulting to "system").
//
// How: trims both inputs and prefers vk8sNamespace when non-empty, falling back
// to configuredNamespace and finally "system".
//
// Parameters:
//   - configuredNamespace: root Terraform namespace value from plan/state.
//   - vk8sNamespace: vK8s namespace extracted from the credential spec/choice block.
//
// Returns:
//   - string: effective backend namespace for Read/Update/Delete operations.
//
// Example:
//
//	ns := ResolveEffectiveNamespace("system", "my-vk8s-ns")
//	// ns == "my-vk8s-ns"
func ResolveEffectiveNamespace(configuredNamespace, vk8sNamespace string) string {
	if trimmedVK8s := strings.TrimSpace(vk8sNamespace); trimmedVK8s != "" {
		return trimmedVK8s
	}
	if trimmedConfigured := strings.TrimSpace(configuredNamespace); trimmedConfigured != "" {
		return trimmedConfigured
	}
	return defaultCredentialNamespace
}

// ResolveApiCredentialNamespace resolves the effective backend namespace for an
// f5xc_api_credential resource from its root namespace and spec block.
//
// Why: f5xc_api_credential configures vK8s KubeConfig parameters under
// spec.type = ["KUBE_CONFIG"] and spec.virtual_k8s_namespace, while the root
// namespace attribute is constrained to "system".
//
// How: inspects specObj; when spec.type is explicitly set to a non-KUBE_CONFIG
// type, ignores virtual_k8s_namespace. Otherwise extracts virtual_k8s_namespace
// and delegates to ResolveEffectiveNamespace.
//
// Parameters:
//   - ctx: request-scoped context.
//   - configuredNamespace: root Terraform namespace attribute.
//   - specObj: Terraform spec SingleNestedAttribute object.
//
// Returns:
//   - string: effective backend namespace for Read/Update/Delete operations.
//
// Example:
//
//	ns := ResolveApiCredentialNamespace(ctx, data.Namespace, data.Spec)
//	// ns == "my-vk8s-ns" when data.Spec has virtual_k8s_namespace = "my-vk8s-ns"
func ResolveApiCredentialNamespace(ctx context.Context, configuredNamespace types.String, specObj types.Object) string {
	_ = ctx

	var cfgNS string
	if !configuredNamespace.IsNull() && !configuredNamespace.IsUnknown() {
		cfgNS = configuredNamespace.ValueString()
	}

	if specObj.IsNull() || specObj.IsUnknown() {
		return ResolveEffectiveNamespace(cfgNS, "")
	}

	attrs := specObj.Attributes()
	if typeAttr, hasType := attrs["type"]; hasType {
		if typeList, ok := typeAttr.(types.List); ok && !typeList.IsNull() && !typeList.IsUnknown() && len(typeList.Elements()) > 0 {
			isKubeConfig := false
			hasKnownElem := false
			for _, elem := range typeList.Elements() {
				strElem, ok := elem.(types.String)
				if !ok || strElem.IsNull() || strElem.IsUnknown() {
					continue
				}
				hasKnownElem = true
				if strings.TrimSpace(strElem.ValueString()) == apiCredentialTypeKubeConfig {
					isKubeConfig = true
					break
				}
			}
			if hasKnownElem && !isKubeConfig {
				return ResolveEffectiveNamespace(cfgNS, "")
			}
		}
	}

	var vk8sNS string
	if nsAttr, ok := attrs["virtual_k8s_namespace"].(types.String); ok && !nsAttr.IsNull() && !nsAttr.IsUnknown() {
		vk8sNS = nsAttr.ValueString()
	}

	return ResolveEffectiveNamespace(cfgNS, vk8sNS)
}

// ResolveServiceCredentialNamespace resolves the effective backend namespace for
// an f5xc_service_credential resource from its root namespace and
// service_credential_choice block.
//
// Why: f5xc_service_credential configures vK8s KubeConfig parameters under
// service_credential_choice.vk8s_kubeconfig.vk8s_namespace, while the root
// namespace attribute is constrained to "system".
//
// How: inspects choiceObj.vk8s_kubeconfig.vk8s_namespace when present and known,
// and delegates to ResolveEffectiveNamespace.
//
// Parameters:
//   - ctx: request-scoped context.
//   - configuredNamespace: root Terraform namespace attribute.
//   - choiceObj: Terraform service_credential_choice SingleNestedAttribute object.
//
// Returns:
//   - string: effective backend namespace for Read/Update/Delete operations.
//
// Example:
//
//	ns := ResolveServiceCredentialNamespace(ctx, data.Namespace, data.ServiceCredentialChoice)
//	// ns == "my-vk8s-ns" when vk8s_kubeconfig.vk8s_namespace is "my-vk8s-ns"
func ResolveServiceCredentialNamespace(ctx context.Context, configuredNamespace types.String, choiceObj types.Object) string {
	_ = ctx

	var cfgNS string
	if !configuredNamespace.IsNull() && !configuredNamespace.IsUnknown() {
		cfgNS = configuredNamespace.ValueString()
	}

	if choiceObj.IsNull() || choiceObj.IsUnknown() {
		return ResolveEffectiveNamespace(cfgNS, "")
	}

	vk8sAttr, hasVK8s := choiceObj.Attributes()["vk8s_kubeconfig"]
	if !hasVK8s {
		return ResolveEffectiveNamespace(cfgNS, "")
	}

	vk8sObj, ok := vk8sAttr.(types.Object)
	if !ok || vk8sObj.IsNull() || vk8sObj.IsUnknown() {
		return ResolveEffectiveNamespace(cfgNS, "")
	}

	var vk8sNS string
	if nsAttr, ok := vk8sObj.Attributes()["vk8s_namespace"].(types.String); ok && !nsAttr.IsNull() && !nsAttr.IsUnknown() {
		vk8sNS = nsAttr.ValueString()
	}

	return ResolveEffectiveNamespace(cfgNS, vk8sNS)
}

// parseCredentialImportID parses the Terraform import ID for f5xc_api_credential
// and f5xc_service_credential.
//
// Why: In F5 XC 10.0+, vK8s kubeconfig credentials are stored in the target vK8s
// namespace while the Terraform resource's top-level namespace attribute must
// remain "system". Import IDs therefore need to carry the vK8s namespace without
// clobbering the top-level "system" namespace in state.
//
// How: Splits id on "/" and trims surrounding whitespace from every segment, so
// whitespace-only segments are treated as empty. For 2-part IDs
// ("<ns>/<credential_name>"), rejects empty segments; if the first segment is
// "system", returns ("system", "", parts[1], nil), otherwise treats parts[0] as
// vk8sNamespace and returns ("system", parts[0], parts[1], nil).
// For 3-part IDs ("system/<vk8s_namespace>/<credential_name>"), validates that no
// segment is empty and the first segment is "system", returning ("system", parts[1], parts[2], nil).
//
// Parameters:
//   - id: raw import identifier passed to terraform import.
//
// Returns:
//   - namespace: root Terraform namespace attribute value (always "system" on success).
//   - vk8sNamespace: target vK8s namespace when importing a vK8s credential, or "" for system credentials.
//   - credentialName: backend credential name.
//   - err: non-nil when id does not match a valid 2-part or 3-part format or contains empty segments.
//
// Example:
//
//	ns, vk8sNS, name, err := parseCredentialImportID("my-vk8s-ns/cred-123")
//	// ns == "system", vk8sNS == "my-vk8s-ns", name == "cred-123", err == nil
func parseCredentialImportID(id string) (namespace, vk8sNamespace, credentialName string, err error) {
	parts := strings.Split(id, "/")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	switch len(parts) {
	case 2:
		if parts[0] == "" {
			return "", "", "", fmt.Errorf("expected import ID format: namespace/credential_name or system/vk8s_namespace/credential_name, got: %s (namespace is empty)", id)
		}
		if parts[1] == "" {
			return "", "", "", fmt.Errorf("expected import ID format: namespace/credential_name or system/vk8s_namespace/credential_name, got: %s (credential_name is empty)", id)
		}
		if parts[0] == defaultCredentialNamespace {
			return defaultCredentialNamespace, "", parts[1], nil
		}
		return defaultCredentialNamespace, parts[0], parts[1], nil
	case 3:
		if parts[0] == "" || parts[1] == "" || parts[2] == "" {
			return "", "", "", fmt.Errorf("expected import ID format: system/vk8s_namespace/credential_name, got: %s (contains empty segment)", id)
		}
		if parts[0] != defaultCredentialNamespace {
			return "", "", "", fmt.Errorf("expected import ID format: system/vk8s_namespace/credential_name, got: %s (first segment must be %q)", id, defaultCredentialNamespace)
		}
		return defaultCredentialNamespace, parts[1], parts[2], nil
	default:
		return "", "", "", fmt.Errorf("expected import ID format: namespace/credential_name or system/vk8s_namespace/credential_name, got: %s", id)
	}
}

// overrideCredentialImportHook is the shared ImportState implementation for
// f5xc_api_credential and f5xc_service_credential.
//
// Why: both resources accept the same import ID formats and differ only in where
// the vK8s namespace lives in their schema, so one implementation keeps the two
// import paths consistent.
//
// How: parses req.ID via parseCredentialImportID, sets root "namespace" and
// "credential_name", and when a vK8s namespace is present seeds it into rootAttr
// at leafPath using nestedObjectWithString (types derived from the live schema).
//
// Parameters:
//   - ctx: request-scoped context from ImportState.
//   - req: Terraform ImportStateRequest containing the raw import ID.
//   - response: Terraform ImportStateResponse whose State and Diagnostics are updated in place.
//   - rootAttr: root object attribute that holds the vK8s namespace (for example, "spec").
//   - leafPath: nested attribute names from rootAttr to the vK8s namespace string.
//
// Returns:
//   - None. Populates response.State or appends diagnostics on invalid import ID.
//
// Example:
//
//	overrideCredentialImportHook(ctx, req, response, "spec", []string{"virtual_k8s_namespace"})
func overrideCredentialImportHook(ctx context.Context, req resource.ImportStateRequest, response *resource.ImportStateResponse, rootAttr string, leafPath []string) {
	namespace, vk8sNamespace, credentialName, err := parseCredentialImportID(req.ID)
	if err != nil {
		response.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("namespace"), namespace)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("credential_name"), credentialName)...)
	if response.Diagnostics.HasError() || vk8sNamespace == "" {
		return
	}

	obj, diags := nestedObjectWithString(ctx, response.State.Schema, rootAttr, leafPath, vk8sNamespace)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root(rootAttr), obj)...)
}

// schemaTypeAtPath is the subset of the framework schema used by import hooks. It is
// satisfied by tfsdk.State.Schema (the framework's internal fwschema.Schema), which
// cannot be imported directly.
type schemaTypeAtPath interface {
	TypeAtPath(ctx context.Context, p path.Path) (attr.Type, diag.Diagnostics)
}

// nestedObjectWithString builds a value for a root object attribute in which every
// attribute is null except one nested string leaf.
//
// Why: import hooks need to seed a single nested value (for example, the vK8s
// namespace) into state. Hard-coding the attribute type maps duplicates the generated
// schema and fails at runtime with a type mismatch when the schema gains or renames a
// nested attribute. Deriving the types from the live schema keeps both in sync.
//
// How: resolves the root attribute's object type from s, then walks leafPath, building
// null objects whose types come from the schema and setting the final element to a
// string value.
//
// Parameters:
//   - ctx: request-scoped context.
//   - s: resource schema (typically response.State.Schema).
//   - rootAttr: root attribute name (for example, "spec").
//   - leafPath: nested attribute names from rootAttr to the string leaf
//     (for example, []string{"vk8s_kubeconfig", "vk8s_namespace"}).
//   - value: string value to place at the leaf.
//
// Returns:
//   - types.Object: object value for rootAttr with only the leaf set.
//   - diag.Diagnostics: errors when an attribute is missing or not an object/string.
//
// Example:
//
//	obj, diags := nestedObjectWithString(ctx, resp.State.Schema, "spec", []string{"virtual_k8s_namespace"}, "my-ns")
func nestedObjectWithString(ctx context.Context, s schemaTypeAtPath, rootAttr string, leafPath []string, value string) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	if s == nil {
		diags.AddError("Import State Error", "resource schema is not available")
		return types.ObjectNull(nil), diags
	}
	rootType, typeDiags := s.TypeAtPath(ctx, path.Root(rootAttr))
	diags.Append(typeDiags...)
	if diags.HasError() {
		return types.ObjectNull(nil), diags
	}
	objType, ok := rootType.(types.ObjectType)
	if !ok {
		diags.AddError("Import State Error", fmt.Sprintf("attribute %q is not an object", rootAttr))
		return types.ObjectNull(nil), diags
	}
	return buildNestedObject(objType, rootAttr, leafPath, value)
}

// buildNestedObject recursively builds objType with all attributes null except the
// string leaf addressed by leafPath. See nestedObjectWithString for details.
//
// Parameters:
//   - objType: object type of the current level, taken from the schema.
//   - where: dotted attribute path of the current level, used in error messages.
//   - leafPath: remaining attribute names to the string leaf; must be non-empty.
//   - value: string value to place at the leaf.
//
// Returns:
//   - types.Object: object value of objType with only the leaf set.
//   - diag.Diagnostics: errors when an attribute is missing or has an unexpected type.
func buildNestedObject(objType types.ObjectType, where string, leafPath []string, value string) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(leafPath) == 0 {
		diags.AddError("Import State Error", fmt.Sprintf("no nested attribute path given for %q", where))
		return types.ObjectNull(objType.AttrTypes), diags
	}
	name := leafPath[0]
	childType, ok := objType.AttrTypes[name]
	if !ok {
		diags.AddError("Import State Error", fmt.Sprintf("attribute %q not found in %q", name, where))
		return types.ObjectNull(objType.AttrTypes), diags
	}

	vals := make(map[string]attr.Value, len(objType.AttrTypes))
	for attrName, attrType := range objType.AttrTypes {
		nullVal, err := nullValueOf(attrType)
		if err != nil {
			diags.AddError("Import State Error", fmt.Sprintf("attribute %q in %q: %s", attrName, where, err))
			return types.ObjectNull(objType.AttrTypes), diags
		}
		vals[attrName] = nullVal
	}

	if len(leafPath) == 1 {
		if childType != types.StringType {
			diags.AddError("Import State Error", fmt.Sprintf("attribute %q in %q is not a string", name, where))
			return types.ObjectNull(objType.AttrTypes), diags
		}
		vals[name] = types.StringValue(value)
	} else {
		childObjType, isObj := childType.(types.ObjectType)
		if !isObj {
			diags.AddError("Import State Error", fmt.Sprintf("attribute %q in %q is not an object", name, where))
			return types.ObjectNull(objType.AttrTypes), diags
		}
		childObj, childDiags := buildNestedObject(childObjType, where+"."+name, leafPath[1:], value)
		diags.Append(childDiags...)
		if diags.HasError() {
			return types.ObjectNull(objType.AttrTypes), diags
		}
		vals[name] = childObj
	}

	obj, objDiags := types.ObjectValue(objType.AttrTypes, vals)
	diags.Append(objDiags...)
	return obj, diags
}

// nullValueOf returns a typed null value for t, used to fill attributes that an import
// hook does not set. Unknown attribute kinds fall back to the type's ValueFromTerraform
// with a null Terraform value.
//
// Parameters:
//   - t: attribute type taken from the schema.
//
// Returns:
//   - attr.Value: null value of type t.
//   - error: non-nil when t is nil or the fallback cannot produce a null value.
func nullValueOf(t attr.Type) (attr.Value, error) {
	if t == nil {
		return nil, fmt.Errorf("attribute type is nil")
	}
	switch typed := t.(type) {
	case types.ObjectType:
		return types.ObjectNull(typed.AttrTypes), nil
	case types.ListType:
		return types.ListNull(typed.ElemType), nil
	case types.SetType:
		return types.SetNull(typed.ElemType), nil
	case types.MapType:
		return types.MapNull(typed.ElemType), nil
	}
	switch t {
	case types.StringType:
		return types.StringNull(), nil
	case types.BoolType:
		return types.BoolNull(), nil
	case types.Int64Type:
		return types.Int64Null(), nil
	case types.Float64Type:
		return types.Float64Null(), nil
	case types.NumberType:
		return types.NumberNull(), nil
	}
	ctx := context.Background()
	v, err := t.ValueFromTerraform(ctx, tftypes.NewValue(t.TerraformType(ctx), nil))
	if err != nil {
		return nil, fmt.Errorf("building null value for %s: %w", t, err)
	}
	if v == nil {
		return nil, fmt.Errorf("building null value for %s: no value returned", t)
	}
	return v, nil
}
