//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package resource

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResourceF5XCEndpoint_WhereTenantSchemaAndDocs verifies that all ObjectRefType.tenant
// attributes under f5xc_endpoint.where.ref_or_selector_choice.{site,virtual_network,virtual_site}.ref
// are generated as Optional+Computed with non-empty pattern and cross-namespace validation.
//
// Why: Having endpoint ObjectRefType.tenant Required forced users to set tenant = "" on
// same-tenant references, triggering "Provider produced inconsistent result after apply" when the
// backend populated the tenant string.
//
// How: Instantiates NewResourceF5XCEndpoint(), traverses all three where ref branches, and asserts
// Required/Optional/Computed flags, documentation, validator count, and plan modifier count.
//
// Parameters:
//   - t: *testing.T test runner handle.
//
// Returns:
//   - None.
func TestResourceF5XCEndpoint_WhereTenantSchemaAndDocs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	NewResourceF5XCEndpoint().Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())

	whereAttr := requireSingleNestedAttr(t, schemaResp.Schema.Attributes["where"])
	choiceAttr := requireSingleNestedAttr(t, whereAttr.Attributes["ref_or_selector_choice"])

	for _, branch := range []string{"site", "virtual_network", "virtual_site"} {
		t.Run(branch, func(t *testing.T) {
			branchAttr := requireSingleNestedAttr(t, choiceAttr.Attributes[branch])
			refAttr := requireListNestedAttr(t, branchAttr.Attributes["ref"])
			tenantAttr := requireStringAttr(t, refAttr.NestedObject.Attributes["tenant"])

			assert.False(t, tenantAttr.Required, "tenant must not be Required for %s.ref", branch)
			assert.True(t, tenantAttr.Optional, "tenant must be Optional for %s.ref", branch)
			assert.True(t, tenantAttr.Computed, "tenant must be Computed for %s.ref", branch)
			assert.Len(t, tenantAttr.Validators, 2, "expected regex and cross-ns validators on %s.ref.tenant", branch)
			assert.Len(t, tenantAttr.PlanModifiers, 2, "expected UseNonNullStateForUnknown and cross-ns modifiers on %s.ref.tenant", branch)
			assert.Contains(t, tenantAttr.MarkdownDescription, "ves-io")
			assert.Contains(t, tenantAttr.MarkdownDescription, "shared")
		})
	}
}

// TestResourceF5XCEndpoint_WhereVirtualSiteTenantValidation verifies plan/validate behavior for
// f5xc_endpoint.where.ref_or_selector_choice.virtual_site.ref[0].tenant across omitted, empty,
// whitespace-only, and cross-namespace shared reference inputs.
//
// Why: Ensures omitting tenant works for same-tenant references while empty/whitespace strings
// (tenant = "" / "   ") and omitted cross-namespace shared references fail before apply.
//
// How: Builds synthetic tfsdk.Config and tfsdk.Plan from ResourceF5XCEndpoint's real schema and
// executes all attached String Validators and PlanModifiers on virtual_site.ref[0].tenant.
//
// Parameters:
//   - t: *testing.T test runner handle.
//
// Returns:
//   - None.
func TestResourceF5XCEndpoint_WhereVirtualSiteTenantValidation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	NewResourceF5XCEndpoint().Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())

	whereAttr := requireSingleNestedAttr(t, schemaResp.Schema.Attributes["where"])
	choiceAttr := requireSingleNestedAttr(t, whereAttr.Attributes["ref_or_selector_choice"])
	vsiteAttr := requireSingleNestedAttr(t, choiceAttr.Attributes["virtual_site"])
	refAttr := requireListNestedAttr(t, vsiteAttr.Attributes["ref"])
	tenantAttr := requireStringAttr(t, refAttr.NestedObject.Attributes["tenant"])

	tenantPath := path.Root("where").
		AtName("ref_or_selector_choice").
		AtName("virtual_site").
		AtName("ref").
		AtListIndex(0).
		AtName("tenant")

	tests := []struct {
		name        string
		rootNS      string
		refNS       string
		tenant      *string
		wantValErr  bool
		wantPlanErr bool
	}{
		{
			name:        "system-to-system-tenant-omitted-succeeds",
			rootNS:      "system",
			refNS:       "system",
			tenant:      nil,
			wantValErr:  false,
			wantPlanErr: false,
		},
		{
			name:        "system-to-system-tenant-empty-string-rejected",
			rootNS:      "system",
			refNS:       "system",
			tenant:      strPtr(""),
			wantValErr:  true,
			wantPlanErr: false,
		},
		{
			name:        "system-to-system-tenant-whitespace-only-rejected",
			rootNS:      "system",
			refNS:       "system",
			tenant:      strPtr("   "),
			wantValErr:  true,
			wantPlanErr: false,
		},
		{
			name:        "system-to-shared-tenant-omitted-rejected",
			rootNS:      "system",
			refNS:       "shared",
			tenant:      nil,
			wantValErr:  true,
			wantPlanErr: true,
		},
		{
			name:        "system-to-shared-tenant-ves-io-succeeds",
			rootNS:      "system",
			refNS:       "shared",
			tenant:      strPtr("ves-io"),
			wantValErr:  false,
			wantPlanErr: false,
		},
		{
			name:        "shared-to-shared-tenant-omitted-succeeds",
			rootNS:      "shared",
			refNS:       "shared",
			tenant:      nil,
			wantValErr:  false,
			wantPlanErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg, plan := buildEndpointVirtualSiteConfigAndPlan(t, ctx, schemaResp.Schema, tc.rootNS, tc.refNS, tc.tenant)
			cfgVal := toTypesString(tc.tenant)

			valDiags := runStringValidators(ctx, tenantAttr.Validators, tenantPath, cfg, cfgVal)
			if tc.wantValErr {
				assert.True(t, valDiags.HasError(), "expected validation error for case %s", tc.name)
			} else {
				assert.False(t, valDiags.HasError(), "unexpected validation error: %v", valDiags)
			}

			planDiags := runStringPlanModifiers(ctx, tenantAttr.PlanModifiers, tenantPath, cfg, plan, cfgVal)
			if tc.wantPlanErr {
				assert.True(t, planDiags.HasError(), "expected plan modifier error for case %s", tc.name)
			} else {
				assert.False(t, planDiags.HasError(), "unexpected plan modifier error: %v", planDiags)
			}
		})
	}
}

// TestResourceF5XCDnsLoadBalancer_FallbackPoolTenantRejectsEmpty verifies that Required
// ObjectRefType.tenant fields (such as dns_load_balancer.fallback_pool.tenant) also reject
// empty and whitespace-only strings at validate time.
//
// Why: Ensures Required ObjectRefType.tenant attributes are equally protected against empty
// string submissions that would trigger post-apply state inconsistencies.
//
// How: Builds a minimal Config for ResourceF5XCDnsLoadBalancer and invokes fallback_pool.tenant
// Validators with empty, whitespace-only, and valid tenant strings.
//
// Parameters:
//   - t: *testing.T test runner handle.
//
// Returns:
//   - None.
func TestResourceF5XCDnsLoadBalancer_FallbackPoolTenantRejectsEmpty(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	NewResourceF5XCDnsLoadBalancer().Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())

	fallbackAttr := requireSingleNestedAttr(t, schemaResp.Schema.Attributes["fallback_pool"])
	tenantAttr := requireStringAttr(t, fallbackAttr.Attributes["tenant"])
	assert.True(t, tenantAttr.Required)

	rootType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	require.True(t, ok)
	fbType := rootType.AttributeTypes["fallback_pool"].(tftypes.Object)
	tenantPath := path.Root("fallback_pool").AtName("tenant")

	for _, tc := range []struct {
		name    string
		tenant  string
		wantErr bool
	}{
		{name: "empty-rejected", tenant: "", wantErr: true},
		{name: "whitespace-rejected", tenant: " \t ", wantErr: true},
		{name: "valid-accepted", tenant: "customer1", wantErr: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fbMap := newNullObjectMap(fbType)
			fbMap["name"] = tftypes.NewValue(tftypes.String, "pool-1")
			fbMap["namespace"] = tftypes.NewValue(tftypes.String, "system")
			fbMap["tenant"] = tftypes.NewValue(tftypes.String, tc.tenant)

			rootMap := newNullObjectMap(rootType)
			rootMap["name"] = tftypes.NewValue(tftypes.String, "test-dns-lb")
			rootMap["namespace"] = tftypes.NewValue(tftypes.String, "system")
			rootMap["fallback_pool"] = tftypes.NewValue(fbType, fbMap)

			cfg := tfsdk.Config{Raw: tftypes.NewValue(rootType, rootMap), Schema: schemaResp.Schema}
			diags := runStringValidators(ctx, tenantAttr.Validators, tenantPath, cfg, toTypesString(strPtr(tc.tenant)))
			assert.Equal(t, tc.wantErr, diags.HasError())
		})
	}
}

// requireListNestedAttr asserts that attr is a schema.ListNestedAttribute and returns it.
//
// Why: Simplifies traversing ListNestedAttribute fields such as ObjectRefType ref lists.
//
// How: Performs a type assertion via testify require.
//
// Parameters:
//   - t:    *testing.T test runner handle.
//   - attr: schema.Attribute to unwrap.
//
// Returns:
//   - schema.ListNestedAttribute: unwrapped attribute.
func requireListNestedAttr(t *testing.T, attr schema.Attribute) schema.ListNestedAttribute {
	t.Helper()
	nested, ok := attr.(schema.ListNestedAttribute)
	require.True(t, ok, "expected schema.ListNestedAttribute, got %T", attr)
	return nested
}

// buildEndpointVirtualSiteConfigAndPlan constructs synthetic tfsdk.Config and tfsdk.Plan
// values for ResourceF5XCEndpoint with where.ref_or_selector_choice.virtual_site.ref[0] set.
//
// Why: Enables isolated unit testing of endpoint's virtual_site reference validation without
// booting a full Terraform CLI subprocess or contacting a live API.
//
// How: Derives the tftypes.Object hierarchy from ResourceF5XCEndpoint's schema and populates
// namespace plus the virtual_site.ref list element.
//
// Parameters:
//   - t:      *testing.T test runner handle.
//   - ctx:    context.Context for TerraformType derivation.
//   - s:      schema.Schema of ResourceF5XCEndpoint.
//   - rootNS: resource-level namespace value.
//   - refNS:  virtual_site reference namespace value.
//   - tenant: optional tenant value for virtual_site.ref[0].
//
// Returns:
//   - tfsdk.Config: constructed config object.
//   - tfsdk.Plan:   constructed plan object.
func buildEndpointVirtualSiteConfigAndPlan(
	t *testing.T,
	ctx context.Context,
	s schema.Schema,
	rootNS string,
	refNS string,
	tenant *string,
) (tfsdk.Config, tfsdk.Plan) {
	t.Helper()

	rootType, ok := s.Type().TerraformType(ctx).(tftypes.Object)
	require.True(t, ok)

	whereType := rootType.AttributeTypes["where"].(tftypes.Object)
	choiceType := whereType.AttributeTypes["ref_or_selector_choice"].(tftypes.Object)
	vsiteType := choiceType.AttributeTypes["virtual_site"].(tftypes.Object)
	refListType := vsiteType.AttributeTypes["ref"].(tftypes.List)
	refElemType := refListType.ElementType.(tftypes.Object)

	refMap := newNullObjectMap(refElemType)
	refMap["name"] = tftypes.NewValue(tftypes.String, "repro-endpoint-bug-vsite")
	refMap["namespace"] = tftypes.NewValue(tftypes.String, refNS)
	if tenant != nil {
		refMap["tenant"] = tftypes.NewValue(tftypes.String, *tenant)
	}

	vsiteMap := newNullObjectMap(vsiteType)
	vsiteMap["ref"] = tftypes.NewValue(refListType, []tftypes.Value{
		tftypes.NewValue(refElemType, refMap),
	})

	choiceMap := newNullObjectMap(choiceType)
	choiceMap["virtual_site"] = tftypes.NewValue(vsiteType, vsiteMap)

	whereMap := newNullObjectMap(whereType)
	whereMap["ref_or_selector_choice"] = tftypes.NewValue(choiceType, choiceMap)

	rootMap := newNullObjectMap(rootType)
	rootMap["name"] = tftypes.NewValue(tftypes.String, "repro-endpoint-bug-ep")
	rootMap["namespace"] = tftypes.NewValue(tftypes.String, rootNS)
	rootMap["where"] = tftypes.NewValue(whereType, whereMap)

	raw := tftypes.NewValue(rootType, rootMap)
	return tfsdk.Config{Raw: raw, Schema: s}, tfsdk.Plan{Raw: raw, Schema: s}
}
