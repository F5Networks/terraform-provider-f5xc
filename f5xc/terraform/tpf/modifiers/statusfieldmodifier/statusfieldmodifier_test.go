//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

package statusfieldmodifier_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/modifiers/statusfieldmodifier"
)

type testAdvertiseChoiceModel struct {
	DoNotAdvertise    types.Bool `tfsdk:"do_not_advertise"`
	AdvertiseOnPublic types.Bool `tfsdk:"advertise_on_public"`
}

type testDNSInfoItem struct {
	IPAddress types.String `tfsdk:"ip_address"`
}

type testResourceModel struct {
	HostName        types.String              `tfsdk:"host_name"`
	DNSInfo         []testDNSInfoItem         `tfsdk:"dns_info"`
	AutoCertInfo    *testAutoCertInfoModel    `tfsdk:"auto_cert_info"`
	AdvertiseChoice *testAdvertiseChoiceModel `tfsdk:"advertise_choice"`
}

type testAutoCertInfoModel struct {
	AutoCertState types.String `tfsdk:"auto_cert_state"`
}

func testSchema() schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host_name": schema.StringAttribute{
				Computed: true,
			},
			"dns_info": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip_address": schema.StringAttribute{
							Optional: true,
						},
					},
				},
			},
			"auto_cert_info": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"auto_cert_state": schema.StringAttribute{Computed: true},
				},
			},
			"advertise_choice": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"do_not_advertise": schema.BoolAttribute{
						Optional: true,
					},
					"advertise_on_public": schema.BoolAttribute{
						Optional: true,
					},
				},
			},
		},
	}
}

func buildTestState(t *testing.T, model testResourceModel) tfsdk.State {
	s := tfsdk.State{Schema: testSchema()}
	diags := s.Set(context.Background(), &model)
	require.False(t, diags.HasError(), "failed to build test state: %v", diags)
	return s
}

func buildTestPlan(t *testing.T, model testResourceModel) tfsdk.Plan {
	s := tfsdk.State{Schema: testSchema()}
	diags := s.Set(context.Background(), &model)
	require.False(t, diags.HasError(), "failed to build test plan: %v", diags)
	return tfsdk.Plan(s)
}

func TestConstructorAndDescriptions(t *testing.T) {
	strMod := statusfieldmodifier.UseStateUnlessDependencyChangesString(path.Root("advertise_choice"))
	require.NotNil(t, strMod)
	assert.NotEmpty(t, strMod.Description(context.Background()))
	assert.Equal(t, strMod.Description(context.Background()), strMod.MarkdownDescription(context.Background()))

	listMod := statusfieldmodifier.UseStateUnlessDependencyChangesList(path.Root("advertise_choice"))
	require.NotNil(t, listMod)
	assert.NotEmpty(t, listMod.Description(context.Background()))
	assert.Equal(t, listMod.Description(context.Background()), listMod.MarkdownDescription(context.Background()))

	objectMod := statusfieldmodifier.UseStateUnlessDependencyChangesObject(path.Root("advertise_choice"))
	require.NotNil(t, objectMod)
	assert.NotEmpty(t, objectMod.Description(context.Background()))
	assert.Equal(t, objectMod.Description(context.Background()), objectMod.MarkdownDescription(context.Background()))
}

func TestUseCase1_UnchangedDependency_Object(t *testing.T) {
	ctx := context.Background()
	mod := statusfieldmodifier.UseStateUnlessDependencyChangesObject(path.Root("advertise_choice"))
	stateModel := testResourceModel{
		AutoCertInfo:    &testAutoCertInfoModel{AutoCertState: types.StringValue("AutoCertStarted")},
		AdvertiseChoice: &testAdvertiseChoiceModel{AdvertiseOnPublic: types.BoolValue(true)},
	}
	state := buildTestState(t, stateModel)
	plan := buildTestPlan(t, testResourceModel{AdvertiseChoice: stateModel.AdvertiseChoice})

	var stateValue types.Object
	diags := state.GetAttribute(ctx, path.Root("auto_cert_info"), &stateValue)
	require.False(t, diags.HasError(), "failed to read prior object state: %v", diags)
	attrTypes := stateValue.AttributeTypes(ctx)
	req := planmodifier.ObjectRequest{
		Path:        path.Root("auto_cert_info"),
		State:       state,
		Plan:        plan,
		StateValue:  stateValue,
		PlanValue:   types.ObjectUnknown(attrTypes),
		ConfigValue: types.ObjectNull(attrTypes),
	}
	resp := &planmodifier.ObjectResponse{PlanValue: req.PlanValue}

	mod.PlanModifyObject(ctx, req, resp)

	assert.False(t, resp.PlanValue.IsUnknown(), "expected prior object state to be preserved")
	assert.True(t, resp.PlanValue.Equal(stateValue), "expected PlanValue to equal StateValue")
}

func TestUseCase2_DependencyChanged_Object(t *testing.T) {
	ctx := context.Background()
	mod := statusfieldmodifier.UseStateUnlessDependencyChangesObject(path.Root("advertise_choice"))
	stateModel := testResourceModel{
		AutoCertInfo:    &testAutoCertInfoModel{AutoCertState: types.StringValue("AutoCertStarted")},
		AdvertiseChoice: &testAdvertiseChoiceModel{AdvertiseOnPublic: types.BoolValue(true)},
	}
	state := buildTestState(t, stateModel)
	plan := buildTestPlan(t, testResourceModel{
		AdvertiseChoice: &testAdvertiseChoiceModel{DoNotAdvertise: types.BoolValue(true)},
	})

	var stateValue types.Object
	diags := state.GetAttribute(ctx, path.Root("auto_cert_info"), &stateValue)
	require.False(t, diags.HasError(), "failed to read prior object state: %v", diags)
	attrTypes := stateValue.AttributeTypes(ctx)
	req := planmodifier.ObjectRequest{
		Path:        path.Root("auto_cert_info"),
		State:       state,
		Plan:        plan,
		StateValue:  stateValue,
		PlanValue:   types.ObjectUnknown(attrTypes),
		ConfigValue: types.ObjectNull(attrTypes),
	}
	resp := &planmodifier.ObjectResponse{PlanValue: req.PlanValue}

	mod.PlanModifyObject(ctx, req, resp)

	assert.True(t, resp.PlanValue.IsUnknown(), "expected changed dependency to keep object plan unknown")
}

func TestInitialCreation_Object(t *testing.T) {
	ctx := context.Background()
	mod := statusfieldmodifier.UseStateUnlessDependencyChangesObject(path.Root("advertise_choice"))
	attrTypes := map[string]attr.Type{"auto_cert_state": types.StringType}
	req := planmodifier.ObjectRequest{
		State:       tfsdk.State{Raw: tftypes.NewValue(tftypes.Object{}, nil), Schema: testSchema()},
		StateValue:  types.ObjectNull(attrTypes),
		PlanValue:   types.ObjectUnknown(attrTypes),
		ConfigValue: types.ObjectNull(attrTypes),
	}
	resp := &planmodifier.ObjectResponse{PlanValue: req.PlanValue}

	mod.PlanModifyObject(ctx, req, resp)

	assert.True(t, resp.PlanValue.IsUnknown(), "create scenario must leave object plan unknown")
}

// -----------------------------------------------------------------------------
// USE CASE 1: Unchanged Dependency & Known Prior State -> Preserves State (No Drift)
// -----------------------------------------------------------------------------

func TestUseCase1_UnchangedDependency_String(t *testing.T) {
	ctx := context.Background()
	mod := statusfieldmodifier.UseStateUnlessDependencyChangesString(path.Root("advertise_choice"))

	adv := &testAdvertiseChoiceModel{AdvertiseOnPublic: types.BoolValue(true)}
	stateModel := testResourceModel{
		HostName:        types.StringValue("ves-io-test.volterra.us"),
		AdvertiseChoice: adv,
	}
	planModel := testResourceModel{
		HostName:        types.StringUnknown(),
		AdvertiseChoice: adv,
	}

	state := buildTestState(t, stateModel)
	plan := buildTestPlan(t, planModel)

	req := planmodifier.StringRequest{
		Path:        path.Root("host_name"),
		State:       state,
		Plan:        plan,
		StateValue:  types.StringValue("ves-io-test.volterra.us"),
		PlanValue:   types.StringUnknown(),
		ConfigValue: types.StringNull(),
	}
	resp := &planmodifier.StringResponse{
		PlanValue: req.PlanValue,
	}

	mod.PlanModifyString(ctx, req, resp)

	assert.False(t, resp.PlanValue.IsUnknown(), "expected PlanValue to be known")
	assert.Equal(t, "ves-io-test.volterra.us", resp.PlanValue.ValueString())
}

func TestUseCase1_UnchangedDependency_List(t *testing.T) {
	ctx := context.Background()
	mod := statusfieldmodifier.UseStateUnlessDependencyChangesList(path.Root("advertise_choice"))

	adv := &testAdvertiseChoiceModel{AdvertiseOnPublic: types.BoolValue(true)}
	stateModel := testResourceModel{
		DNSInfo:         []testDNSInfoItem{{IPAddress: types.StringValue("72.19.3.36")}},
		AdvertiseChoice: adv,
	}
	planModel := testResourceModel{
		AdvertiseChoice: adv,
	}

	state := buildTestState(t, stateModel)
	plan := buildTestPlan(t, planModel)

	var stateList types.List
	diags := state.GetAttribute(ctx, path.Root("dns_info"), &stateList)
	require.False(t, diags.HasError())

	req := planmodifier.ListRequest{
		Path:        path.Root("dns_info"),
		State:       state,
		Plan:        plan,
		StateValue:  stateList,
		PlanValue:   types.ListUnknown(stateList.ElementType(ctx)),
		ConfigValue: types.ListNull(stateList.ElementType(ctx)),
	}
	resp := &planmodifier.ListResponse{
		PlanValue: req.PlanValue,
	}

	mod.PlanModifyList(ctx, req, resp)

	assert.False(t, resp.PlanValue.IsUnknown(), "expected PlanValue to be known")
	assert.True(t, resp.PlanValue.Equal(stateList), "expected PlanValue to equal StateValue")
}

// -----------------------------------------------------------------------------
// USE CASE 2: Unchanged Dependency & Null Prior State -> Preserves State (Null)
// -----------------------------------------------------------------------------

func TestUseCase2_UnchangedDependency_NullPriorState_String(t *testing.T) {
	ctx := context.Background()
	mod := statusfieldmodifier.UseStateUnlessDependencyChangesString(path.Root("advertise_choice"))

	adv := &testAdvertiseChoiceModel{DoNotAdvertise: types.BoolValue(true)}
	stateModel := testResourceModel{
		HostName:        types.StringNull(),
		AdvertiseChoice: adv,
	}
	planModel := testResourceModel{
		HostName:        types.StringUnknown(),
		AdvertiseChoice: adv,
	}

	state := buildTestState(t, stateModel)
	plan := buildTestPlan(t, planModel)

	req := planmodifier.StringRequest{
		Path:        path.Root("host_name"),
		State:       state,
		Plan:        plan,
		StateValue:  types.StringNull(),
		PlanValue:   types.StringUnknown(),
		ConfigValue: types.StringNull(),
	}
	resp := &planmodifier.StringResponse{
		PlanValue: req.PlanValue,
	}

	mod.PlanModifyString(ctx, req, resp)

	assert.False(t, resp.PlanValue.IsUnknown(), "expected PlanValue not to be Unknown when dependency is unchanged")
	assert.True(t, resp.PlanValue.IsNull(), "expected PlanValue to preserve prior Null state")
}

func TestUseCase2_UnchangedDependency_NullPriorState_List(t *testing.T) {
	ctx := context.Background()
	mod := statusfieldmodifier.UseStateUnlessDependencyChangesList(path.Root("advertise_choice"))

	adv := &testAdvertiseChoiceModel{DoNotAdvertise: types.BoolValue(true)}
	stateModel := testResourceModel{
		AdvertiseChoice: adv,
	}
	planModel := testResourceModel{
		AdvertiseChoice: adv,
	}

	state := buildTestState(t, stateModel)
	plan := buildTestPlan(t, planModel)

	var stateList types.List
	diags := state.GetAttribute(ctx, path.Root("dns_info"), &stateList)
	require.False(t, diags.HasError())

	req := planmodifier.ListRequest{
		Path:        path.Root("dns_info"),
		State:       state,
		Plan:        plan,
		StateValue:  types.ListNull(stateList.ElementType(ctx)),
		PlanValue:   types.ListUnknown(stateList.ElementType(ctx)),
		ConfigValue: types.ListNull(stateList.ElementType(ctx)),
	}
	resp := &planmodifier.ListResponse{
		PlanValue: req.PlanValue,
	}

	mod.PlanModifyList(ctx, req, resp)

	assert.False(t, resp.PlanValue.IsUnknown(), "expected PlanValue not to be Unknown when dependency is unchanged")
	assert.True(t, resp.PlanValue.IsNull(), "expected PlanValue to preserve prior Null state")
}

// -----------------------------------------------------------------------------
// USE CASE 3: Dependency Changed -> Remains/Transitions to Unknown
// -----------------------------------------------------------------------------

func TestUseCase3_DependencyChanged_DeAdvertise_String(t *testing.T) {
	ctx := context.Background()
	mod := statusfieldmodifier.UseStateUnlessDependencyChangesString(path.Root("advertise_choice"))

	// Prior: advertise on public -> host_name was populated
	stateAdv := &testAdvertiseChoiceModel{AdvertiseOnPublic: types.BoolValue(true)}
	// Plan: changed to do_not_advertise
	planAdv := &testAdvertiseChoiceModel{DoNotAdvertise: types.BoolValue(true)}

	stateModel := testResourceModel{
		HostName:        types.StringValue("ves-io-test.volterra.us"),
		AdvertiseChoice: stateAdv,
	}
	planModel := testResourceModel{
		HostName:        types.StringUnknown(),
		AdvertiseChoice: planAdv,
	}

	state := buildTestState(t, stateModel)
	plan := buildTestPlan(t, planModel)

	req := planmodifier.StringRequest{
		Path:        path.Root("host_name"),
		State:       state,
		Plan:        plan,
		StateValue:  types.StringValue("ves-io-test.volterra.us"),
		PlanValue:   types.StringUnknown(),
		ConfigValue: types.StringNull(),
	}
	resp := &planmodifier.StringResponse{
		PlanValue: req.PlanValue,
	}

	mod.PlanModifyString(ctx, req, resp)

	assert.True(t, resp.PlanValue.IsUnknown(), "expected PlanValue to be Unknown when advertise_choice changed to do_not_advertise")
}

func TestUseCase3_DependencyChanged_DeAdvertise_List(t *testing.T) {
	ctx := context.Background()
	mod := statusfieldmodifier.UseStateUnlessDependencyChangesList(path.Root("advertise_choice"))

	stateAdv := &testAdvertiseChoiceModel{AdvertiseOnPublic: types.BoolValue(true)}
	planAdv := &testAdvertiseChoiceModel{DoNotAdvertise: types.BoolValue(true)}

	stateModel := testResourceModel{
		DNSInfo:         []testDNSInfoItem{{IPAddress: types.StringValue("72.19.3.36")}},
		AdvertiseChoice: stateAdv,
	}
	planModel := testResourceModel{
		AdvertiseChoice: planAdv,
	}

	state := buildTestState(t, stateModel)
	plan := buildTestPlan(t, planModel)

	var stateList types.List
	diags := state.GetAttribute(ctx, path.Root("dns_info"), &stateList)
	require.False(t, diags.HasError())

	req := planmodifier.ListRequest{
		Path:        path.Root("dns_info"),
		State:       state,
		Plan:        plan,
		StateValue:  stateList,
		PlanValue:   types.ListUnknown(stateList.ElementType(ctx)),
		ConfigValue: types.ListNull(stateList.ElementType(ctx)),
	}
	resp := &planmodifier.ListResponse{
		PlanValue: req.PlanValue,
	}

	mod.PlanModifyList(ctx, req, resp)

	assert.True(t, resp.PlanValue.IsUnknown(), "expected PlanValue to be Unknown when advertise_choice changed to do_not_advertise")
}

func TestUseCase3_DependencyChanged_ActivateAdvertise_String(t *testing.T) {
	ctx := context.Background()
	mod := statusfieldmodifier.UseStateUnlessDependencyChangesString(path.Root("advertise_choice"))

	stateAdv := &testAdvertiseChoiceModel{DoNotAdvertise: types.BoolValue(true)}
	planAdv := &testAdvertiseChoiceModel{AdvertiseOnPublic: types.BoolValue(true)}

	stateModel := testResourceModel{
		HostName:        types.StringNull(),
		AdvertiseChoice: stateAdv,
	}
	planModel := testResourceModel{
		HostName:        types.StringUnknown(),
		AdvertiseChoice: planAdv,
	}

	state := buildTestState(t, stateModel)
	plan := buildTestPlan(t, planModel)

	req := planmodifier.StringRequest{
		Path:        path.Root("host_name"),
		State:       state,
		Plan:        plan,
		StateValue:  types.StringNull(),
		PlanValue:   types.StringUnknown(),
		ConfigValue: types.StringNull(),
	}
	resp := &planmodifier.StringResponse{
		PlanValue: req.PlanValue,
	}

	mod.PlanModifyString(ctx, req, resp)

	assert.True(t, resp.PlanValue.IsUnknown(), "expected PlanValue to be Unknown when advertise_choice changed from do_not_advertise to public")
}

func TestUseCase3_DependencyChanged_ActivateAdvertise_List(t *testing.T) {
	ctx := context.Background()
	mod := statusfieldmodifier.UseStateUnlessDependencyChangesList(path.Root("advertise_choice"))

	stateAdv := &testAdvertiseChoiceModel{DoNotAdvertise: types.BoolValue(true)}
	planAdv := &testAdvertiseChoiceModel{AdvertiseOnPublic: types.BoolValue(true)}

	stateModel := testResourceModel{
		AdvertiseChoice: stateAdv,
	}
	planModel := testResourceModel{
		AdvertiseChoice: planAdv,
	}

	state := buildTestState(t, stateModel)
	plan := buildTestPlan(t, planModel)

	var stateList types.List
	diags := state.GetAttribute(ctx, path.Root("dns_info"), &stateList)
	require.False(t, diags.HasError())

	req := planmodifier.ListRequest{
		Path:        path.Root("dns_info"),
		State:       state,
		Plan:        plan,
		StateValue:  types.ListNull(stateList.ElementType(ctx)),
		PlanValue:   types.ListUnknown(stateList.ElementType(ctx)),
		ConfigValue: types.ListNull(stateList.ElementType(ctx)),
	}
	resp := &planmodifier.ListResponse{
		PlanValue: req.PlanValue,
	}

	mod.PlanModifyList(ctx, req, resp)

	assert.True(t, resp.PlanValue.IsUnknown(), "expected PlanValue to be Unknown when advertise_choice changed from do_not_advertise to public")
}

// -----------------------------------------------------------------------------
// Edge Cases: Create, Interpolation Guard, Known Plan
// -----------------------------------------------------------------------------

func TestCreateScenario_StateRawIsNull(t *testing.T) {
	ctx := context.Background()
	mod := statusfieldmodifier.UseStateUnlessDependencyChangesString(path.Root("advertise_choice"))

	rawNullState := tfsdk.State{
		Raw:    tftypes.NewValue(tftypes.Object{}, nil),
		Schema: testSchema(),
	}

	req := planmodifier.StringRequest{
		Path:        path.Root("host_name"),
		State:       rawNullState,
		PlanValue:   types.StringUnknown(),
		ConfigValue: types.StringNull(),
	}
	resp := &planmodifier.StringResponse{
		PlanValue: req.PlanValue,
	}

	mod.PlanModifyString(ctx, req, resp)

	assert.True(t, resp.PlanValue.IsUnknown(), "create scenario must leave plan value as Unknown")
}

func TestConfigUnknown_InterpolationGuard(t *testing.T) {
	ctx := context.Background()
	mod := statusfieldmodifier.UseStateUnlessDependencyChangesString(path.Root("advertise_choice"))

	req := planmodifier.StringRequest{
		Path:        path.Root("host_name"),
		StateValue:  types.StringValue("prior"),
		PlanValue:   types.StringUnknown(),
		ConfigValue: types.StringUnknown(),
	}
	resp := &planmodifier.StringResponse{
		PlanValue: req.PlanValue,
	}

	mod.PlanModifyString(ctx, req, resp)

	assert.True(t, resp.PlanValue.IsUnknown(), "unknown config must leave plan value unchanged")
}

func TestKnownPlanValue_Unchanged(t *testing.T) {
	ctx := context.Background()
	mod := statusfieldmodifier.UseStateUnlessDependencyChangesString(path.Root("advertise_choice"))

	req := planmodifier.StringRequest{
		Path:        path.Root("host_name"),
		StateValue:  types.StringValue("prior"),
		PlanValue:   types.StringValue("explicit-plan"),
		ConfigValue: types.StringNull(),
	}
	resp := &planmodifier.StringResponse{
		PlanValue: req.PlanValue,
	}

	mod.PlanModifyString(ctx, req, resp)

	assert.Equal(t, "explicit-plan", resp.PlanValue.ValueString())
}

var depSchema = schema.Schema{Attributes: map[string]schema.Attribute{
	"dep": schema.StringAttribute{Optional: true},
}}

var depObjectType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{"dep": tftypes.String}}

func depRaw(v any) tftypes.Value {
	return tftypes.NewValue(depObjectType, map[string]tftypes.Value{"dep": tftypes.NewValue(tftypes.String, v)})
}

func depState(v any) tfsdk.State { return tfsdk.State{Schema: depSchema, Raw: depRaw(v)} }
func depPlan(v any) tfsdk.Plan   { return tfsdk.Plan{Schema: depSchema, Raw: depRaw(v)} }

var depPath = path.Root("dep")

// dependencyCase describes the dependency value in state and plan and whether
// isDependencyChanged reports a change for that pair.
type dependencyCase struct {
	name        string
	dep         path.Path
	state, plan any
	changed     bool
}

var dependencyCases = []dependencyCase{
	{name: "empty path is unchanged", dep: path.Empty(), state: "a", plan: "b", changed: false},
	{name: "unreadable path is changed", dep: path.Root("missing"), state: "a", plan: "a", changed: true},
	{name: "unknown plan is changed", dep: depPath, state: "a", plan: tftypes.UnknownValue, changed: true},
	{name: "unknown state is changed", dep: depPath, state: tftypes.UnknownValue, plan: "a", changed: true},
	{name: "both null is unchanged", dep: depPath, state: nil, plan: nil, changed: false},
	{name: "null to value is changed", dep: depPath, state: nil, plan: "a", changed: true},
	{name: "equal values are unchanged", dep: depPath, state: "a", plan: "a", changed: false},
	{name: "different values are changed", dep: depPath, state: "a", plan: "b", changed: true},
}

// PlanModifyString keeps state when the dependency is unchanged and plans unknown when it changes.
func TestPlanModifyString_DependencyMatrix(t *testing.T) {
	for _, tc := range dependencyCases {
		t.Run(tc.name, func(t *testing.T) {
			m := statusfieldmodifier.UseStateUnlessDependencyChangesString(tc.dep)
			req := planmodifier.StringRequest{
				State:       depState(tc.state),
				Plan:        depPlan(tc.plan),
				StateValue:  types.StringValue("prior"),
				PlanValue:   types.StringUnknown(),
				ConfigValue: types.StringNull(),
			}
			// A known response value shows that the changed branch resets it to unknown.
			resp := &planmodifier.StringResponse{PlanValue: types.StringValue("stale")}
			m.PlanModifyString(context.Background(), req, resp)

			if tc.changed {
				assert.True(t, resp.PlanValue.IsUnknown())
			} else {
				assert.Equal(t, "prior", resp.PlanValue.ValueString())
			}
		})
	}
}

// PlanModifyString leaves the response untouched for a known plan or an unknown config.
func TestPlanModifyString_Guards(t *testing.T) {
	m := statusfieldmodifier.UseStateUnlessDependencyChangesString(depPath)
	base := planmodifier.StringRequest{
		State:       depState("a"),
		Plan:        depPlan("a"),
		StateValue:  types.StringValue("prior"),
		PlanValue:   types.StringUnknown(),
		ConfigValue: types.StringNull(),
	}

	known := base
	known.PlanValue = types.StringValue("known")
	resp := &planmodifier.StringResponse{PlanValue: known.PlanValue}
	m.PlanModifyString(context.Background(), known, resp)
	assert.Equal(t, "known", resp.PlanValue.ValueString())

	unknownCfg := base
	unknownCfg.ConfigValue = types.StringUnknown()
	resp = &planmodifier.StringResponse{PlanValue: types.StringUnknown()}
	m.PlanModifyString(context.Background(), unknownCfg, resp)
	assert.True(t, resp.PlanValue.IsUnknown())
}

// PlanModifyList takes the unknown element type from the plan, or from state when the plan has none.
func TestPlanModifyList_DependencyChanged(t *testing.T) {
	m := statusfieldmodifier.UseStateUnlessDependencyChangesList(depPath)
	stateList := types.ListValueMust(types.StringType, nil)

	tests := []struct {
		name        string
		planValue   types.List
		stateValue  types.List
		wantUnknown bool
	}{
		{name: "element type from plan", planValue: types.ListUnknown(types.StringType), stateValue: types.ListNull(types.StringType), wantUnknown: true},
		{name: "element type from state", planValue: types.ListUnknown(nil), stateValue: stateList, wantUnknown: true},
		{name: "no element type keeps response", planValue: types.ListUnknown(nil), stateValue: types.ListNull(nil), wantUnknown: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := planmodifier.ListRequest{
				State:       depState("a"),
				Plan:        depPlan("b"),
				StateValue:  tc.stateValue,
				PlanValue:   tc.planValue,
				ConfigValue: types.ListNull(types.StringType),
			}
			stale := types.ListValueMust(types.StringType, nil)
			resp := &planmodifier.ListResponse{PlanValue: stale}
			m.PlanModifyList(context.Background(), req, resp)

			if tc.wantUnknown {
				assert.True(t, resp.PlanValue.IsUnknown())
				assert.Equal(t, types.StringType, resp.PlanValue.ElementType(context.Background()))
			} else {
				assert.True(t, resp.PlanValue.Equal(stale))
			}
		})
	}
}

// PlanModifyList leaves the response untouched for a known plan or an unknown config.
func TestPlanModifyList_Guards(t *testing.T) {
	m := statusfieldmodifier.UseStateUnlessDependencyChangesList(depPath)
	base := planmodifier.ListRequest{
		State:       depState("a"),
		Plan:        depPlan("a"),
		StateValue:  types.ListNull(types.StringType),
		PlanValue:   types.ListUnknown(types.StringType),
		ConfigValue: types.ListNull(types.StringType),
	}

	known := base
	known.PlanValue = types.ListValueMust(types.StringType, nil)
	resp := &planmodifier.ListResponse{PlanValue: known.PlanValue}
	m.PlanModifyList(context.Background(), known, resp)
	assert.False(t, resp.PlanValue.IsUnknown())

	unknownCfg := base
	unknownCfg.ConfigValue = types.ListUnknown(types.StringType)
	resp = &planmodifier.ListResponse{PlanValue: base.PlanValue}
	m.PlanModifyList(context.Background(), unknownCfg, resp)
	assert.True(t, resp.PlanValue.IsUnknown())
}

// PlanModifyList does nothing during create, when prior state is null.
func TestPlanModifyList_Create(t *testing.T) {
	m := statusfieldmodifier.UseStateUnlessDependencyChangesList(depPath)
	req := planmodifier.ListRequest{
		State:     tfsdk.State{Schema: depSchema, Raw: tftypes.NewValue(depObjectType, nil)},
		PlanValue: types.ListUnknown(types.StringType),
	}
	resp := &planmodifier.ListResponse{PlanValue: req.PlanValue}
	m.PlanModifyList(context.Background(), req, resp)
	assert.True(t, resp.PlanValue.IsUnknown())
}
