//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// KnownLabelOverrideImportHook implements composite-key ImportState parsing for
// f5xc_known_label.
//
// Why: f5xc_known_label is uniquely identified by the tuple (namespace, key, value),
// because multiple known_label objects can share the same key with distinct values.
// In addition, Kubernetes-style label keys (e.g. "ves.io/region") contain slashes,
// which breaks naive strings.Split(req.ID, "/") parsing.
//
// How: Parses req.ID with parseKnownLabelImportID using the format
// "<namespace>/<key>=<value>" (splitting at the first "/" for namespace and the first
// "=" for key and value), populates "namespace" and "key", and populates "value" when
// non-empty (leaving "value" null for trailing "=" so Read() matches empty values).
//
// Parameters:
//   - ctx: request-scoped context from ImportState.
//   - req: Terraform ImportStateRequest containing the raw import ID.
//   - response: Terraform ImportStateResponse whose State and Diagnostics are updated in place.
//
// Returns:
//   - None. Updates response.State or appends "Invalid Import ID" error diagnostics.
//
// Example:
//
//	hook.KnownLabelOverrideImportHook(ctx, req, response)
func KnownLabelOverrideImportHook(ctx context.Context, req resource.ImportStateRequest, response *resource.ImportStateResponse) {
	namespace, key, value, err := parseKnownLabelImportID(req.ID)
	if err != nil {
		response.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("namespace"), namespace)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("key"), key)...)
	if value != "" {
		response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("value"), value)...)
	}
}

// parseKnownLabelImportID splits a f5xc_known_label import ID of the form
// "<namespace>/<key>=<value>" into its three components.
//
// Why: Separating the parsing logic allows unit testing import ID edge cases directly
// (e.g. label keys containing "/", empty values with trailing "=", and malformed inputs).
//
// How: Splits at the first "/" (namespaces cannot contain slashes) and splits the
// remainder at the first "=" (Kubernetes label keys and values cannot contain "=").
//
// Parameters:
//   - rawID: raw Terraform import ID string supplied via CLI or import block.
//
// Returns:
//   - namespace: parsed resource namespace.
//   - key: parsed label key (may contain "/").
//   - value: parsed label value (empty string when rawID ends with "=").
//   - err: descriptive error when rawID does not match "<namespace>/<key>=<value>" or
//     leaves namespace or key empty.
//
// Example:
//
//	ns, key, val, err := parseKnownLabelImportID("shared/ves.io/region=ves-io-par")
func parseKnownLabelImportID(rawID string) (namespace string, key string, value string, err error) {
	const usage = "Expected import ID format: <namespace>/<key>=<value> (for example: shared/env=prod, shared/ves.io/region=ves-io-par, or shared/my-key= for an empty value)"

	slashIdx := strings.IndexByte(rawID, '/')
	if slashIdx < 0 {
		return "", "", "", fmt.Errorf("%s, got: %s", usage, rawID)
	}

	namespace = rawID[:slashIdx]
	if namespace == "" {
		return "", "", "", fmt.Errorf("%s (namespace is empty), got: %s", usage, rawID)
	}

	rest := rawID[slashIdx+1:]
	eqIdx := strings.IndexByte(rest, '=')
	if eqIdx < 0 {
		return "", "", "", fmt.Errorf("%s (missing '=' separator before value), got: %s", usage, rawID)
	}

	key = rest[:eqIdx]
	if key == "" {
		return "", "", "", fmt.Errorf("%s (key is empty), got: %s", usage, rawID)
	}

	value = rest[eqIdx+1:]
	return namespace, key, value, nil
}
