// Package client
// Copyright (c) 2026 F5 Inc. All rights reserved.

package client

import (
	"context"
	"strings"

	ves_io_schema_combined "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/combined"
)

var (
	// apiGWServiceSlugs is a mapping of public API FQN to the service slug to use on the api gateway
	// this will be replaced by discovery when api gateway supports it. The service slug is the part
	// at https://<xyz>.volterra.us/api/<service-slug>/<resource-uri>
	apiGWServiceSlugs = ves_io_schema_combined.MDR.APIGwServiceSlugs
)

// ctxServiceSlug is a context key type used to store the API gateway service slug in a context.Context.
// This struct is intentionally empty and serves only as a unique key for context value storage.
// Usage: context.WithValue(ctx, ctxServiceSlug{}, slug)
// The service slug identifies the target API gateway service for a request.
type ctxServiceSlug struct{}

// contextFromServiceSlug sets the API gateway service discriminator in the context.
// Parameters:
//   - ctx: The input context to be extended.
//   - slug: The service slug string to be added to the context.
//
// Returns:
//   - A new context.Context containing the service slug value.
func contextFromServiceSlug(ctx context.Context, slug string) context.Context {
	return context.WithValue(ctx, ctxServiceSlug{}, slug)
}

// addSvcSlugToContextForCRUD retrieves the service slug for the given object type and adds it to the context.
// Parameters:
//   - ctx: The input context to be extended.
//   - objType: The object type string for which the service slug is required.
//
// Returns:
//   - A new context.Context containing the service slug value if found.
//   - An error (always nil in current implementation, reserved for future use).
func addSvcSlugToContextForCRUD(ctx context.Context, objType string) (context.Context, error) {
	mdReg := ves_io_schema_combined.MDR
	apiFQN := mdReg.GetCRUDAPIName(objType, true)
	if slug, exists := apiGWServiceSlugs[apiFQN]; exists {
		ctx = contextFromServiceSlug(ctx, slug)
	}
	return ctx, nil
}

// addSvcSlugToContextForCustom retrieves the service slug for the given RPC FQN and adds it to the context.
// Parameters:
//   - ctx: The input context to be extended.
//   - rpcFQN: The fully qualified name of the RPC for which the service slug is required.
//
// Returns:
//   - A new context.Context containing the service slug value if found.
//   - An error (always nil in current implementation, reserved for future use).
func addSvcSlugToContextForCustom(ctx context.Context, rpcFQN string) (context.Context, error) {
	var pbSvc string
	rpcFQNList := strings.Split(rpcFQN, ".")
	pbSvc = strings.Join(rpcFQNList[:len(rpcFQNList)-1], ".")
	if slug, exists := apiGWServiceSlugs[pbSvc]; exists {
		ctx = contextFromServiceSlug(ctx, slug)
	}
	return ctx, nil
}
