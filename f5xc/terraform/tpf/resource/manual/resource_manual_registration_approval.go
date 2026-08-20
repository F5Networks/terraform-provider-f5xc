//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package manual

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gopkg.volterra.us/stdlib/codec"

	ves_io_schema "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema"
	ves_io_schema_registration "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/registration"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/client"
	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/utils"
)

const (
	registrationDefaultNamespace = "system"

	registrationGetRPCFQN = "ves.io.schema.registration.CustomAPI.Get"
	registrationGetURI    = "/public/namespaces/%s/registrations/%s"

	registrationListByStateRPCFQN = "ves.io.schema.registration.CustomAPI.ListRegistrationsByState"
	registrationListByStateURI    = "/public/namespaces/%s/listregistrationsbystate"

	registrationApproveRPCFQN = "ves.io.schema.registration.CustomAPI.RegistrationApprove"
	registrationApproveURI    = "/public/namespaces/%s/registration/%s/approve"
)

var registrationSearchStates = []ves_io_schema_registration.ObjectState{
	ves_io_schema_registration.PENDING,
	ves_io_schema_registration.APPROVED,
}

type registrationApprovalModel struct {
	Namespace          types.String   `tfsdk:"namespace"`
	ClusterName        types.String   `tfsdk:"cluster_name"`
	ClusterSize        types.Int64    `tfsdk:"cluster_size"`
	Hostname           types.String   `tfsdk:"hostname"`
	Retry              types.Int64    `tfsdk:"retry"`
	WaitTime           types.Int64    `tfsdk:"wait_time"`
	ConnectedRegion    types.String   `tfsdk:"connected_region"`
	Latitude           types.Float64  `tfsdk:"latitude"`
	Longitude          types.Float64  `tfsdk:"longitude"`
	TunnelType         types.List     `tfsdk:"tunnel_type"`
	PrivateNetworkName types.String   `tfsdk:"private_network_name"`
	RegistrationName   types.String   `tfsdk:"registration_name"`
	Timeouts           timeouts.Value `tfsdk:"timeouts"`
}

// ResourceF5XCRegistrationApproval defines manual action-style resource behavior.
type ResourceF5XCRegistrationApproval struct {
	client *client.APIClient
}

var (
	_ resource.Resource                   = &ResourceF5XCRegistrationApproval{}
	_ resource.ResourceWithConfigure      = &ResourceF5XCRegistrationApproval{}
	_ resource.ResourceWithModifyPlan     = &ResourceF5XCRegistrationApproval{}
	_ resource.ResourceWithImportState    = &ResourceF5XCRegistrationApproval{}
	_ resource.ResourceWithValidateConfig = &ResourceF5XCRegistrationApproval{}
)

func NewResourceF5XCRegistrationApproval() resource.Resource {
	return &ResourceF5XCRegistrationApproval{}
}

func (r *ResourceF5XCRegistrationApproval) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_registration_approval"
}

func (r *ResourceF5XCRegistrationApproval) Schema(ctx context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description:         "Registration approval action resource",
		MarkdownDescription: "Registration approval action resource",
		Attributes: map[string]schema.Attribute{
			"namespace": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(registrationDefaultNamespace),
				Description:         "Namespace used for registration approval operations.",
				MarkdownDescription: "Namespace used for registration approval operations.",
			},
			"cluster_name": schema.StringAttribute{
				Required:            true,
				Description:         "Cluster name used to locate the registration.",
				MarkdownDescription: "Cluster name used to locate the registration.",
			},
			"cluster_size": schema.Int64Attribute{
				Required:            true,
				Description:         "Cluster size override applied to the registration passport.",
				MarkdownDescription: "Cluster size override applied to the registration passport.",
			},
			"hostname": schema.StringAttribute{
				Required:            true,
				Description:         "Hostname used to locate the registration.",
				MarkdownDescription: "Hostname used to locate the registration.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"retry": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1),
				Description:         "Number of polling attempts while locating registrations.",
				MarkdownDescription: "Number of polling attempts while locating registrations.",
			},
			"wait_time": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1),
				Description:         "Delay in seconds between polling attempts.",
				MarkdownDescription: "Delay in seconds between polling attempts.",
			},
			"connected_region": schema.StringAttribute{
				Optional:            true,
				Description:         "Connected region override applied during approval.",
				MarkdownDescription: "Connected region override applied during approval.",
			},
			"latitude": schema.Float64Attribute{
				Optional:            true,
				Description:         "Latitude override applied during approval.",
				MarkdownDescription: "Latitude override applied during approval.",
			},
			"longitude": schema.Float64Attribute{
				Optional:            true,
				Description:         "Longitude override applied during approval.",
				MarkdownDescription: "Longitude override applied during approval.",
			},
			"tunnel_type": schema.ListAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				Default:             listdefault.StaticValue(utils.GetListStringDefaultValue("SITE_TO_SITE_TUNNEL_IPSEC_OR_SSL")),
				Description:         "Tunnel type used during registration approval.",
				MarkdownDescription: "Tunnel type used during registration approval.",
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						stringvalidator.OneOf(
							"SITE_TO_SITE_TUNNEL_IPSEC_OR_SSL",
							"SITE_TO_SITE_TUNNEL_IPSEC",
							"SITE_TO_SITE_TUNNEL_SSL",
						),
					),
					listvalidator.SizeAtMost(1),
				},
			},
			"private_network_name": schema.StringAttribute{
				Optional:            true,
				Description:         "Private network name override applied during approval.",
				MarkdownDescription: "Private network name override applied during approval.",
			},
			"registration_name": schema.StringAttribute{
				Computed:            true,
				Description:         "Resolved registration name approved by this resource.",
				MarkdownDescription: "Resolved registration name approved by this resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

func (r *ResourceF5XCRegistrationApproval) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	apiClient, ok := request.ProviderData.(*client.APIClient)
	if !ok {
		response.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *APIClient, got: %T. Please report this issue to the provider developers.", request.ProviderData),
		)
		return
	}

	r.client = apiClient
}

func (r *ResourceF5XCRegistrationApproval) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data registrationApprovalModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ResourceF5XCRegistrationApproval) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		resp.Diagnostics.AddWarning(
			"Resource Destruction",
			"This will retire the matching registration from F5 XC.",
		)
		return
	}

	if req.State.Raw.IsNull() {
		var plan registrationApprovalModel
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}

func (r *ResourceF5XCRegistrationApproval) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data registrationApprovalModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	createTimeout, ok := resolveTimeout(ctx, data.Timeouts, timeoutSelectorCreate, "10m0s", &response.Diagnostics)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	resolvedName, resolvedPassport, resolvedState, err := resolveRegistrationForApproval(
		ctx,
		r.client,
		normalizeNamespace(data.Namespace.ValueString()),
		data.ClusterName.ValueString(),
		data.Hostname.ValueString(),
		data.Retry.ValueInt64(),
		data.WaitTime.ValueInt64(),
	)
	if err != nil {
		response.Diagnostics.AddError("Resolve Registration Error", err.Error())
		return
	}

	data.Namespace = types.StringValue(normalizeNamespace(data.Namespace.ValueString()))
	data.RegistrationName = types.StringValue(resolvedName)

	if resolvedState != ves_io_schema_registration.APPROVED {
		req, reqErr := buildApprovalRequest(data, resolvedName, resolvedPassport, ves_io_schema_registration.APPROVED)
		if reqErr != nil {
			response.Diagnostics.AddError("Build Approval Request Error", reqErr.Error())
			return
		}

		if err = runRegistrationApproval(ctx, r.client, req); err != nil {
			response.Diagnostics.AddError("Registration Approve Error", err.Error())
			return
		}
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *ResourceF5XCRegistrationApproval) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data registrationApprovalModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	readTimeout, ok := resolveTimeout(ctx, data.Timeouts, timeoutSelectorRead, "5m0s", &response.Diagnostics)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	registrationName := strings.TrimSpace(data.RegistrationName.ValueString())
	if registrationName == "" {
		response.State.RemoveResource(ctx)
		return
	}

	namespace := normalizeNamespace(data.Namespace.ValueString())
	data.Namespace = types.StringValue(namespace)

	req := &ves_io_schema_registration.GetRequest{
		Namespace: namespace,
		Name:      registrationName,
	}

	yamlReq, err := codec.ToYAML(req)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("Error marshalling read request to yaml: %v", err))
		return
	}

	_, err = r.client.CustomAPI(
		ctx,
		http.MethodGet,
		fmt.Sprintf(registrationGetURI, namespace, registrationName),
		registrationGetRPCFQN,
		yamlReq,
	)
	if err != nil {
		if isStatusNotFound(err) {
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read registration_approval, got error: %s", err))
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *ResourceF5XCRegistrationApproval) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data registrationApprovalModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	updateTimeout, ok := resolveTimeout(ctx, data.Timeouts, timeoutSelectorUpdate, "10m0s", &response.Diagnostics)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	resolvedName, resolvedPassport, resolvedState, err := resolveRegistrationForApproval(
		ctx,
		r.client,
		normalizeNamespace(data.Namespace.ValueString()),
		data.ClusterName.ValueString(),
		data.Hostname.ValueString(),
		data.Retry.ValueInt64(),
		data.WaitTime.ValueInt64(),
	)
	if err != nil {
		response.Diagnostics.AddError("Resolve Registration Error", err.Error())
		return
	}

	data.Namespace = types.StringValue(normalizeNamespace(data.Namespace.ValueString()))
	data.RegistrationName = types.StringValue(resolvedName)

	if resolvedState != ves_io_schema_registration.APPROVED {
		req, reqErr := buildApprovalRequest(data, resolvedName, resolvedPassport, ves_io_schema_registration.APPROVED)
		if reqErr != nil {
			response.Diagnostics.AddError("Build Approval Request Error", reqErr.Error())
			return
		}

		if err = runRegistrationApproval(ctx, r.client, req); err != nil {
			response.Diagnostics.AddError("Registration Approve Error", err.Error())
			return
		}
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *ResourceF5XCRegistrationApproval) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data registrationApprovalModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	deleteTimeout, ok := resolveTimeout(ctx, data.Timeouts, timeoutSelectorDelete, "10m0s", &response.Diagnostics)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	resolvedName := strings.TrimSpace(data.RegistrationName.ValueString())
	if resolvedName == "" {
		return
	}

	req := &ves_io_schema_registration.ApprovalReq{
		Namespace: normalizeNamespace(data.Namespace.ValueString()),
		Name:      resolvedName,
		State:     ves_io_schema_registration.RETIRED,
		Passport: &ves_io_schema_registration.Passport{
			ClusterName: data.ClusterName.ValueString(),
			ClusterSize: int32(data.ClusterSize.ValueInt64()),
		},
	}

	err := runRegistrationApproval(ctx, r.client, req)
	if err != nil && !isStatusNotFound(err) {
		response.Diagnostics.AddError("Registration Retire Error", err.Error())
		return
	}
}

func (r *ResourceF5XCRegistrationApproval) ImportState(ctx context.Context, req resource.ImportStateRequest, response *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		response.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: namespace/registration_name, got: %s", req.ID),
		)
		return
	}

	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("namespace"), parts[0])...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("registration_name"), parts[1])...)
}

func normalizeNamespace(namespace string) string {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return registrationDefaultNamespace
	}

	return namespace
}

func normalizeRetryAttempts(retry int64) int64 {
	if retry < 1 {
		return 1
	}

	return retry
}

func normalizeWaitInterval(waitTimeSeconds int64) time.Duration {
	if waitTimeSeconds < 30 || waitTimeSeconds > 120 {
		waitTimeSeconds = 60
	}

	return time.Duration(waitTimeSeconds) * time.Second
}

func resolveRegistrationForApproval(
	ctx context.Context,
	apiClient *client.APIClient,
	namespace string,
	clusterName string,
	hostname string,
	retry int64,
	waitTimeSeconds int64,
) (string, *ves_io_schema_registration.Passport, ves_io_schema_registration.ObjectState, error) {
	if apiClient == nil {
		return "", nil, ves_io_schema_registration.NOTSET, fmt.Errorf("resolveRegistrationForApproval: api client is required")
	}

	trimmedClusterName := strings.TrimSpace(clusterName)
	trimmedHostname := strings.TrimSpace(hostname)
	if trimmedClusterName == "" {
		return "", nil, ves_io_schema_registration.NOTSET, fmt.Errorf("resolveRegistrationForApproval: cluster_name is required")
	}
	if trimmedHostname == "" {
		return "", nil, ves_io_schema_registration.NOTSET, fmt.Errorf("resolveRegistrationForApproval: hostname is required")
	}

	normalizedNS := normalizeNamespace(namespace)
	attempts := normalizeRetryAttempts(retry)
	pollInterval := normalizeWaitInterval(waitTimeSeconds)

	for attempt := int64(0); attempt < attempts; attempt++ {
		for _, stateToSearch := range registrationSearchStates {
			item, err := findRegistrationByClusterAndHostname(ctx, apiClient, normalizedNS, stateToSearch, trimmedClusterName, trimmedHostname)
			if err != nil {
				return "", nil, ves_io_schema_registration.NOTSET, err
			}
			if item == nil {
				continue
			}

			resolvedName := strings.TrimSpace(item.GetName())
			if resolvedName == "" {
				resolvedName = strings.TrimSpace(item.GetMetadata().GetName())
			}
			if resolvedName == "" {
				return "", nil, ves_io_schema_registration.NOTSET, fmt.Errorf("resolveRegistrationForApproval: matched registration has empty name")
			}

			passport := extractItemPassport(item)
			if passport == nil {
				return "", nil, ves_io_schema_registration.NOTSET, fmt.Errorf("resolveRegistrationForApproval: matched registration has empty passport")
			}

			return resolvedName, passport, stateToSearch, nil
		}

		if attempt == attempts-1 {
			break
		}

		if errSleep := sleepWithContext(ctx, pollInterval); errSleep != nil {
			return "", nil, ves_io_schema_registration.NOTSET, fmt.Errorf("resolveRegistrationForApproval: context done while waiting for registration: %w", errSleep)
		}
	}

	return "", nil, ves_io_schema_registration.NOTSET, fmt.Errorf(
		"resolveRegistrationForApproval: registration not found for cluster_name=%q hostname=%q in namespace=%q after %d attempt(s)",
		clusterName,
		hostname,
		normalizedNS,
		attempts,
	)
}

func findRegistrationByClusterAndHostname(
	ctx context.Context,
	apiClient *client.APIClient,
	namespace string,
	stateToSearch ves_io_schema_registration.ObjectState,
	clusterName string,
	hostname string,
) (*ves_io_schema_registration.ListResponseItem, error) {
	listReq := &ves_io_schema_registration.ListStateReq{
		Namespace: namespace,
		State:     stateToSearch,
	}

	yamlReq, err := codec.ToYAML(listReq)
	if err != nil {
		return nil, fmt.Errorf("findRegistrationByClusterAndHostname: failed to marshal list request: %w", err)
	}

	listRespMsg, err := apiClient.CustomAPI(
		ctx,
		http.MethodPost,
		fmt.Sprintf(registrationListByStateURI, namespace),
		registrationListByStateRPCFQN,
		yamlReq,
	)
	if err != nil {
		return nil, fmt.Errorf("findRegistrationByClusterAndHostname: list by state call failed: %w", err)
	}

	listResp, ok := listRespMsg.(*ves_io_schema_registration.ListResponse)
	if !ok {
		return nil, fmt.Errorf("findRegistrationByClusterAndHostname: unexpected response type %T", listRespMsg)
	}

	wantCluster := strings.TrimSpace(clusterName)
	wantHostname := normalizeHostname(hostname)
	clusterMatches := make([]*ves_io_schema_registration.ListResponseItem, 0)
	clusterHostnames := make([]string, 0)

	for _, item := range listResp.GetItems() {
		if item == nil {
			continue
		}

		itemCluster, itemHostnameRaw := extractItemClusterAndHostname(item)
		itemHostname := normalizeHostname(itemHostnameRaw)

		if !strings.EqualFold(itemCluster, wantCluster) {
			continue
		}

		clusterMatches = append(clusterMatches, item)
		clusterHostnames = append(clusterHostnames, strings.TrimSpace(itemHostnameRaw))

		if itemHostname == wantHostname {
			return item, nil
		}
	}

	if len(clusterMatches) == 1 {
		return clusterMatches[0], nil
	}
	if len(clusterMatches) > 1 {
		return nil, fmt.Errorf("findRegistrationByClusterAndHostname: found %d registrations in state=%q for cluster_name=%q but none matched hostname=%q (available hostnames: %s)", len(clusterMatches), stateToSearch.String(), clusterName, hostname, strings.Join(clusterHostnames, ", "))
	}

	return nil, nil
}

func extractItemClusterAndHostname(item *ves_io_schema_registration.ListResponseItem) (string, string) {
	if item == nil {
		return "", ""
	}

	cluster := strings.TrimSpace(item.GetGetSpec().GetPassport().GetClusterName())
	hostname := strings.TrimSpace(item.GetGetSpec().GetInfra().GetHostname())

	if cluster == "" || hostname == "" {
		gcSpec := item.GetObject().GetSpec().GetGcSpec()
		if cluster == "" {
			cluster = strings.TrimSpace(gcSpec.GetPassport().GetClusterName())
		}
		if hostname == "" {
			hostname = strings.TrimSpace(gcSpec.GetInfra().GetHostname())
		}
	}

	return cluster, hostname
}

func extractItemPassport(item *ves_io_schema_registration.ListResponseItem) *ves_io_schema_registration.Passport {
	if item == nil {
		return nil
	}

	if item.GetGetSpec() != nil {
		if passport := item.GetGetSpec().GetPassport(); passport != nil {
			return passport
		}
	}

	if item.GetObject() != nil && item.GetObject().GetSpec() != nil {
		return item.GetObject().GetSpec().GetGcSpec().GetPassport()
	}

	return nil
}

func normalizeHostname(hostname string) string {
	trimmed := strings.ToLower(strings.TrimSpace(hostname))
	if idx := strings.Index(trimmed, "."); idx > 0 {
		return trimmed[:idx]
	}

	return trimmed
}

func buildApprovalRequest(
	data registrationApprovalModel,
	registrationName string,
	basePassport *ves_io_schema_registration.Passport,
	state ves_io_schema_registration.ObjectState,
) (*ves_io_schema_registration.ApprovalReq, error) {
	if basePassport == nil {
		return nil, fmt.Errorf("buildApprovalRequest: base passport is required")
	}

	req := &ves_io_schema_registration.ApprovalReq{
		Namespace: normalizeNamespace(data.Namespace.ValueString()),
		Name:      registrationName,
		State:     state,
		Passport: &ves_io_schema_registration.Passport{
			ClusterName:        basePassport.GetClusterName(),
			ClusterType:        basePassport.GetClusterType(),
			Latitude:           basePassport.GetLatitude(),
			Longitude:          basePassport.GetLongitude(),
			ClusterSize:        basePassport.GetClusterSize(),
			VpmVersion:         basePassport.GetVpmVersion(),
			PrivateNetworkName: basePassport.GetPrivateNetworkName(),
		},
	}

	if data.ClusterSize.ValueInt64() > 0 {
		req.Passport.ClusterSize = int32(data.ClusterSize.ValueInt64())
	}

	if !data.PrivateNetworkName.IsNull() && !data.PrivateNetworkName.IsUnknown() {
		if privateNetworkName := strings.TrimSpace(data.PrivateNetworkName.ValueString()); privateNetworkName != "" {
			req.Passport.PrivateNetworkName = privateNetworkName
		}
	}

	if !data.Latitude.IsNull() && !data.Latitude.IsUnknown() {
		req.Passport.Latitude = float32(data.Latitude.ValueFloat64())
	}

	if !data.Longitude.IsNull() && !data.Longitude.IsUnknown() {
		req.Passport.Longitude = float32(data.Longitude.ValueFloat64())
	}

	if !data.ConnectedRegion.IsNull() && !data.ConnectedRegion.IsUnknown() {
		req.ConnectedRegion = strings.TrimSpace(data.ConnectedRegion.ValueString())
	}

	if !data.TunnelType.IsNull() && !data.TunnelType.IsUnknown() {
		var tunnelValues []types.String
		tunnelDiags := data.TunnelType.ElementsAs(context.Background(), &tunnelValues, false)
		if tunnelDiags.HasError() {
			return nil, fmt.Errorf("buildApprovalRequest: unable to decode tunnel_type list")
		}

		if len(tunnelValues) > 0 && !tunnelValues[0].IsNull() && !tunnelValues[0].IsUnknown() {
			tunnelType := strings.TrimSpace(tunnelValues[0].ValueString())
			if tunnelType != "" {
				mappedTunnel, ok := ves_io_schema.SiteToSiteTunnelType_value[tunnelType]
				if !ok {
					return nil, fmt.Errorf("buildApprovalRequest: invalid tunnel_type %q", tunnelType)
				}
				req.TunnelType = ves_io_schema.SiteToSiteTunnelType(mappedTunnel)
			}
		}
	}

	return req, nil
}

func runRegistrationApproval(ctx context.Context, apiClient *client.APIClient, req *ves_io_schema_registration.ApprovalReq) error {
	if apiClient == nil {
		return fmt.Errorf("runRegistrationApproval: api client is required")
	}
	if req == nil {
		return fmt.Errorf("runRegistrationApproval: approval request is required")
	}

	resolvedName := strings.TrimSpace(req.GetName())
	if resolvedName == "" {
		return fmt.Errorf("runRegistrationApproval: registration name is required")
	}

	resolvedNamespace := normalizeNamespace(req.GetNamespace())
	req.Namespace = resolvedNamespace

	yamlReq, err := codec.ToYAML(req)
	if err != nil {
		return fmt.Errorf("runRegistrationApproval: failed to marshal approval request: %w", err)
	}

	_, err = apiClient.CustomAPI(
		ctx,
		http.MethodPost,
		fmt.Sprintf(registrationApproveURI, resolvedNamespace, resolvedName),
		registrationApproveRPCFQN,
		yamlReq,
	)
	if err != nil {
		return fmt.Errorf("runRegistrationApproval: custom API call failed: %w", err)
	}

	return nil
}

type timeoutSelector int

const (
	timeoutSelectorCreate timeoutSelector = iota
	timeoutSelectorRead
	timeoutSelectorUpdate
	timeoutSelectorDelete
)

func resolveTimeout(ctx context.Context, value timeouts.Value, selector timeoutSelector, fallback string, diags *diag.Diagnostics) (time.Duration, bool) {
	duration, err := time.ParseDuration(fallback)
	if err != nil {
		diags.AddError(
			"Invalid Duration Format",
			fmt.Sprintf("Could not parse 'timeout' value %q: %s.", fallback, err.Error()),
		)
		return 0, false
	}

	var (
		resolved time.Duration
		d        diag.Diagnostics
	)

	switch selector {
	case timeoutSelectorCreate:
		resolved, d = value.Create(ctx, duration)
	case timeoutSelectorRead:
		resolved, d = value.Read(ctx, duration)
	case timeoutSelectorUpdate:
		resolved, d = value.Update(ctx, duration)
	case timeoutSelectorDelete:
		resolved, d = value.Delete(ctx, duration)
	default:
		diags.AddError("Timeout Error", "Unknown timeout selector")
		return 0, false
	}

	diags.Append(d...)
	if diags.HasError() {
		return 0, false
	}

	return resolved, true
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func isStatusNotFound(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "status code 404")
}
