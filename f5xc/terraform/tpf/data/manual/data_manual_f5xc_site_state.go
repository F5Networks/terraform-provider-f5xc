//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package manual

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/client"
	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/utils"

	ves_io_schema_site "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/site"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &DataSourceF5XCSiteState{}
	_ datasource.DataSourceWithConfigure = &DataSourceF5XCSiteState{}
)

// DataSourceF5XCSiteState implements the manual data source for querying Customer Edge (CE) site operational state.
type DataSourceF5XCSiteState struct {
	client *client.APIClient
}

// SiteNodeModel represents individual node operational information within a site.
type SiteNodeModel struct {
	Name       types.String `tfsdk:"name"`
	SloAddress types.String `tfsdk:"slo_address"`
	SliAddress types.String `tfsdk:"sli_address"`
}

// DataSourceF5XCSiteStateModel maps the data source schema data to Go types.
type DataSourceF5XCSiteStateModel struct {
	Name                   types.String    `tfsdk:"name"`
	Namespace              types.String    `tfsdk:"namespace"`
	Id                     types.String    `tfsdk:"id"`
	SiteState              types.String    `tfsdk:"site_state"`
	IsOnline               types.Bool      `tfsdk:"is_online"`
	SiteType               types.String    `tfsdk:"site_type"`
	SoftwareVersion        types.String    `tfsdk:"software_version"`
	OperatingSystemVersion types.String    `tfsdk:"operating_system_version"`
	Nodes                  []SiteNodeModel `tfsdk:"nodes"`
	Timeouts               timeouts.Value  `tfsdk:"timeouts"`
}

// NewDataSourceF5XCSiteState returns a new instance of DataSourceF5XCSiteState.
//
// Why: Factory constructor required by Terraform Plugin Framework to register the data source.
//
// How: Allocates and returns a pointer to an empty DataSourceF5XCSiteState struct.
//
// Parameters:
//   - none
//
// Returns:
//   - datasource.DataSource: the data source implementation
//
// Example:
//
//	ds := NewDataSourceF5XCSiteState()
func NewDataSourceF5XCSiteState() datasource.DataSource {
	return &DataSourceF5XCSiteState{}
}

// Metadata sets the provider-prefixed data source type name.
//
// Why: Informs Terraform core of the resource/data source type name for HCL resolution.
//
// How: Combines request.ProviderTypeName with "_site_state".
//
// Parameters:
//   - ctx: context for cancellation/tracing
//   - request: metadata request from framework
//   - response: metadata response to populate
//
// Returns:
//   - none
//
// Example:
//
//	d.Metadata(ctx, req, resp) // sets resp.TypeName = "f5xc_site_state"
func (d *DataSourceF5XCSiteState) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_site_state"
}

// Schema defines the schema attributes for the site_state data source.
//
// Why: Declares all input and computed attributes so Terraform can validate HCL and construct the state model.
//
// How: Builds a schema.Schema struct with String, Bool, ListNested, and Timeout attributes.
//
// Parameters:
//   - ctx: context for cancellation/tracing
//   - request: schema request from framework
//   - response: schema response to populate
//
// Returns:
//   - none
//
// Example:
//
//	d.Schema(ctx, req, resp)
func (d *DataSourceF5XCSiteState) Schema(ctx context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Data source for retrieving the operational readiness and state of an F5 Distributed Cloud Customer Edge (CE) site.",
		MarkdownDescription: "Data source for retrieving the operational readiness and state of an F5 Distributed Cloud Customer Edge (CE) site. " +
			"This manual data source queries the core site object directly in namespace `system`, allowing automation workflows to verify whether a CE is `ONLINE`.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description:         "Name of the site configuration object in namespace 'system'.",
				MarkdownDescription: "Name of the site configuration object. It must match the name of the provisioned site (e.g. AWS VPC site, SecureMesh site).",
				Required:            true,
			},
			"namespace": schema.StringAttribute{
				Description:         "Namespace of the site. Defaults to 'system'.",
				MarkdownDescription: "Namespace of the site object. Site objects reside in namespace `system`. Defaults to `system`.",
				Optional:            true,
				Computed:            true,
			},
			"id": schema.StringAttribute{
				Description:         "Unique identifier for the site, formatted as '<namespace>/<name>'.",
				MarkdownDescription: "Unique identifier for the site, formatted as `<namespace>/<name>`.",
				Computed:            true,
			},
			"site_state": schema.StringAttribute{
				Description:         "Operational state of the CE site (e.g., ONLINE, PROVISIONING, UPGRADING, FAILED).",
				MarkdownDescription: "Operational state of the CE site reported by the platform. Possible values include `ONLINE`, `PROVISIONING`, `UPGRADING`, `STANDBY`, `FAILED`, `REREGISTRATION`, `WAITINGNODES`, `DECOMMISSIONING`, `WAITING_FOR_REGISTRATION`, `ORCHESTRATION_IN_PROGRESS`, `ORCHESTRATION_COMPLETE`, `ERROR_IN_ORCHESTRATION`, `DELETING_CLOUD_RESOURCES`.",
				Computed:            true,
			},
			"is_online": schema.BoolAttribute{
				Description:         "Convenience boolean flag indicating whether the site state is ONLINE.",
				MarkdownDescription: "Convenience boolean flag that evaluates to `true` when `site_state` is `ONLINE`. Useful for conditional checks in automation workflows.",
				Computed:            true,
			},
			"site_type": schema.StringAttribute{
				Description:         "Type of the site (e.g., CUSTOMER_EDGE, REGIONAL_EDGE).",
				MarkdownDescription: "Type of the site as registered in F5 Distributed Cloud (e.g. `CUSTOMER_EDGE`, `REGIONAL_EDGE`).",
				Computed:            true,
			},
			"software_version": schema.StringAttribute{
				Description:         "F5 Distributed Cloud software version running on the site.",
				MarkdownDescription: "F5 Distributed Cloud software version currently active on the site.",
				Computed:            true,
			},
			"operating_system_version": schema.StringAttribute{
				Description:         "Underlying operating system version running on the site.",
				MarkdownDescription: "Underlying operating system version running on the site.",
				Computed:            true,
			},
			"nodes": schema.ListNestedAttribute{
				Description:         "List of master/main nodes belonging to the site.",
				MarkdownDescription: "List of master/main nodes belonging to the site.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description:         "Name of the node.",
							MarkdownDescription: "Name of the node.",
							Computed:            true,
						},
						"slo_address": schema.StringAttribute{
							Description:         "Site Local Outside (SLO) IP address.",
							MarkdownDescription: "Site Local Outside (SLO) IP address.",
							Computed:            true,
						},
						"sli_address": schema.StringAttribute{
							Description:         "Site Local Inside (SLI) IP address.",
							MarkdownDescription: "Site Local Inside (SLI) IP address.",
							Computed:            true,
						},
					},
				},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Read: true,
			}),
		},
	}
}

// Configure supplies the API client from provider data.
//
// Why: Provides access to the authenticated API client configured at the provider level.
//
// How: Casts request.ProviderData to *client.APIClient and stores it on the struct.
//
// Parameters:
//   - ctx: context for cancellation/tracing
//   - request: configure request from framework
//   - response: configure response to populate diagnostics
//
// Returns:
//   - none
//
// Example:
//
//	d.Configure(ctx, req, resp)
func (d *DataSourceF5XCSiteState) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	apiClient, ok := request.ProviderData.(*client.APIClient)
	if !ok {
		response.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.APIClient, got: %T. Please report this issue to provider developers.", request.ProviderData),
		)
		return
	}

	d.client = apiClient
}

// Read queries the F5 XC core site object and populates the Terraform state.
//
// Why: Fulfills Terraform data source read operations, extracting the live operational state of the CE site.
//
// How: Resolves the namespace (defaulting to "system"), applies the read timeout, invokes APIClient.GetObject
// for ves_io_schema_site.ObjectType, unpacks SpecType.GcSpec, and sets the computed attributes on the model.
//
// Parameters:
//   - ctx: context for cancellation/tracing
//   - request: read request containing configuration
//   - response: read response to populate with state and diagnostics
//
// Returns:
//   - none
//
// Example:
//
//	d.Read(ctx, req, resp)
func (d *DataSourceF5XCSiteState) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data DataSourceF5XCSiteStateModel

	// Read Terraform configuration data into the model
	response.Diagnostics.Append(request.Config.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	namespace := data.Namespace.ValueString()
	if namespace == "" {
		namespace = "system"
	}
	name := data.Name.ValueString()

	// Apply read timeout if configured, defaulting to 5 minutes
	readTimeout, diags := data.Timeouts.Read(ctx, 5*time.Minute)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	tflog.Debug(ctx, fmt.Sprintf("Reading site state for site %s/%s", namespace, name))

	httpResp, err := d.client.GetObject(ctx, ves_io_schema_site.ObjectType, namespace, name)
	if err != nil {
		response.Diagnostics.AddError("Client Error", utils.SanitizeAPIError(ctx, "read", "site_state", err))
		return
	}

	specData, ok := httpResp.GetObjSpec().(*ves_io_schema_site.SpecType)
	if !ok || specData == nil || specData.GcSpec == nil {
		response.Diagnostics.AddError("Parsing Error", fmt.Sprintf("Unable to extract site specification for site %s/%s", namespace, name))
		return
	}

	gcSpec := specData.GcSpec
	siteStateStr := gcSpec.GetSiteState().String()

	data.Name = types.StringValue(name)
	data.Namespace = types.StringValue(namespace)
	data.Id = types.StringValue(fmt.Sprintf("%s/%s", namespace, name))
	data.SiteState = types.StringValue(siteStateStr)
	data.IsOnline = types.BoolValue(siteStateStr == "ONLINE")
	data.SiteType = types.StringValue(gcSpec.GetSiteType().String())
	data.SoftwareVersion = types.StringValue(gcSpec.GetVolterraSoftwareVersion())
	data.OperatingSystemVersion = types.StringValue(gcSpec.GetOperatingSystemVersion())

	// Populate nodes
	mainNodes := gcSpec.GetMainNodes()
	nodesList := make([]SiteNodeModel, 0, len(mainNodes))
	for _, n := range mainNodes {
		if n == nil {
			continue
		}
		nodesList = append(nodesList, SiteNodeModel{
			Name:       types.StringValue(n.GetName()),
			SloAddress: types.StringValue(n.GetSloAddress()),
			SliAddress: types.StringValue(n.GetSliAddress()),
		})
	}
	data.Nodes = nodesList

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}
