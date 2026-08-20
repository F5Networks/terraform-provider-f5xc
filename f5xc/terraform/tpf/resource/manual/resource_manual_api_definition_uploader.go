//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package manual

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/client"
	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/validators/bytesvalidator"
)

// ResourceF5XCApiDefinitionUploader defines the resource implementation.
type ResourceF5XCApiDefinitionUploader struct {
	client *client.APIClient
}

type ResourceF5XCApiDefinitionUploaderMetadataModel struct {
	Url               types.String `tfsdk:"url"`
	Name              types.String `tfsdk:"name"`
	Version           types.String `tfsdk:"version"`
	Namespace         types.String `tfsdk:"namespace"`
	Description       types.String `tfsdk:"description"`
	CreationTimestamp types.String `tfsdk:"creation_timestamp"`
}

// ResourceF5XCApiDefinitionUploaderModel describes the resource data model.
type ResourceF5XCApiDefinitionUploaderModel struct {
	Status      types.String                                   `tfsdk:"status"`
	Metadata    ResourceF5XCApiDefinitionUploaderMetadataModel `tfsdk:"metadata"`
	StringValue types.String                                   `tfsdk:"string_value"`
	Timeouts    timeouts.Value                                 `tfsdk:"timeouts"`
}

// apiDefinitionUploaderMetadataResponse is the metadata section of the API response payload.
type apiDefinitionUploaderMetadataResponse struct {
	Url               string `json:"url"`
	Name              string `json:"name"`
	Version           string `json:"version"`
	Namespace         string `json:"namespace"`
	Description       string `json:"description"`
	CreationTimestamp string `json:"creationTimestamp"`
}

// apiDefinitionUploaderResponse is the full API response payload (includes Status, unlike the request model in client.go).
type apiDefinitionUploaderResponse struct {
	Status      string                                `json:"status"`
	Metadata    apiDefinitionUploaderMetadataResponse `json:"metadata"`
	StringValue string                                `json:"string_value"`
}

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ResourceF5XCApiDefinitionUploader{}
var _ resource.ResourceWithConfigure = &ResourceF5XCApiDefinitionUploader{}
var _ resource.ResourceWithModifyPlan = &ResourceF5XCApiDefinitionUploader{}
var _ resource.ResourceWithValidateConfig = &ResourceF5XCApiDefinitionUploader{}

func NewResourceF5XCApiDefinitionUploader() resource.Resource {
	return &ResourceF5XCApiDefinitionUploader{}
}

func (r *ResourceF5XCApiDefinitionUploader) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_api_definition_uploader"
}

func (r *ResourceF5XCApiDefinitionUploader) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Api definition uploader resource",
		Description:         "Api definition uploader resource",

		Attributes: map[string]schema.Attribute{
			"metadata": schema.SingleNestedAttribute{
				Description: "metadata Set schema update strategy for the API Inventory schemas management.",
				Required:    true,
				Optional:    false,
				Computed:    false,
				Attributes: map[string]schema.Attribute{
					"url": schema.StringAttribute{
						Description: "url This defines the workspace within which each the configuration object is to be created.",
						Required:    false,
						Optional:    false,
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "name This is the name of configuration object. It has to be unique within the namespace.",
						Required:    true,
						Optional:    false,
						Computed:    false,
					},
					"version": schema.StringAttribute{
						Description: "version This defines the workspace within which each the configuration object is to be created.",
						Required:    false,
						Optional:    false,
						Computed:    true,
					},
					"namespace": schema.StringAttribute{
						Description: "namespace This defines the workspace within which each the configuration object is to be created.",
						Required:    true,
						Optional:    false,
						Computed:    false,
					},
					"description": schema.StringAttribute{
						Description: "description Human readable description for the object",
						Required:    false,
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString(""),
						Validators: []validator.String{
							bytesvalidator.BytesAtMost(1200),
						},
					},
					"creation_timestamp": schema.StringAttribute{
						Description: "creation_timestamp This defines the workspace within which each the configuration object is to be created.",
						Required:    false,
						Optional:    false,
						Computed:    true,
					},
				},
			},
			"status": schema.StringAttribute{
				Description: "status This defines the workspace within which each the configuration object is to be created.",
				Required:    false,
				Optional:    false,
				Computed:    true,
			},
			"string_value": schema.StringAttribute{
				Description: "string_value This defines the workspace within which each the configuration object is to be created.",
				Required:    true,
				Optional:    false,
				Computed:    false,
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true, // activate timeouts.create
				Update: true, // activate timeouts.update (false for CRLD resources)
				Delete: true, // activate timeouts.delete
				Read:   true, // activate timeouts.read
			}),
		},
	}
}

func (r *ResourceF5XCApiDefinitionUploader) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

// ValidateConfig implements resource.ResourceWithValidateConfig
func (r *ResourceF5XCApiDefinitionUploader) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ResourceF5XCApiDefinitionUploaderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// ModifyPlan implements resource.ResourceWithModifyPlan
func (r *ResourceF5XCApiDefinitionUploader) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		resp.Diagnostics.AddWarning(
			"Resource Destruction",
			"This will permanently delete the API Definition from F5 XC.",
		)
		return
	}

	if req.State.Raw.IsNull() {
		var plan ResourceF5XCApiDefinitionUploaderModel
		resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}

		if plan.Metadata.Name.IsUnknown() {
			resp.Diagnostics.AddWarning(
				"Unknown Resource Name",
				"The resource name is not yet known. This may affect planning for dependent resources.",
			)
		}
	}
}

func (r *ResourceF5XCApiDefinitionUploader) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data ResourceF5XCApiDefinitionUploaderModel

	// api/object_store/namespaces/<namespace_name>/stored_objects/swagger/<name>
	// Read Terraform plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	duration, err := time.ParseDuration("10m0s")
	if err != nil {
		response.Diagnostics.AddError(
			"Invalid Duration Format",
			fmt.Sprintf("Could not parse 'timeout' value %q: %s. "+
				"Please use a valid Go duration string like '10m', '1h30m' or '30s'.",
				"10m0s", err.Error()),
		)
		return
	}

	createTimeout, diags := data.Timeouts.Create(ctx, duration)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	// Convert from Terraform data model into API request model (client.go types, no Status field)
	createReqMetadata := client.ResourceF5XCApiDefinitionUploaderMetadataApiResponse{
		Name:        data.Metadata.Name.ValueString(),
		Namespace:   data.Metadata.Namespace.ValueString(),
		Description: data.Metadata.Description.ValueString(),
	}

	// Convert from Terraform data model into API request model
	createReq := client.ResourceF5XCApiDefinitionUploaderApiModel{
		Metadata:    createReqMetadata,
		StringValue: data.StringValue.ValueString(),
	}

	httpReqBody, err := json.Marshal(createReq)
	if err != nil {
		response.Diagnostics.AddError(
			"Serialization Error",
			fmt.Sprintf("Unable to serialize request body: %s", err.Error()),
		)
		return
	}

	tflog.Info(ctx, fmt.Sprintf("GetClientData: %s", r.client.RestClient.Authenticator.GetClientData()))
	tflog.Info(ctx, fmt.Sprintf("API token URL: %s/object_store/namespaces/%s/stored_objects/swagger/%s", r.client.RestClient.URL, data.Metadata.Namespace.ValueString(), data.Metadata.Name.ValueString()))

	httpResp, err := r.client.RestClient.Execute(ctx, http.MethodPut,
		fmt.Sprintf("%s/object_store/namespaces/%s/stored_objects/swagger/%s",
			r.client.RestClient.URL,
			data.Metadata.Namespace.ValueString(),
			data.Metadata.Name.ValueString()), httpReqBody)
	if err != nil {
		response.Diagnostics.AddError(
			"Client Error", fmt.Sprintf("Unable to create %s, got error: %s", "Api definition uploader", err),
		)
		return
	}

	var createResp apiDefinitionUploaderResponse
	err = json.NewDecoder(bytes.NewBuffer(httpResp)).Decode(&createResp)

	if err != nil {
		response.Diagnostics.AddError(
			"Unable to Create Resource",
			"An unexpected error occurred while parsing the resource creation response. "+
				"Please report this issue to the provider developers.\n\n"+
				"JSON Error: "+err.Error(),
		)

		return
	}

	data.Status = types.StringValue(createResp.Status)
	data.Metadata.Url = types.StringValue(createResp.Metadata.Url)
	data.Metadata.Name = types.StringValue(createResp.Metadata.Name)
	data.Metadata.Version = types.StringValue(createResp.Metadata.Version)
	data.Metadata.Namespace = types.StringValue(createResp.Metadata.Namespace)
	data.Metadata.Description = types.StringValue(createResp.Metadata.Description)
	data.Metadata.CreationTimestamp = types.StringValue(createResp.Metadata.CreationTimestamp)

	tflog.Info(ctx, fmt.Sprintf("created a %s resource", "Api definition uploader"))

	// Set state to fully populated data
	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *ResourceF5XCApiDefinitionUploader) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data ResourceF5XCApiDefinitionUploaderModel

	// Read Terraform prior state data into the model
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	duration, err := time.ParseDuration("10m0s")
	if err != nil {
		response.Diagnostics.AddError(
			"Invalid Duration Format",
			fmt.Sprintf("Could not parse 'timeout' value %q: %s. "+
				"Please use a valid Go duration string like '10m', '1h30m' or '30s'.",
				"10m0s", err.Error()),
		)
		return
	}

	readTimeout, diags := data.Timeouts.Read(ctx, duration)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	httpResp, err := r.client.RestClient.Execute(ctx, http.MethodGet, data.Metadata.Url.ValueString(), nil)
	if err != nil {
		response.Diagnostics.AddError(
			"Client Error", fmt.Sprintf("Unable to read %s, got error: %s", "Api definition uploader", err),
		)
		return
	}

	var readResp apiDefinitionUploaderResponse
	err = json.NewDecoder(bytes.NewBuffer(httpResp)).Decode(&readResp)

	if err != nil {
		response.Diagnostics.AddError(
			"Unable to Read Resource",
			"An unexpected error occurred while parsing the resource creation response. "+
				"Please report this issue to the provider developers.\n\n"+
				"JSON Error: "+err.Error(),
		)

		return
	}

	data.Metadata.Url = types.StringValue(readResp.Metadata.Url)
	data.Metadata.Name = types.StringValue(readResp.Metadata.Name)
	data.Metadata.Version = types.StringValue(readResp.Metadata.Version)
	data.Metadata.Namespace = types.StringValue(readResp.Metadata.Namespace)
	data.Metadata.Description = types.StringValue(readResp.Metadata.Description)
	data.Metadata.CreationTimestamp = types.StringValue(readResp.Metadata.CreationTimestamp)
	data.StringValue = types.StringValue(readResp.StringValue)

	// Save updated data into Terraform state
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *ResourceF5XCApiDefinitionUploader) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	// Update is a replace and read operation
	var data ResourceF5XCApiDefinitionUploaderModel

	// Read Terraform plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	duration, err := time.ParseDuration("10m0s")
	if err != nil {
		response.Diagnostics.AddError(
			"Invalid Duration Format",
			fmt.Sprintf("Could not parse 'timeout' value %q: %s. "+
				"Please use a valid Go duration string like '10m', '1h30m' or '30s'.",
				"10m0s", err.Error()),
		)
		return
	}

	updateTimeout, diags := data.Timeouts.Update(ctx, duration)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	// Save updated data into Terraform state
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *ResourceF5XCApiDefinitionUploader) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data ResourceF5XCApiDefinitionUploaderModel

	// Read Terraform prior state data into the model
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	duration, err := time.ParseDuration("10m0s")
	if err != nil {
		response.Diagnostics.AddError(
			"Invalid Duration Format",
			fmt.Sprintf("Could not parse 'timeout' value %q: %s. "+
				"Please use a valid Go duration string like '10m', '1h30m' or '30s'.",
				"10m0s", err.Error()),
		)
		return
	}

	deleteTimeout, diags := data.Timeouts.Delete(ctx, duration)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	_, err = r.client.RestClient.Execute(ctx, http.MethodDelete, data.Metadata.Url.ValueString(), nil)
	if err != nil {
		response.Diagnostics.AddError(
			"Client Error", fmt.Sprintf("Unable to delete %s, got error: %s", "Api definition uploader", err),
		)
		return
	}
}
