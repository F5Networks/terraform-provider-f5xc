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

// VirtualSiteOverrideDeleteHook deletes a virtual_site and treats only "NotFound"
// as already-gone. A bare "status code 404" is NOT silenced because the API
// returns 404 for unrelated error conditions on this resource type.
// This is a temporary workaround until the API is fixed in 2026-10-0 release.
func VirtualSiteOverrideDeleteHook(ctx context.Context, apiClient virtualSiteDeleteClient, namespace, name string) error {
	opts := []vesapi.CallOpt{
		vesapi.WithFailIfReferred(),
	}

	err := apiClient.DeleteObject(ctx, ves_io_schema_virtual_site.ObjectType, namespace, name, opts...)
	if err != nil && strings.Contains(err.Error(), "NotFound") {
		return nil
	}
	return err
}
