//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package namespacetypevalidator

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

const (
	systemNamespace = "system"
	sharedNamespace = "shared"
	userNamespace   = "user"
)

type allowedInValidator struct {
	allowedTypes map[string]struct{}
}

var _ validator.String = allowedInValidator{}

// AllowedIn creates a validator for the namespace categories allowed by an object schema.
//
// Why: in_namespace_types identifies backend namespace categories, not literal namespace
// names. A user namespace can have any name other than the reserved system and shared names.
//
// How: The validator stores the configured categories and compares a configured namespace
// against the same reserved-name rules used by the backend.
//
// Parameters:
//   - namespaceTypes: allowed categories; supported values are system, shared, and user.
//
// Returns:
//   - validator.String: a Terraform schema validator for the supplied categories.
func AllowedIn(namespaceTypes ...string) validator.String {
	allowedTypes := make(map[string]struct{}, len(namespaceTypes))
	for _, namespaceType := range namespaceTypes {
		allowedTypes[namespaceType] = struct{}{}
	}

	return allowedInValidator{allowedTypes: allowedTypes}
}

// Description returns the namespace-category rule represented by this validator.
//
// Parameters:
//   - ctx: context for Terraform validator descriptions.
//
// Returns:
//   - string: a human-readable description of the allowed namespace categories.
func (v allowedInValidator) Description(_ context.Context) string {
	types := make([]string, 0, len(v.allowedTypes))
	for _, namespaceType := range []string{systemNamespace, sharedNamespace, userNamespace} {
		if _, ok := v.allowedTypes[namespaceType]; ok {
			types = append(types, namespaceType)
		}
	}

	return fmt.Sprintf("namespace must belong to one of: %s", strings.Join(types, ", "))
}

// MarkdownDescription returns a Markdown-formatted namespace-category rule.
//
// Parameters:
//   - ctx: context for Terraform validator descriptions.
//
// Returns:
//   - string: the Markdown description.
func (v allowedInValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// ValidateString validates a configured namespace without performing an API request.
//
// Why: Terraform should reject invalid namespace categories during planning while allowing
// arbitrary user namespace names accepted by the backend.
//
// How: system and shared are reserved names; every other name belongs to the user category.
// Null and unknown values are deferred to Terraform's normal planning lifecycle.
//
// Parameters:
//   - ctx: context for Terraform validation.
//   - req: configured attribute path and value.
//   - resp: diagnostics to which an invalid namespace error is appended.
func (v allowedInValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	if v.allows(req.ConfigValue.ValueString()) {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid Namespace",
		fmt.Sprintf("Namespace %q is not allowed. namespace must belong to one of: %s", req.ConfigValue.ValueString(), v.suggestion()),
	)
}

// suggestion returns an actionable replacement hint for the configured namespace categories.
//
// Parameters: none.
//
// Returns:
//   - string: a user-facing suggestion for a valid namespace category.
func (v allowedInValidator) suggestion() string {
	_, allowsSystem := v.allowedTypes[systemNamespace]
	_, allowsShared := v.allowedTypes[sharedNamespace]
	_, allowsUser := v.allowedTypes[userNamespace]

	switch {
	case allowsSystem && allowsShared && allowsUser:
		return "the system namespace, shared namespace, or a user-defined namespace instead."
	case allowsSystem && allowsShared:
		return "the system or shared namespace instead."
	case allowsSystem && allowsUser:
		return "the system namespace or a user-defined namespace instead."
	case allowsShared && allowsUser:
		return "shared namespace or a user-defined namespace instead."
	case allowsSystem:
		return "the system namespace instead."
	case allowsShared:
		return "shared namespace instead."
	case allowsUser:
		return "a user-defined namespace instead."
	default:
		return "a namespace allowed by this resource instead."
	}
}

// allows mirrors backend namespace category handling using only local string checks.
//
// Parameters:
//   - namespace: configured namespace value.
//
// Returns:
//   - bool: true when namespace belongs to an allowed category.
func (v allowedInValidator) allows(namespace string) bool {
	if _, ok := v.allowedTypes[systemNamespace]; ok && namespace == systemNamespace {
		return true
	}
	if _, ok := v.allowedTypes[sharedNamespace]; ok && namespace == sharedNamespace {
		return true
	}
	if _, ok := v.allowedTypes[userNamespace]; ok && namespace != systemNamespace && namespace != sharedNamespace {
		return true
	}

	return false
}
