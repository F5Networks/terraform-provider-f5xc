//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"
	"strings"

	"gopkg.volterra.us/stdlib/client/vesapi"

	ves_io_schema_virtual_site "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/virtual_site"
)

type virtualSiteDeleteClient interface {
	DeleteObject(ctx context.Context, objType string, namespace, name string, opts ...vesapi.CallOpt) error
}

// VirtualSiteOverrideDeleteHook deletes a virtual_site resource with optional
// reference checking and treats only "NotFound" errors as already-gone.
//
// Why: A bare "status code 404" is not silenced because the F5 XC backend API
// returns HTTP 404 for unrelated error conditions on virtual_site. Additionally,
// virtual_site deletion supports fail_if_referred to prevent deleting a virtual
// site that is still referenced by other objects.
//
// How: Appends vesapi.WithFailIfReferred() to the DeleteObject call options when
// failIfReferred is true, invokes apiClient.DeleteObject for the virtual_site
// object type, and ignores errors containing "NotFound".
//
// Parameters:
//   - ctx: request-scoped context propagated from the resource Delete() method.
//   - apiClient: client capable of invoking DeleteObject with vesapi.CallOpt options.
//   - namespace: namespace of the virtual_site resource.
//   - name: name of the virtual_site resource.
//   - failIfReferred: when true, passes vesapi.WithFailIfReferred() so deletion
//     fails if the virtual_site is still referenced.
//
// Returns:
//   - error: nil when deletion succeeds or the backend reports "NotFound";
//     otherwise returns the underlying DeleteObject error.
//
// Example:
//
//	err := VirtualSiteOverrideDeleteHook(ctx, r.client, data.Namespace.ValueString(), data.Name.ValueString(), true)
func VirtualSiteOverrideDeleteHook(ctx context.Context, apiClient virtualSiteDeleteClient, namespace, name string, failIfReferred bool) error {
	var opts []vesapi.CallOpt
	if failIfReferred {
		opts = []vesapi.CallOpt{
			vesapi.WithFailIfReferred(),
		}
	}

	err := apiClient.DeleteObject(ctx, ves_io_schema_virtual_site.ObjectType, namespace, name, opts...)
	if err != nil && strings.Contains(err.Error(), "NotFound") {
		return nil
	}
	return err
}
