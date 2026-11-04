//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/client"
	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/utils"
)

type tcpLoadbalancerServicePolicyAttachmentsModel struct {
	ID                  types.String   `tfsdk:"id"`
	Namespace           types.String   `tfsdk:"namespace"`
	TcpLoadbalancerName types.String   `tfsdk:"tcp_loadbalancer_name"`
	ServicePolicies     types.List     `tfsdk:"service_policies"`
	Timeouts            timeouts.Value `tfsdk:"timeouts"`
}

// ResourceF5xcTcpLoadbalancerServicePolicyAttachments defines the attachment resource implementation for f5xc_tcp_loadbalancer_service_policy_attachments.
type ResourceF5xcTcpLoadbalancerServicePolicyAttachments struct {
	client *client.APIClient
}

var (
	_ resource.Resource                = &ResourceF5xcTcpLoadbalancerServicePolicyAttachments{}
	_ resource.ResourceWithConfigure   = &ResourceF5xcTcpLoadbalancerServicePolicyAttachments{}
	_ resource.ResourceWithImportState = &ResourceF5xcTcpLoadbalancerServicePolicyAttachments{}
)

func NewResourceF5xcTcpLoadbalancerServicePolicyAttachments() resource.Resource {
	return &ResourceF5xcTcpLoadbalancerServicePolicyAttachments{}
}

func (r *ResourceF5xcTcpLoadbalancerServicePolicyAttachments) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_tcp_loadbalancer_service_policy_attachments"
}

func (r *ResourceF5xcTcpLoadbalancerServicePolicyAttachments) Schema(ctx context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description:         "Resource to manage service policy attachments on a TCP LoadBalancer.",
		MarkdownDescription: "Resource to manage service policy attachments on a TCP LoadBalancer.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Unique identifier for the association in the format 'namespace/target_name'.",
				MarkdownDescription: "Unique identifier for the association in the format `namespace/target_name`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"namespace": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("default"),
				Description:         "Namespace of the target resource (defaults to 'default').",
				MarkdownDescription: "Namespace of the target resource (defaults to `default`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"tcp_loadbalancer_name": schema.StringAttribute{
				Required:            true,
				Description:         "Name of the target parent resource.",
				MarkdownDescription: "Name of the target parent resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service_policies": schema.ListAttribute{
				Required:            true,
				ElementType:         types.StringType,
				Description:         "Authoritative list of attached policy names.",
				MarkdownDescription: "Authoritative list of attached policy names.",
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

func (r *ResourceF5xcTcpLoadbalancerServicePolicyAttachments) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *ResourceF5xcTcpLoadbalancerServicePolicyAttachments) fetchTargetResource(ctx context.Context, namespace, targetName string) ([]byte, string, error) {
	urls := r.getTargetURLs(r.client.RestClient.URL, namespace, targetName)
	var lastErr error
	for _, url := range urls {
		respBytes, err := r.client.RestClient.Execute(ctx, http.MethodGet, url, nil)
		if err == nil {
			return respBytes, url, nil
		}
		lastErr = err
	}
	return nil, urls[0], lastErr
}

func (r *ResourceF5xcTcpLoadbalancerServicePolicyAttachments) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data tcpLoadbalancerServicePolicyAttachmentsModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	namespace := data.Namespace.ValueString()
	if namespace == "" {
		namespace = "default"
	}
	targetName := data.TcpLoadbalancerName.ValueString()

	policyNames, err := r.extractStringList(ctx, data.ServicePolicies)
	if err != nil {
		response.Diagnostics.AddError("Extract Attachment List Error", err.Error())
		return
	}

	unlock := lockAttachmentTarget("tcp_loadbalancer", namespace, targetName)
	defer unlock()

	tflog.Info(ctx, fmt.Sprintf("Setting attachments for tcp_loadbalancer %s in namespace %s: %v", targetName, namespace, policyNames))

	respBytes, targetURL, err := r.fetchTargetResource(ctx, namespace, targetName)
	if err != nil {
		response.Diagnostics.AddError(
			"Get Target Resource Error",
			fmt.Sprintf("Failed to fetch tcp_loadbalancer '%s' in namespace '%s' (URL: %s): %s.\nPlease ensure 'namespace' = '%s' matches the target resource's actual namespace.", targetName, namespace, targetURL, err.Error(), namespace),
		)
		return
	}

	var parentObj map[string]any
	if err := json.Unmarshal(respBytes, &parentObj); err != nil {
		response.Diagnostics.AddError("Unmarshal Target Resource Error", fmt.Sprintf("Failed to parse tcp_loadbalancer JSON: %s", err.Error()))
		return
	}

	spec, _ := parentObj["spec"].(map[string]any)
	if spec == nil {
		spec = make(map[string]any)
		parentObj["spec"] = spec
	}

	// Guardrail: Check if active policies are already configured on the target parent resource.
	// This prevents multiple attachment resources or an inline configuration from silently overwriting each other.
	if activeMap, ok := spec["active_service_policies"].(map[string]any); ok && activeMap != nil {
		if policies, ok := activeMap["policies"].([]any); ok && len(policies) > 0 {
			response.Diagnostics.AddError(
				"Attachment Conflict Error",
				fmt.Sprintf("Failed to create attachment resource for tcp_loadbalancer '%s' in namespace '%s': "+
					"active service policies are already configured on this tcp_loadbalancer. "+
					"Only one attachment resource may manage policies for a given parent resource, and inline policies must not coexist with attachment resources.",
					targetName, namespace),
			)
			return
		}
	}

	if len(policyNames) == 0 {
		delete(spec, "active_service_policies")
		spec["no_service_policies"] = map[string]any{}
	} else {
		var policiesRaw []any
		for _, spName := range policyNames {
			policiesRaw = append(policiesRaw, map[string]any{
				"name":      spName,
				"namespace": namespace,
			})
		}
		spec["active_service_policies"] = map[string]any{
			"policies": policiesRaw,
		}
		delete(spec, "no_service_policies")
	}

	updatedBody, err := json.Marshal(parentObj)
	if err != nil {
		response.Diagnostics.AddError("Marshal Target Resource Error", fmt.Sprintf("Failed to encode updated tcp_loadbalancer JSON: %s", err.Error()))
		return
	}

	_, err = r.client.RestClient.Execute(ctx, http.MethodPut, targetURL, updatedBody)
	if err != nil {
		response.Diagnostics.AddError("Update Target Resource Error", fmt.Sprintf("Failed to update tcp_loadbalancer %s with attachments: %s", targetName, err.Error()))
		return
	}

	data.Namespace = types.StringValue(namespace)
	data.ID = types.StringValue(fmt.Sprintf("%s/%s", namespace, targetName))
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *ResourceF5xcTcpLoadbalancerServicePolicyAttachments) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data tcpLoadbalancerServicePolicyAttachmentsModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	namespace := data.Namespace.ValueString()
	if namespace == "" {
		namespace = "default"
	}
	targetName := data.TcpLoadbalancerName.ValueString()

	respBytes, _, err := r.fetchTargetResource(ctx, namespace, targetName)
	if err != nil {
		if utils.IsNotFoundError(err) {
			tflog.Warn(ctx, fmt.Sprintf("Target resource tcp_loadbalancer %s not found in namespace %s: %s. Removing association from state.", targetName, namespace, err.Error()))
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError(
			"Fetch Target Resource Error",
			fmt.Sprintf("Failed to fetch tcp_loadbalancer '%s' in namespace '%s': %s", targetName, namespace, err.Error()),
		)
		return
	}

	var parentObj map[string]any
	if err := json.Unmarshal(respBytes, &parentObj); err != nil {
		response.Diagnostics.AddError("Unmarshal Target Resource Error", fmt.Sprintf("Failed to parse tcp_loadbalancer JSON: %s", err.Error()))
		return
	}

	spec, _ := parentObj["spec"].(map[string]any)
	activeChoice, _ := spec["active_service_policies"].(map[string]any)
	policiesRaw, _ := activeChoice["policies"].([]any)

	retrievedPolicies := make([]string, 0)
	for _, p := range policiesRaw {
		if pMap, ok := p.(map[string]any); ok {
			if spName, ok := pMap["name"].(string); ok && spName != "" {
				retrievedPolicies = append(retrievedPolicies, spName)
			}
		}
	}

	policiesListVal, diags := types.ListValueFrom(ctx, types.StringType, retrievedPolicies)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	data.Namespace = types.StringValue(namespace)
	data.ServicePolicies = policiesListVal
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *ResourceF5xcTcpLoadbalancerServicePolicyAttachments) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data tcpLoadbalancerServicePolicyAttachmentsModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	namespace := data.Namespace.ValueString()
	if namespace == "" {
		namespace = "default"
	}
	targetName := data.TcpLoadbalancerName.ValueString()

	policyNames, err := r.extractStringList(ctx, data.ServicePolicies)
	if err != nil {
		response.Diagnostics.AddError("Extract Attachment List Error", err.Error())
		return
	}

	unlock := lockAttachmentTarget("tcp_loadbalancer", namespace, targetName)
	defer unlock()

	tflog.Info(ctx, fmt.Sprintf("Updating attachments for tcp_loadbalancer %s in namespace %s: %v", targetName, namespace, policyNames))

	respBytes, targetURL, err := r.fetchTargetResource(ctx, namespace, targetName)
	if err != nil {
		response.Diagnostics.AddError(
			"Get Target Resource Error",
			fmt.Sprintf("Failed to fetch tcp_loadbalancer '%s' in namespace '%s' (URL: %s): %s.", targetName, namespace, targetURL, err.Error()),
		)
		return
	}

	var parentObj map[string]any
	if err := json.Unmarshal(respBytes, &parentObj); err != nil {
		response.Diagnostics.AddError("Unmarshal Target Resource Error", fmt.Sprintf("Failed to parse tcp_loadbalancer JSON: %s", err.Error()))
		return
	}

	spec, _ := parentObj["spec"].(map[string]any)
	if spec == nil {
		spec = make(map[string]any)
		parentObj["spec"] = spec
	}

	if len(policyNames) == 0 {
		delete(spec, "active_service_policies")
		spec["no_service_policies"] = map[string]any{}
	} else {
		var policiesRaw []any
		for _, spName := range policyNames {
			policiesRaw = append(policiesRaw, map[string]any{
				"name":      spName,
				"namespace": namespace,
			})
		}
		spec["active_service_policies"] = map[string]any{
			"policies": policiesRaw,
		}
		delete(spec, "no_service_policies")
	}

	updatedBody, err := json.Marshal(parentObj)
	if err != nil {
		response.Diagnostics.AddError("Marshal Target Resource Error", fmt.Sprintf("Failed to encode updated tcp_loadbalancer JSON: %s", err.Error()))
		return
	}

	_, err = r.client.RestClient.Execute(ctx, http.MethodPut, targetURL, updatedBody)
	if err != nil {
		response.Diagnostics.AddError("Update Target Resource Error", fmt.Sprintf("Failed to update tcp_loadbalancer %s with attachments: %s", targetName, err.Error()))
		return
	}

	data.Namespace = types.StringValue(namespace)
	data.ID = types.StringValue(fmt.Sprintf("%s/%s", namespace, targetName))
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *ResourceF5xcTcpLoadbalancerServicePolicyAttachments) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data tcpLoadbalancerServicePolicyAttachmentsModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	namespace := data.Namespace.ValueString()
	if namespace == "" {
		namespace = "default"
	}
	targetName := data.TcpLoadbalancerName.ValueString()

	unlock := lockAttachmentTarget("tcp_loadbalancer", namespace, targetName)
	defer unlock()

	tflog.Info(ctx, fmt.Sprintf("Clearing attachments for tcp_loadbalancer %s in namespace %s", targetName, namespace))

	respBytes, targetURL, err := r.fetchTargetResource(ctx, namespace, targetName)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Target resource tcp_loadbalancer %s already deleted or not found during policy disassociation: %s", targetName, err.Error()))
		return
	}

	var parentObj map[string]any
	if err := json.Unmarshal(respBytes, &parentObj); err != nil {
		response.Diagnostics.AddError("Unmarshal Target Resource Error", fmt.Sprintf("Failed to parse tcp_loadbalancer JSON: %s", err.Error()))
		return
	}

	spec, _ := parentObj["spec"].(map[string]any)
	if spec == nil {
		return
	}

	delete(spec, "active_service_policies")
	spec["no_service_policies"] = map[string]any{}

	updatedBody, err := json.Marshal(parentObj)
	if err != nil {
		response.Diagnostics.AddError("Marshal Target Resource Error", fmt.Sprintf("Failed to encode updated tcp_loadbalancer JSON: %s", err.Error()))
		return
	}

	_, err = r.client.RestClient.Execute(ctx, http.MethodPut, targetURL, updatedBody)
	if err != nil {
		response.Diagnostics.AddError("Update Target Resource Error", fmt.Sprintf("Failed to clear attachments on tcp_loadbalancer %s: %s", targetName, err.Error()))
		return
	}
}

func (r *ResourceF5xcTcpLoadbalancerServicePolicyAttachments) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	parts := strings.Split(request.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		response.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format 'namespace/target_name', got: %s", request.ID),
		)
		return
	}

	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("id"), request.ID)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("namespace"), parts[0])...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("tcp_loadbalancer_name"), parts[1])...)
}

func (r *ResourceF5xcTcpLoadbalancerServicePolicyAttachments) getTargetURLs(baseURL, namespace, targetName string) []string {
	cleanBase := strings.TrimRight(baseURL, "/")
	return []string{
		fmt.Sprintf("%s/config/namespaces/%s/tcp_loadbalancers/%s", cleanBase, namespace, targetName),
		fmt.Sprintf("%s/public/namespaces/%s/tcp_loadbalancers/%s", cleanBase, namespace, targetName),
	}
}

func (r *ResourceF5xcTcpLoadbalancerServicePolicyAttachments) extractStringList(ctx context.Context, listVal types.List) ([]string, error) {
	if listVal.IsNull() || listVal.IsUnknown() {
		return nil, nil
	}
	var elems []string
	diags := listVal.ElementsAs(ctx, &elems, false)
	if diags.HasError() {
		var errMsgs []string
		for _, d := range diags.Errors() {
			errMsgs = append(errMsgs, d.Detail())
		}
		return nil, fmt.Errorf("failed to extract list elements: %s", strings.Join(errMsgs, "; "))
	}
	return elems, nil
}
