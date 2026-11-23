//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package resource

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResourceF5XCExternalConnector_CrossNamespaceSharedTenantDocs verifies that
// ObjectRefType child attributes on f5xc_external_connector (ce_site_reference,
// ike_phase1_profile, and ike_phase2_profile) expose non-empty documentation and
// explicitly mention tenant = "ves-io" for cross-namespace shared references.
//
// Why: tfplugindocs renders schema attribute Description/MarkdownDescription fields
// rather than validator descriptions, so missing attribute docs hide the mandatory
// tenant = "ves-io" requirement from users.
//
// How: Instantiates ResourceF5XCExternalConnector, retrieves its Schema, traverses
// ce_site_reference and connection_type_choice.ipsec.ike_parameters.{ike_phase1_profile,ike_phase2_profile},
// and asserts non-empty descriptions with cross-namespace tenant guidance.
//
// Parameters:
//   - t: *testing.T test runner handle.
//
// Returns:
//   - None.
func TestResourceF5XCExternalConnector_CrossNamespaceSharedTenantDocs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	NewResourceF5XCExternalConnector().Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())

	ceSiteAttr := requireSingleNestedAttr(t, schemaResp.Schema.Attributes["ce_site_reference"])
	ikeParamsAttr := requireSingleNestedAttr(t,
		requireSingleNestedAttr(t,
			requireSingleNestedAttr(t, schemaResp.Schema.Attributes["connection_type_choice"]).
				Attributes["ipsec"],
		).Attributes["ike_parameters"],
	)
	ikeP1Attr := requireSingleNestedAttr(t, ikeParamsAttr.Attributes["ike_phase1_profile"])
	ikeP2Attr := requireSingleNestedAttr(t, ikeParamsAttr.Attributes["ike_phase2_profile"])

	for name, refAttr := range map[string]schema.SingleNestedAttribute{
		"ce_site_reference":  ceSiteAttr,
		"ike_phase1_profile": ikeP1Attr,
		"ike_phase2_profile": ikeP2Attr,
	} {
		t.Run(name, func(t *testing.T) {
			nameAttr := requireStringAttr(t, refAttr.Attributes["name"])
			assert.NotEmpty(t, nameAttr.Description)
			assert.NotEmpty(t, nameAttr.MarkdownDescription)

			nsAttr := requireStringAttr(t, refAttr.Attributes["namespace"])
			assert.NotEmpty(t, nsAttr.Description)
			assert.NotEmpty(t, nsAttr.MarkdownDescription)

			tenantAttr := requireStringAttr(t, refAttr.Attributes["tenant"])
			assert.NotEmpty(t, tenantAttr.Description)
			assert.NotEmpty(t, tenantAttr.MarkdownDescription)
			assert.Contains(t, tenantAttr.MarkdownDescription, "ves-io")
			assert.Contains(t, tenantAttr.MarkdownDescription, "shared")
		})
	}
}

// TestResourceF5XCExternalConnector_IPSecIKEProfilesCrossNamespaceValidation verifies
// that the generated Validators and PlanModifiers on ike_phase1_profile.tenant and
// ike_phase2_profile.tenant enforce tenant when referencing "shared" from a non-shared
// namespace while permitting omitted tenant within the "shared" namespace.
//
// Why: Ensures f5xc_external_connector IPSec configurations catch missing tenant = "ves-io"
// on predefined IKE profiles during validate and plan.
//
// How: Builds tfsdk.Config and tfsdk.Plan from the real ResourceF5XCExternalConnector schema
// and executes the attached string Validators and PlanModifiers on both profile tenant paths.
//
// Parameters:
//   - t: *testing.T test runner handle.
//
// Returns:
//   - None.
func TestResourceF5XCExternalConnector_IPSecIKEProfilesCrossNamespaceValidation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	NewResourceF5XCExternalConnector().Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())

	ikeParamsAttr := requireSingleNestedAttr(t,
		requireSingleNestedAttr(t,
			requireSingleNestedAttr(t, schemaResp.Schema.Attributes["connection_type_choice"]).
				Attributes["ipsec"],
		).Attributes["ike_parameters"],
	)
	p1TenantAttr := requireStringAttr(t, requireSingleNestedAttr(t, ikeParamsAttr.Attributes["ike_phase1_profile"]).Attributes["tenant"])
	p2TenantAttr := requireStringAttr(t, requireSingleNestedAttr(t, ikeParamsAttr.Attributes["ike_phase2_profile"]).Attributes["tenant"])

	p1Path := path.Root("connection_type_choice").AtName("ipsec").AtName("ike_parameters").AtName("ike_phase1_profile").AtName("tenant")
	p2Path := path.Root("connection_type_choice").AtName("ipsec").AtName("ike_parameters").AtName("ike_phase2_profile").AtName("tenant")

	tests := []struct {
		name      string
		rootNS    string
		p1Tenant  *string
		p2Tenant  *string
		wantP1Err bool
		wantP2Err bool
	}{
		{
			name:      "system-ns-missing-ike-phase1-tenant-rejected",
			rootNS:    "system",
			p1Tenant:  nil,
			p2Tenant:  strPtr("ves-io"),
			wantP1Err: true,
			wantP2Err: false,
		},
		{
			name:      "system-ns-missing-ike-phase2-tenant-rejected",
			rootNS:    "system",
			p1Tenant:  strPtr("ves-io"),
			p2Tenant:  nil,
			wantP1Err: false,
			wantP2Err: true,
		},
		{
			name:      "system-ns-with-ves-io-tenant-succeeds",
			rootNS:    "system",
			p1Tenant:  strPtr("ves-io"),
			p2Tenant:  strPtr("ves-io"),
			wantP1Err: false,
			wantP2Err: false,
		},
		{
			name:      "shared-ns-with-omitted-tenant-succeeds",
			rootNS:    "shared",
			p1Tenant:  nil,
			p2Tenant:  nil,
			wantP1Err: false,
			wantP2Err: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg, plan := buildExternalConnectorIPSecConfigAndPlan(t, ctx, schemaResp.Schema, tc.rootNS, tc.p1Tenant, tc.p2Tenant)

			p1ValDiags := runStringValidators(ctx, p1TenantAttr.Validators, p1Path, cfg, toTypesString(tc.p1Tenant))
			p1PlanDiags := runStringPlanModifiers(ctx, p1TenantAttr.PlanModifiers, p1Path, cfg, plan, toTypesString(tc.p1Tenant))
			if tc.wantP1Err {
				assert.True(t, p1ValDiags.HasError(), "expected validator error for ike_phase1_profile.tenant")
				assert.True(t, p1PlanDiags.HasError(), "expected plan modifier error for ike_phase1_profile.tenant")
			} else {
				assert.False(t, p1ValDiags.HasError(), "unexpected validator error for ike_phase1_profile.tenant: %v", p1ValDiags)
				assert.False(t, p1PlanDiags.HasError(), "unexpected plan modifier error for ike_phase1_profile.tenant: %v", p1PlanDiags)
			}

			p2ValDiags := runStringValidators(ctx, p2TenantAttr.Validators, p2Path, cfg, toTypesString(tc.p2Tenant))
			p2PlanDiags := runStringPlanModifiers(ctx, p2TenantAttr.PlanModifiers, p2Path, cfg, plan, toTypesString(tc.p2Tenant))
			if tc.wantP2Err {
				assert.True(t, p2ValDiags.HasError(), "expected validator error for ike_phase2_profile.tenant")
				assert.True(t, p2PlanDiags.HasError(), "expected plan modifier error for ike_phase2_profile.tenant")
			} else {
				assert.False(t, p2ValDiags.HasError(), "unexpected validator error for ike_phase2_profile.tenant: %v", p2ValDiags)
				assert.False(t, p2PlanDiags.HasError(), "unexpected plan modifier error for ike_phase2_profile.tenant: %v", p2PlanDiags)
			}
		})
	}
}

// requireSingleNestedAttr asserts that attr is a schema.SingleNestedAttribute and returns it.
//
// Why: Simplifies nested schema traversal in unit tests with clear failure diagnostics.
//
// How: Performs a type assertion via testify require.
//
// Parameters:
//   - t:    *testing.T test runner handle.
//   - attr: schema.Attribute to unwrap.
//
// Returns:
//   - schema.SingleNestedAttribute: unwrapped attribute.
func requireSingleNestedAttr(t *testing.T, attr schema.Attribute) schema.SingleNestedAttribute {
	t.Helper()
	nested, ok := attr.(schema.SingleNestedAttribute)
	require.True(t, ok, "expected schema.SingleNestedAttribute, got %T", attr)
	return nested
}

// requireStringAttr asserts that attr is a schema.StringAttribute and returns it.
//
// Why: Simplifies leaf attribute inspection in unit tests with clear failure diagnostics.
//
// How: Performs a type assertion via testify require.
//
// Parameters:
//   - t:    *testing.T test runner handle.
//   - attr: schema.Attribute to unwrap.
//
// Returns:
//   - schema.StringAttribute: unwrapped attribute.
func requireStringAttr(t *testing.T, attr schema.Attribute) schema.StringAttribute {
	t.Helper()
	strAttr, ok := attr.(schema.StringAttribute)
	require.True(t, ok, "expected schema.StringAttribute, got %T", attr)
	return strAttr
}

// strPtr returns a pointer to the given string literal.
//
// Why: Allows expressing nullable string test inputs concisely in table-driven test cases.
//
// How: Allocates and returns the address of s.
//
// Parameters:
//   - s: string value to wrap.
//
// Returns:
//   - *string: pointer to s.
func strPtr(s string) *string {
	return &s
}

// toTypesString converts an optional *string into a types.String (null when nil).
//
// Why: Aligns ConfigValue/PlanValue inputs with nullable test cases.
//
// How: Returns types.StringNull() when s is nil, otherwise types.StringValue(*s).
//
// Parameters:
//   - s: *string optional string pointer.
//
// Returns:
//   - types.String: Framework string value.
func toTypesString(s *string) types.String {
	if s == nil {
		return types.StringNull()
	}
	return types.StringValue(*s)
}

// newNullObjectMap creates a map of null tftypes.Values for every attribute of an object type.
//
// Why: tftypes.NewValue requires all object attributes to be present in the backing map.
//
// How: Iterates over objType.AttributeTypes and populates each key with a null tftypes.Value.
//
// Parameters:
//   - objType: tftypes.Object schema type descriptor.
//
// Returns:
//   - map[string]tftypes.Value: map initialized with null values for every attribute.
func newNullObjectMap(objType tftypes.Object) map[string]tftypes.Value {
	m := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for k, vType := range objType.AttributeTypes {
		m[k] = tftypes.NewValue(vType, nil)
	}
	return m
}

// buildExternalConnectorIPSecConfigAndPlan constructs a tfsdk.Config and tfsdk.Plan
// for f5xc_external_connector with IPSec ike_phase1_profile and ike_phase2_profile populated.
//
// Why: Enables testing the real schema Validators and PlanModifiers on nested IPSec IKE profile
// tenant attributes without spinning up a full provider server.
//
// How: Derives the tftypes.Object hierarchy from the resource schema and populates namespace plus
// connection_type_choice.ipsec.ike_parameters.{ike_phase1_profile,ike_phase2_profile}.
//
// Parameters:
//   - t:        *testing.T test runner handle.
//   - ctx:      context.Context for TerraformType derivation.
//   - s:        schema.Schema of ResourceF5XCExternalConnector.
//   - rootNS:   resource-level namespace value.
//   - p1Tenant: optional tenant value for ike_phase1_profile.
//   - p2Tenant: optional tenant value for ike_phase2_profile.
//
// Returns:
//   - tfsdk.Config: constructed config object.
//   - tfsdk.Plan:   constructed plan object.
func buildExternalConnectorIPSecConfigAndPlan(
	t *testing.T,
	ctx context.Context,
	s schema.Schema,
	rootNS string,
	p1Tenant *string,
	p2Tenant *string,
) (tfsdk.Config, tfsdk.Plan) {
	t.Helper()

	rootType, ok := s.Type().TerraformType(ctx).(tftypes.Object)
	require.True(t, ok)

	connType := rootType.AttributeTypes["connection_type_choice"].(tftypes.Object)
	ipsecType := connType.AttributeTypes["ipsec"].(tftypes.Object)
	ikeParamsType := ipsecType.AttributeTypes["ike_parameters"].(tftypes.Object)
	p1Type := ikeParamsType.AttributeTypes["ike_phase1_profile"].(tftypes.Object)
	p2Type := ikeParamsType.AttributeTypes["ike_phase2_profile"].(tftypes.Object)

	p1Map := newNullObjectMap(p1Type)
	p1Map["name"] = tftypes.NewValue(tftypes.String, "ves-io-f5-ike-phase1-default-profile")
	p1Map["namespace"] = tftypes.NewValue(tftypes.String, "shared")
	if p1Tenant != nil {
		p1Map["tenant"] = tftypes.NewValue(tftypes.String, *p1Tenant)
	}

	p2Map := newNullObjectMap(p2Type)
	p2Map["name"] = tftypes.NewValue(tftypes.String, "ves-io-f5-ike-phase2-default-profile")
	p2Map["namespace"] = tftypes.NewValue(tftypes.String, "shared")
	if p2Tenant != nil {
		p2Map["tenant"] = tftypes.NewValue(tftypes.String, *p2Tenant)
	}

	ikeParamsMap := newNullObjectMap(ikeParamsType)
	ikeParamsMap["ike_phase1_profile"] = tftypes.NewValue(p1Type, p1Map)
	ikeParamsMap["ike_phase2_profile"] = tftypes.NewValue(p2Type, p2Map)

	ipsecMap := newNullObjectMap(ipsecType)
	ipsecMap["ike_parameters"] = tftypes.NewValue(ikeParamsType, ikeParamsMap)

	connMap := newNullObjectMap(connType)
	connMap["ipsec"] = tftypes.NewValue(ipsecType, ipsecMap)

	rootMap := newNullObjectMap(rootType)
	rootMap["name"] = tftypes.NewValue(tftypes.String, "test-external-connector")
	rootMap["namespace"] = tftypes.NewValue(tftypes.String, rootNS)
	rootMap["connection_type_choice"] = tftypes.NewValue(connType, connMap)

	raw := tftypes.NewValue(rootType, rootMap)
	return tfsdk.Config{Raw: raw, Schema: s}, tfsdk.Plan{Raw: raw, Schema: s}
}

// runStringValidators executes a slice of validator.String instances and collects diagnostics.
//
// Why: Isolates validation execution for nested string attributes in unit tests.
//
// How: Invokes ValidateString on each validator and appends its diagnostics.
//
// Parameters:
//   - ctx:        context.Context for validation.
//   - validators: []validator.String attached to the schema attribute.
//   - attrPath:   path.Path to the attribute under test.
//   - cfg:        tfsdk.Config containing the test configuration.
//   - cfgVal:     types.String attribute configuration value.
//
// Returns:
//   - []string / diag.Diagnostics: collected diagnostics via validator.StringResponse.
func runStringValidators(
	ctx context.Context,
	validators []validator.String,
	attrPath path.Path,
	cfg tfsdk.Config,
	cfgVal types.String,
) diag.Diagnostics {
	resp := &validator.StringResponse{}
	for _, v := range validators {
		v.ValidateString(ctx, validator.StringRequest{
			Path:        attrPath,
			Config:      cfg,
			ConfigValue: cfgVal,
		}, resp)
	}
	return resp.Diagnostics
}

// runStringPlanModifiers executes a slice of planmodifier.String instances and collects diagnostics.
//
// Why: Isolates plan modifier execution for nested string attributes in unit tests.
//
// How: Invokes PlanModifyString on each modifier with Unknown PlanValue when cfgVal is null.
//
// Parameters:
//   - ctx:       context.Context for plan modification.
//   - modifiers: []planmodifier.String attached to the schema attribute.
//   - attrPath:  path.Path to the attribute under test.
//   - cfg:       tfsdk.Config containing the test configuration.
//   - plan:      tfsdk.Plan containing the planned values.
//   - cfgVal:    types.String attribute configuration value.
//
// Returns:
//   - diag.Diagnostics: collected diagnostics from all plan modifiers.
func runStringPlanModifiers(
	ctx context.Context,
	modifiers []planmodifier.String,
	attrPath path.Path,
	cfg tfsdk.Config,
	plan tfsdk.Plan,
	cfgVal types.String,
) diag.Diagnostics {
	planVal := cfgVal
	if planVal.IsNull() {
		planVal = types.StringUnknown()
	}
	resp := &planmodifier.StringResponse{PlanValue: planVal}
	for _, m := range modifiers {
		m.PlanModifyString(ctx, planmodifier.StringRequest{
			Path:        attrPath,
			Config:      cfg,
			ConfigValue: cfgVal,
			Plan:        plan,
			PlanValue:   planVal,
		}, resp)
	}
	return resp.Diagnostics
}
