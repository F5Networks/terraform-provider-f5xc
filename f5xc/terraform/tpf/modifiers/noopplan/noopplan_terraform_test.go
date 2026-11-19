//
// Copyright (c) 2026 F5 Inc. All rights reserved.
//

//go:build acceptance

package noopplan_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/modifiers/noopplan"
)

// These tests run real Terraform against an in-memory provider (no F5 XC API), so core's
// planning of Optional+Computed nested attributes is exercised end to end.

var (
	refAttrTypes    = map[string]attr.Type{"name": types.StringType}
	refType         = types.ObjectType{AttrTypes: refAttrTypes}
	choiceAttrTypes = map[string]attr.Type{"flag": types.BoolType}
)

// TestTerraform_CascadeOnlyPlanIsEmpty: a backend-defaulted Optional+Computed object makes
// core propose null on every plan; noopplan must keep post-apply plans empty.
func TestTerraform_CascadeOnlyPlanIsEmpty(t *testing.T) {
	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []tfresource.TestStep{
			{
				Config: `resource "noopplantest_thing" "example" { name = "a" }`,
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr("noopplantest_thing.example", "choice.flag", "true"),
					tfresource.TestCheckResourceAttr("noopplantest_thing.example", "status", "ready"),
				),
			},
			{
				Config: `resource "noopplantest_thing" "example" { name = "b" }`,
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("noopplantest_thing.example", plancheck.ResourceActionUpdate),
					},
				},
			},
		},
	})
}

// TestTerraform_RemovedBlockPlansUpdate: removing a configured Optional+Computed block must
// plan an update (fingerprint guard), not be collapsed into "No changes".
func TestTerraform_RemovedBlockPlansUpdate(t *testing.T) {
	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []tfresource.TestStep{
			{
				Config: `resource "noopplantest_thing" "example" {
  name = "a"
  refs = [{ name = "r1" }]
}`,
				Check: tfresource.TestCheckResourceAttr("noopplantest_thing.example", "refs.0.name", "r1"),
			},
			{
				Config: `resource "noopplantest_thing" "example" { name = "a" }`,
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("noopplantest_thing.example", plancheck.ResourceActionUpdate),
					},
				},
				Check: tfresource.TestCheckNoResourceAttr("noopplantest_thing.example", "refs.0.name"),
			},
		},
	})
}

func testProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"noopplantest": providerserver.NewProtocol6WithError(&testProvider{}),
	}
}

type testProvider struct{}

func (p *testProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "noopplantest"
}

func (p *testProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = providerschema.Schema{}
}

func (p *testProvider) Configure(_ context.Context, _ provider.ConfigureRequest, _ *provider.ConfigureResponse) {
}

func (p *testProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

func (p *testProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{func() resource.Resource { return &thingResource{} }}
}

type thingModel struct {
	ID     types.String `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Refs   types.List   `tfsdk:"refs"`
	Choice types.Object `tfsdk:"choice"`
	Status types.String `tfsdk:"status"`
}

// thingResource mimics a generated resource: Optional+Computed nested attributes without
// plan modifiers, a Computed status without UseStateForUnknown, and noopplan wiring.
type thingResource struct{}

func (r *thingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_thing"
}

func (r *thingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"name": schema.StringAttribute{Required: true},
		"refs": schema.ListNestedAttribute{
			Optional: true,
			Computed: true,
			NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{Required: true},
			}},
		},
		"choice": schema.SingleNestedAttribute{
			Optional: true,
			Computed: true,
			Attributes: map[string]schema.Attribute{
				"flag": schema.BoolAttribute{Optional: true},
			},
		},
		"status": schema.StringAttribute{Computed: true},
	}}
}

func (r *thingResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	noopplan.Apply(ctx, req, resp)
}

func (r *thingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	defer noopplan.RecordConfig(ctx, req.Config, resp.Private)
	var m thingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	fillBackendValues(&m)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *thingResource) Read(_ context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) {}

func (r *thingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	defer noopplan.RecordConfig(ctx, req.Config, resp.Private)
	var m thingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	fillBackendValues(&m)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *thingResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

// fillBackendValues plays the backend: unsent refs are cleared, an unsent choice gets its
// default branch, and status is always server-set.
func fillBackendValues(m *thingModel) {
	if m.ID.IsUnknown() {
		m.ID = types.StringValue("thing")
	}
	if m.Refs.IsUnknown() {
		m.Refs = types.ListNull(refType)
	}
	if m.Choice.IsUnknown() || m.Choice.IsNull() {
		m.Choice = types.ObjectValueMust(choiceAttrTypes, map[string]attr.Value{"flag": types.BoolValue(true)})
	}
	m.Status = types.StringValue("ready")
}
