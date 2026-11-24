//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package requiredifvalidator_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/validators/requiredifvalidator"
)

// TestCrossNamespaceSharedTenantValidator verifies all QA cases (A, B, C static, C dynamic, C2)
// and edge cases for CrossNamespaceSharedTenantValidator across both ValidateString and PlanModifyString.
//
// Why: Ensures that omitting ObjectRefType.tenant when referencing the "shared" namespace
// from a non-shared resource namespace is reliably rejected during terraform validate/plan,
// while preserving omitted tenant convenience inside the "shared" namespace.
//
// How: Executes table-driven subtests covering Phase 1 (ValidateString) and Phase 2 (PlanModifyString),
// including unknown root.namespace resolution in Phase 2.
//
// Parameters:
//   - t: *testing.T test runner handle.
//
// Returns:
//   - None.
func TestCrossNamespaceSharedTenantValidator(t *testing.T) {
	t.Parallel()

	v := requiredifvalidator.RequireTenantForCrossNamespaceShared()
	targetPath := path.Root("ref").AtListIndex(0).AtName("tenant")

	t.Run("case-a-shared-to-shared-tenant-omitted", func(t *testing.T) {
		t.Parallel()
		cfg, plan := buildRefConfigAndPlan(t, tftypes.NewValue(tftypes.String, "shared"), tftypes.NewValue(tftypes.String, "shared"), tftypes.NewValue(tftypes.String, nil))

		valResp := &validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{
			Path:        targetPath,
			Config:      cfg,
			ConfigValue: types.StringNull(),
		}, valResp)
		assert.False(t, valResp.Diagnostics.HasError())

		planResp := &planmodifier.StringResponse{PlanValue: types.StringUnknown()}
		v.PlanModifyString(context.Background(), planmodifier.StringRequest{
			Path:        targetPath,
			Config:      cfg,
			ConfigValue: types.StringNull(),
			Plan:        plan,
			PlanValue:   types.StringUnknown(),
		}, planResp)
		assert.False(t, planResp.Diagnostics.HasError())
	})

	t.Run("case-b-shared-to-shared-tenant-set", func(t *testing.T) {
		t.Parallel()
		cfg, plan := buildRefConfigAndPlan(t, tftypes.NewValue(tftypes.String, "shared"), tftypes.NewValue(tftypes.String, "shared"), tftypes.NewValue(tftypes.String, "ves-io"))

		valResp := &validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{
			Path:        targetPath,
			Config:      cfg,
			ConfigValue: types.StringValue("ves-io"),
		}, valResp)
		assert.False(t, valResp.Diagnostics.HasError())

		planResp := &planmodifier.StringResponse{PlanValue: types.StringValue("ves-io")}
		v.PlanModifyString(context.Background(), planmodifier.StringRequest{
			Path:        targetPath,
			Config:      cfg,
			ConfigValue: types.StringValue("ves-io"),
			Plan:        plan,
			PlanValue:   types.StringValue("ves-io"),
		}, planResp)
		assert.False(t, planResp.Diagnostics.HasError())
	})

	t.Run("case-c-static-non-shared-to-shared-tenant-omitted", func(t *testing.T) {
		t.Parallel()
		cfg, _ := buildRefConfigAndPlan(t, tftypes.NewValue(tftypes.String, "advp-repro-ns"), tftypes.NewValue(tftypes.String, "shared"), tftypes.NewValue(tftypes.String, nil))

		valResp := &validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{
			Path:        targetPath,
			Config:      cfg,
			ConfigValue: types.StringNull(),
		}, valResp)
		require.True(t, valResp.Diagnostics.HasError())
		assert.Contains(t, valResp.Diagnostics.Errors()[0].Detail(), "ves-io")
		assert.Contains(t, valResp.Diagnostics.Errors()[0].Detail(), "advp-repro-ns")
	})

	t.Run("case-c-dynamic-unknown-root-ns-in-config-known-in-plan", func(t *testing.T) {
		t.Parallel()
		cfgUnknownRoot, _ := buildRefConfigAndPlan(t, tftypes.NewValue(tftypes.String, tftypes.UnknownValue), tftypes.NewValue(tftypes.String, "shared"), tftypes.NewValue(tftypes.String, nil))
		_, planResolvedRoot := buildRefConfigAndPlan(t, tftypes.NewValue(tftypes.String, "advp-repro-ns"), tftypes.NewValue(tftypes.String, "shared"), tftypes.NewValue(tftypes.String, tftypes.UnknownValue))

		// Phase 1 (ValidateString): root.namespace is Unknown in Config -> defers without error
		valResp := &validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{
			Path:        targetPath,
			Config:      cfgUnknownRoot,
			ConfigValue: types.StringNull(),
		}, valResp)
		assert.False(t, valResp.Diagnostics.HasError())

		// Phase 2 (PlanModifyString): root.namespace is resolved to "advp-repro-ns" in Plan while ConfigValue is Null -> catches error!
		planResp := &planmodifier.StringResponse{PlanValue: types.StringUnknown()}
		v.PlanModifyString(context.Background(), planmodifier.StringRequest{
			Path:        targetPath,
			Config:      cfgUnknownRoot,
			ConfigValue: types.StringNull(),
			Plan:        planResolvedRoot,
			PlanValue:   types.StringUnknown(),
		}, planResp)
		require.True(t, planResp.Diagnostics.HasError())
		assert.Contains(t, planResp.Diagnostics.Errors()[0].Detail(), "ves-io")
		assert.Contains(t, planResp.Diagnostics.Errors()[0].Detail(), "advp-repro-ns")
	})

	t.Run("case-c2-non-shared-to-shared-tenant-set", func(t *testing.T) {
		t.Parallel()
		cfg, plan := buildRefConfigAndPlan(t, tftypes.NewValue(tftypes.String, "advp-repro-ns2"), tftypes.NewValue(tftypes.String, "shared"), tftypes.NewValue(tftypes.String, "ves-io"))

		valResp := &validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{
			Path:        targetPath,
			Config:      cfg,
			ConfigValue: types.StringValue("ves-io"),
		}, valResp)
		assert.False(t, valResp.Diagnostics.HasError())

		planResp := &planmodifier.StringResponse{PlanValue: types.StringValue("ves-io")}
		v.PlanModifyString(context.Background(), planmodifier.StringRequest{
			Path:        targetPath,
			Config:      cfg,
			ConfigValue: types.StringValue("ves-io"),
			Plan:        plan,
			PlanValue:   types.StringValue("ves-io"),
		}, planResp)
		assert.False(t, planResp.Diagnostics.HasError())
	})

	t.Run("case-c-non-shared-to-shared-tenant-empty-string", func(t *testing.T) {
		t.Parallel()
		cfg, plan := buildRefConfigAndPlan(t, tftypes.NewValue(tftypes.String, "advp-repro-ns"), tftypes.NewValue(tftypes.String, "shared"), tftypes.NewValue(tftypes.String, ""))

		valResp := &validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{
			Path:        targetPath,
			Config:      cfg,
			ConfigValue: types.StringValue(""),
		}, valResp)
		require.True(t, valResp.Diagnostics.HasError())
		assert.Contains(t, valResp.Diagnostics.Errors()[0].Detail(), "ves-io")

		planResp := &planmodifier.StringResponse{PlanValue: types.StringValue("")}
		v.PlanModifyString(context.Background(), planmodifier.StringRequest{
			Path:        targetPath,
			Config:      cfg,
			ConfigValue: types.StringValue(""),
			Plan:        plan,
			PlanValue:   types.StringValue(""),
		}, planResp)
		require.True(t, planResp.Diagnostics.HasError())
		assert.Contains(t, planResp.Diagnostics.Errors()[0].Detail(), "ves-io")
	})

	t.Run("case-c-non-shared-to-shared-tenant-whitespace-only", func(t *testing.T) {
		t.Parallel()
		cfg, plan := buildRefConfigAndPlan(t, tftypes.NewValue(tftypes.String, "advp-repro-ns"), tftypes.NewValue(tftypes.String, "shared"), tftypes.NewValue(tftypes.String, "   "))

		valResp := &validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{
			Path:        targetPath,
			Config:      cfg,
			ConfigValue: types.StringValue("   "),
		}, valResp)
		require.True(t, valResp.Diagnostics.HasError())
		assert.Contains(t, valResp.Diagnostics.Errors()[0].Detail(), "ves-io")

		planResp := &planmodifier.StringResponse{PlanValue: types.StringValue("   ")}
		v.PlanModifyString(context.Background(), planmodifier.StringRequest{
			Path:        targetPath,
			Config:      cfg,
			ConfigValue: types.StringValue("   "),
			Plan:        plan,
			PlanValue:   types.StringValue("   "),
		}, planResp)
		require.True(t, planResp.Diagnostics.HasError())
		assert.Contains(t, planResp.Diagnostics.Errors()[0].Detail(), "ves-io")
	})

	t.Run("same-namespace-non-shared-ref-tenant-omitted", func(t *testing.T) {
		t.Parallel()
		cfg, plan := buildRefConfigAndPlan(t, tftypes.NewValue(tftypes.String, "advp-repro-ns"), tftypes.NewValue(tftypes.String, "advp-repro-ns"), tftypes.NewValue(tftypes.String, nil))

		valResp := &validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{
			Path:        targetPath,
			Config:      cfg,
			ConfigValue: types.StringNull(),
		}, valResp)
		assert.False(t, valResp.Diagnostics.HasError())

		planResp := &planmodifier.StringResponse{PlanValue: types.StringUnknown()}
		v.PlanModifyString(context.Background(), planmodifier.StringRequest{
			Path:        targetPath,
			Config:      cfg,
			ConfigValue: types.StringNull(),
			Plan:        plan,
			PlanValue:   types.StringUnknown(),
		}, planResp)
		assert.False(t, planResp.Diagnostics.HasError())
	})

	t.Run("destroy-plan-skipped", func(t *testing.T) {
		t.Parallel()
		cfg, plan := buildRefConfigAndPlan(t, tftypes.NewValue(tftypes.String, "advp-repro-ns"), tftypes.NewValue(tftypes.String, "shared"), tftypes.NewValue(tftypes.String, nil))
		plan.Raw = tftypes.NewValue(plan.Raw.Type(), nil)

		planResp := &planmodifier.StringResponse{}
		v.PlanModifyString(context.Background(), planmodifier.StringRequest{
			Path:        targetPath,
			Config:      cfg,
			ConfigValue: types.StringNull(),
			Plan:        plan,
		}, planResp)
		assert.False(t, planResp.Diagnostics.HasError())
	})

	t.Run("description-and-markdown-description", func(t *testing.T) {
		t.Parallel()
		assert.NotEmpty(t, v.Description(context.Background()))
		assert.NotEmpty(t, v.MarkdownDescription(context.Background()))
	})
}

// buildRefConfigAndPlan constructs a synthetic tfsdk.Config and tfsdk.Plan containing
// root "namespace" and a nested "ref" list with "name", "namespace", and "tenant" attributes.
//
// Why: Enables isolated unit testing of CrossNamespaceSharedTenantValidator across both
// ValidateString and PlanModifyString without spinning up a full provider server.
//
// How: Defines a minimal TPF resource schema matching ObjectRefType list attributes and
// populates tftypes.Value structures for both tfsdk.Config and tfsdk.Plan.
//
// Parameters:
//   - t:        *testing.T test runner handle.
//   - rootNS:   tftypes.Value for the root "namespace" attribute.
//   - refNS:    tftypes.Value for "ref[0].namespace".
//   - tenantVal: tftypes.Value for "ref[0].tenant".
//
// Returns:
//   - tfsdk.Config: Constructed configuration object.
//   - tfsdk.Plan:   Constructed plan object.
func buildRefConfigAndPlan(
	t *testing.T,
	rootNS tftypes.Value,
	refNS tftypes.Value,
	tenantVal tftypes.Value,
) (tfsdk.Config, tfsdk.Plan) {
	t.Helper()

	s := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"namespace": schema.StringAttribute{Required: true},
			"ref": schema.ListNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":      schema.StringAttribute{Optional: true},
						"namespace": schema.StringAttribute{Optional: true},
						"tenant":    schema.StringAttribute{Optional: true, Computed: true},
					},
				},
			},
		},
	}

	refElemType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"name":      tftypes.String,
			"namespace": tftypes.String,
			"tenant":    tftypes.String,
		},
	}
	rootType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"namespace": tftypes.String,
			"ref":       tftypes.List{ElementType: refElemType},
		},
	}

	raw := tftypes.NewValue(rootType, map[string]tftypes.Value{
		"namespace": rootNS,
		"ref": tftypes.NewValue(tftypes.List{ElementType: refElemType}, []tftypes.Value{
			tftypes.NewValue(refElemType, map[string]tftypes.Value{
				"name":      tftypes.NewValue(tftypes.String, "public"),
				"namespace": refNS,
				"tenant":    tenantVal,
			}),
		}),
	})

	return tfsdk.Config{Raw: raw, Schema: s}, tfsdk.Plan{Raw: raw, Schema: s}
}

// TestCrossNamespaceSharedTenant_SkipCases verifies that the tenant check is
// skipped when the tenant is unknown or when the sibling or root namespace
// cannot be read.
func TestCrossNamespaceSharedTenant_SkipCases(t *testing.T) {
	v := requiredifvalidator.RequireTenantForCrossNamespaceShared()
	tenantPath := path.Root("ref").AtListIndex(0).AtName("tenant")

	refElemType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"namespace": tftypes.String,
		"tenant":    tftypes.String,
	}}
	refListType := tftypes.List{ElementType: refElemType}
	noRootNSConfig := tfsdk.Config{
		Schema: schema.Schema{Attributes: map[string]schema.Attribute{
			"ref": schema.ListNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"namespace": schema.StringAttribute{Optional: true},
					"tenant":    schema.StringAttribute{Optional: true},
				}},
			},
		}},
		Raw: tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{"ref": refListType}}, map[string]tftypes.Value{
			"ref": tftypes.NewValue(refListType, []tftypes.Value{
				tftypes.NewValue(refElemType, map[string]tftypes.Value{
					"namespace": tftypes.NewValue(tftypes.String, "shared"),
					"tenant":    tftypes.NewValue(tftypes.String, nil),
				}),
			}),
		}),
	}
	sharedCfg, _ := buildRefConfigAndPlan(t,
		tftypes.NewValue(tftypes.String, "app"),
		tftypes.NewValue(tftypes.String, "shared"),
		tftypes.NewValue(tftypes.String, nil))

	tests := []struct {
		name   string
		path   path.Path
		cfg    tfsdk.Config
		tenant types.String
	}{
		{"tenant_unknown", tenantPath, sharedCfg, types.StringUnknown()},
		{"sibling_namespace_unreadable", path.Root("missing").AtName("tenant"), sharedCfg, types.StringNull()},
		{"root_namespace_unreadable", tenantPath, noRootNSConfig, types.StringNull()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), validator.StringRequest{
				Path:        tt.path,
				Config:      tt.cfg,
				ConfigValue: tt.tenant,
			}, resp)

			if resp.Diagnostics.HasError() {
				t.Errorf("unexpected error: %v", resp.Diagnostics)
			}
		})
	}
}
