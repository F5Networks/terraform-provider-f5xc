//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package requiredifvalidator

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const sharedNamespace = "shared"

// Ensure CrossNamespaceSharedTenantValidator implements both validator.String and planmodifier.String.
var (
	_ validator.String    = CrossNamespaceSharedTenantValidator{}
	_ planmodifier.String = CrossNamespaceSharedTenantValidator{}
)

// attributeGetter abstracts tfsdk.Config and tfsdk.Plan attribute lookup so the
// same validation logic runs during both ValidateResourceConfig and PlanResourceChange.
type attributeGetter interface {
	GetAttribute(ctx context.Context, target path.Path, val any) diag.Diagnostics
}

// CrossNamespaceSharedTenantValidator enforces that ObjectRefType.tenant is
// explicitly set whenever a resource outside the "shared" namespace references
// an object in the "shared" namespace.
//
// Why: When a resource resides in a non-shared namespace (e.g. "app-ns") and
// references a shared object (such as virtual_network "shared/public"), the
// F5 XC backend cannot infer the owning tenant ("ves-io" vs. customer tenant)
// and rejects the request during apply with HTTP 400 ("Bad reference values").
// By implementing both validator.String (Phase 1: ValidateResourceConfig) and
// planmodifier.String (Phase 2: PlanResourceChange), this type catches the
// missing tenant during terraform validate/plan — even when root.namespace is
// a dynamic reference (e.g. f5xc_namespace.example.name) that only resolves
// during PlanResourceChange.
type CrossNamespaceSharedTenantValidator struct{}

// RequireTenantForCrossNamespaceShared returns a CrossNamespaceSharedTenantValidator
// that can be registered in both Validators and PlanModifiers on ObjectRefType.tenant.
//
// Why: Provides a single constructor for generated TPF resource schemas to attach
// dual-phase (validate + plan) cross-namespace shared reference checking.
//
// How: Instantiates and returns a stateless CrossNamespaceSharedTenantValidator value.
//
// Parameters:
//   - None.
//
// Returns:
//   - CrossNamespaceSharedTenantValidator: Instance implementing validator.String and planmodifier.String.
//
// Example:
//
//	v := requiredifvalidator.RequireTenantForCrossNamespaceShared()
func RequireTenantForCrossNamespaceShared() CrossNamespaceSharedTenantValidator {
	return CrossNamespaceSharedTenantValidator{}
}

// Description returns a plain-text description of the validator and plan modifier behavior.
//
// Why: Required by the validator.String and planmodifier.String interfaces for
// diagnostics and documentation generation.
//
// How: Returns a static descriptive string explaining the cross-namespace shared tenant rule.
//
// Parameters:
//   - ctx: context.Context (unused).
//
// Returns:
//   - string: Human-readable description of the rule.
//
// Example:
//
//	desc := v.Description(ctx)
func (v CrossNamespaceSharedTenantValidator) Description(_ context.Context) string {
	return "Requires 'tenant' to be set when referencing an object in the 'shared' namespace from a non-shared resource namespace (e.g., set tenant = \"ves-io\" for F5 XC system-shared objects)."
}

// MarkdownDescription returns a Markdown-formatted description of the validator and plan modifier behavior.
//
// Why: Required by the validator.String and planmodifier.String interfaces for
// provider documentation generation.
//
// How: Returns a static Markdown string explaining the cross-namespace shared tenant rule.
//
// Parameters:
//   - ctx: context.Context (unused).
//
// Returns:
//   - string: Markdown-formatted description of the rule.
//
// Example:
//
//	md := v.MarkdownDescription(ctx)
func (v CrossNamespaceSharedTenantValidator) MarkdownDescription(_ context.Context) string {
	return "Requires `tenant` to be set when referencing an object in the `shared` namespace from a non-shared resource namespace (e.g., set `tenant = \"ves-io\"` for F5 XC system-shared objects)."
}

// ValidateString executes Phase 1 validation during ValidateResourceConfig.
//
// Why: Catches missing tenant configurations immediately during 'terraform validate'
// and Phase 1 of 'terraform plan' whenever both the sibling reference namespace and
// the root resource namespace are statically known in HCL.
//
// How: Delegates to checkCrossNamespaceSharedTenant using req.ConfigValue and req.Config.
// If root.namespace is still Unknown (e.g., referencing another resource), it returns
// without error so Phase 2 (PlanModifyString) can evaluate the resolved plan value.
//
// Parameters:
//   - ctx:  Request context.
//   - req:  validator.StringRequest containing the attribute path, config value, and full config.
//   - resp: *validator.StringResponse to which attribute error diagnostics are appended.
//
// Returns:
//   - None (diagnostics are appended to resp.Diagnostics).
//
// Example:
//
//	v.ValidateString(ctx, req, resp)
func (v CrossNamespaceSharedTenantValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	diags := checkCrossNamespaceSharedTenant(ctx, req.Path, req.ConfigValue, req.Config)
	resp.Diagnostics.Append(diags...)
}

// PlanModifyString executes Phase 2 validation during PlanResourceChange.
//
// Why: When root.namespace references another resource (e.g. f5xc_namespace.this.name),
// its value is Unknown in req.Config during ValidateResourceConfig (Phase 1), but
// resolves to a known string in req.Plan during PlanResourceChange (Phase 2) if the
// referenced namespace already exists or its name is known at plan time.
//
// How: Skips destroy plans (req.Plan.Raw.IsNull()). Otherwise, delegates to
// checkCrossNamespaceSharedTenant using req.ConfigValue (which remains Null when
// omitted by the user, even though req.PlanValue is Unknown for Optional+Computed
// attributes) and req.Plan (which holds the resolved root.namespace).
//
// Parameters:
//   - ctx:  Request context.
//   - req:  planmodifier.StringRequest containing the attribute path, config value, and resolved plan.
//   - resp: *planmodifier.StringResponse to which attribute error diagnostics are appended.
//
// Returns:
//   - None (diagnostics are appended to resp.Diagnostics).
//
// Example:
//
//	v.PlanModifyString(ctx, req, resp)
func (v CrossNamespaceSharedTenantValidator) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	diags := checkCrossNamespaceSharedTenant(ctx, req.Path, req.ConfigValue, req.Plan)
	resp.Diagnostics.Append(diags...)
}

// checkCrossNamespaceSharedTenant evaluates whether tenant is missing on a cross-namespace reference to "shared".
//
// Why: Centralizes the diagnostic check shared between ValidateString (Phase 1) and PlanModifyString (Phase 2).
//
// How:
//  1. If tenantConfigVal is Unknown or already populated with a non-empty string, returns nil.
//  2. Reads the sibling "namespace" attribute from the enclosing ObjectRefType container.
//     If sibling namespace is Null, Unknown, or not equal to "shared", returns nil.
//  3. Reads the resource-level root "namespace" attribute.
//     If root namespace is Null, Unknown, or equal to "shared", returns nil.
//  4. Otherwise (root.namespace != "shared", sibling.namespace == "shared", and tenant is omitted/empty),
//     returns an AttributeErrorDiagnostic explaining that tenant must be specified.
//
// Parameters:
//   - ctx:             Request context.
//   - attrPath:        Path of the tenant attribute being validated.
//   - tenantConfigVal: User-configured value for tenant from req.ConfigValue.
//   - getter:          Config or Plan attribute reader (req.Config or req.Plan).
//
// Returns:
//   - diag.Diagnostics: Error diagnostic if the rule is violated; nil otherwise.
//
// Example:
//
//	diags := checkCrossNamespaceSharedTenant(ctx, req.Path, req.ConfigValue, req.Plan)
func checkCrossNamespaceSharedTenant(
	ctx context.Context,
	attrPath path.Path,
	tenantConfigVal types.String,
	getter attributeGetter,
) diag.Diagnostics {
	if tenantConfigVal.IsUnknown() {
		return nil
	}
	if !tenantConfigVal.IsNull() && strings.TrimSpace(tenantConfigVal.ValueString()) != "" {
		return nil
	}

	siblingNSPath := attrPath.ParentPath().AtName("namespace")
	var siblingNS types.String
	if diags := getter.GetAttribute(ctx, siblingNSPath, &siblingNS); diags.HasError() {
		return nil
	}
	if siblingNS.IsNull() || siblingNS.IsUnknown() || siblingNS.ValueString() != sharedNamespace {
		return nil
	}

	rootNSPath := path.Root("namespace")
	var rootNS types.String
	if diags := getter.GetAttribute(ctx, rootNSPath, &rootNS); diags.HasError() {
		return nil
	}
	if rootNS.IsNull() || rootNS.IsUnknown() || rootNS.ValueString() == sharedNamespace {
		return nil
	}

	var diags diag.Diagnostics
	diags.AddAttributeError(
		attrPath,
		"Missing Required Attribute",
		fmt.Sprintf(
			"Attribute %q is required when referencing an object in the %q namespace from a non-shared resource namespace (%q). "+
				"For F5 XC system-shared objects (such as virtual_network \"public\"), set tenant = \"ves-io\".",
			attrPath.String(),
			sharedNamespace,
			rootNS.ValueString(),
		),
	)
	return diags
}
