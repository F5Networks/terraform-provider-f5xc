//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package client

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// knownCustomAPIService returns a custom API service name from the slug map,
// chosen deterministically, and its slug.
func knownCustomAPIService(t *testing.T) (string, string) {
	t.Helper()
	var services []string
	for k := range apiGWServiceSlugs {
		if strings.HasSuffix(k, "CustomAPI") {
			services = append(services, k)
		}
	}
	require.NotEmpty(t, services, "slug map must contain at least one custom API service")
	sort.Strings(services)
	return services[0], apiGWServiceSlugs[services[0]]
}

// TestAddSvcSlugToContextForCRUD verifies that a known object type gets the
// slug of its CRUD API, and that an unknown type leaves the context unchanged.
func TestAddSvcSlugToContextForCRUD(t *testing.T) {
	siteSlug := apiGWServiceSlugs["ves.io.schema.site.API"]
	require.NotEmpty(t, siteSlug, "site CRUD API must have a slug")

	tests := []struct {
		name     string
		objType  string
		initial  string
		wantSlug string
	}{
		{name: "known type", objType: "ves.io.schema.site.Object", wantSlug: siteSlug},
		{name: "known views type", objType: "ves.io.schema.views.http_loadbalancer.Object",
			wantSlug: apiGWServiceSlugs["ves.io.schema.views.http_loadbalancer.API"]},
		{name: "unknown type", objType: "ves.io.schema.does_not_exist.Object", wantSlug: ""},
		{name: "empty type", objType: "", wantSlug: ""},
		{name: "unknown type keeps existing slug", objType: "ves.io.schema.does_not_exist.Object", initial: "web", wantSlug: "web"},
		{name: "known type replaces existing slug", objType: "ves.io.schema.site.Object", initial: "web", wantSlug: siteSlug},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			if tc.initial != "" {
				ctx = contextFromServiceSlug(ctx, tc.initial)
			}

			got, err := addSvcSlugToContextForCRUD(ctx, tc.objType)

			require.NoError(t, err)
			assert.Equal(t, tc.wantSlug, serviceSlugFromContext(got))
		})
	}
}

// TestAddSvcSlugToContextForCustom verifies that the RPC method is dropped to
// find the service slug, and that an unknown service leaves the context unchanged.
func TestAddSvcSlugToContextForCustom(t *testing.T) {
	svc, slug := knownCustomAPIService(t)

	tests := []struct {
		name     string
		rpcFQN   string
		initial  string
		wantSlug string
	}{
		{name: "known service", rpcFQN: svc + ".SomeMethod", wantSlug: slug},
		{name: "service name without method", rpcFQN: svc, wantSlug: ""},
		{name: "unknown service", rpcFQN: "ves.io.schema.does_not_exist.CustomAPI.Do", wantSlug: ""},
		{name: "single segment", rpcFQN: "Do", wantSlug: ""},
		{name: "empty", rpcFQN: "", wantSlug: ""},
		{name: "unknown service keeps existing slug", rpcFQN: "x.y.Do", initial: "web", wantSlug: "web"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			if tc.initial != "" {
				ctx = contextFromServiceSlug(ctx, tc.initial)
			}

			got, err := addSvcSlugToContextForCustom(ctx, tc.rpcFQN)

			require.NoError(t, err)
			assert.Equal(t, tc.wantSlug, serviceSlugFromContext(got))
		})
	}
}

// TestAPIGWServiceSlugs verifies that the slug map is loaded and that no entry
// has an empty slug, which would produce a malformed gateway URL.
func TestAPIGWServiceSlugs(t *testing.T) {
	require.NotEmpty(t, apiGWServiceSlugs)
	for svc, slug := range apiGWServiceSlugs {
		assert.NotEmpty(t, strings.TrimSpace(slug), "service %q has an empty slug", svc)
	}
}
