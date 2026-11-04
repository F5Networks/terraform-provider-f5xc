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
