//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package manual

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/client"
	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/utils"
	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/validators/bytesvalidator"
)

// ResourceF5XCApiDefinitionUploader defines the resource implementation.
type ResourceF5XCApiDefinitionUploader struct {
	client *client.APIClient
}

// ResourceF5XCApiDefinitionUploaderMetadataModel describes the metadata section of the model.
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
	Status             types.String                                   `tfsdk:"status"`
	Metadata           ResourceF5XCApiDefinitionUploaderMetadataModel `tfsdk:"metadata"`
	StringValue        types.String                                   `tfsdk:"string_value"`
	ContentHash        types.String                                   `tfsdk:"content_hash"`
	RetainVersions     types.Int64                                    `tfsdk:"retain_versions"`
	PreviousVersionUrl types.String                                   `tfsdk:"previous_version_url"`
	ForceDelete        types.Bool                                     `tfsdk:"force_delete"`
	Timeouts           timeouts.Value                                 `tfsdk:"timeouts"`
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

// apiDefinitionUploaderResponse is the full API response payload.
type apiDefinitionUploaderResponse struct {
	Status      string                                `json:"status"`
	Metadata    apiDefinitionUploaderMetadataResponse `json:"metadata"`
	StringValue string                                `json:"string_value"`
}

// storedObjectVersionItem represents an item in the ListObjects API response.
type storedObjectVersionItem struct {
	Version           string `json:"version"`
	Url               string `json:"url"`
	Description       string `json:"description"`
	CreationTimestamp string `json:"creationTimestamp"`
	LatestVersion     bool   `json:"latestVersion"`
}

// storedObjectListItem represents an entry in the ListObjects response list.
type storedObjectListItem struct {
	Name     string                    `json:"name"`
	Versions []storedObjectVersionItem `json:"versions"`
}

// storedObjectListResponse is the response shape for ListObjects.
type storedObjectListResponse struct {
	Items []storedObjectListItem `json:"items"`
}

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ResourceF5XCApiDefinitionUploader{}
var _ resource.ResourceWithConfigure = &ResourceF5XCApiDefinitionUploader{}
var _ resource.ResourceWithModifyPlan = &ResourceF5XCApiDefinitionUploader{}
var _ resource.ResourceWithValidateConfig = &ResourceF5XCApiDefinitionUploader{}

// computeSHA256 calculates the lowercase hex-encoded SHA-256 digest of an input string.
//
// Why: Used to track content changes and prevent unnecessary version churn when
// uploading OpenAPI specifications to the F5 XC Object Store.
//
// How: Computes the sha256 checksum of the input byte slice and formats it as a hex string.
//
// Parameters:
//   - content: string content to hash.
//
// Returns:
//   - string: lowercase hex-encoded SHA-256 digest.
//
// Example:
//
//	h := computeSHA256("{\"openapi\":\"3.0.0\"}")
func computeSHA256(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

// resolveAbsoluteURL ensures that a URL returned from the F5 XC API has an absolute http/https scheme
// and host, prefixing the client base URL if the returned URL is a relative path.
//
// Why: The F5 XC Object Store API can return either full URLs or relative paths in metadata.url.
// To reliably execute HTTP requests, the URL must be absolute.
//
// How: Checks if targetURL has an http:// or https:// prefix. If not, combines baseURL with targetURL.
//
// Parameters:
//   - baseURL: the configured REST client base URL (e.g. "https://tenant.console.ves.volterra.io/api")
//   - targetURL: the URL to check and possibly prefix
//
// Returns:
//   - string: the resolved absolute URL.
//
// Example:
//
//	abs := resolveAbsoluteURL("https://tenant.console.ves.volterra.io/api", "/object_store/...")
func resolveAbsoluteURL(baseURL, targetURL string) string {
	if strings.HasPrefix(targetURL, "http://") || strings.HasPrefix(targetURL, "https://") {
		return targetURL
	}
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(targetURL, "/")
}

// NewResourceF5XCApiDefinitionUploader creates a new instance of the ResourceF5XCApiDefinitionUploader.
//
// Why: Required by the Terraform Plugin Framework to register the resource with the provider.
//
// How: Allocates and returns a pointer to an unconfigured ResourceF5XCApiDefinitionUploader struct.
//
// Parameters:
//   - None
//
// Returns:
//   - resource.Resource: the newly instantiated resource implementation.
//
// Example:
//
//	res := NewResourceF5XCApiDefinitionUploader()
func NewResourceF5XCApiDefinitionUploader() resource.Resource {
	return &ResourceF5XCApiDefinitionUploader{}
}

// Metadata specifies the resource type name.
//
// Why: Informs Terraform what resource type this struct implements (e.g., "f5xc_api_definition_uploader").
//
// How: Combines the provider's type name prefix with "_api_definition_uploader".
//
// Parameters:
//   - ctx: execution context.
//   - request: metadata request provided by the framework.
//   - response: metadata response where TypeName is assigned.
//
// Returns:
//   - None
//
// Example:
//
//	r.Metadata(ctx, req, resp)
func (r *ResourceF5XCApiDefinitionUploader) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_api_definition_uploader"
}

// Schema defines the schema for the f5xc_api_definition_uploader resource.
//
// Why: Defines all attributes, types, defaults, plan modifiers, and validators for Terraform.
//
// How: Constructs a framework schema.Schema mapping out metadata, string_value, content_hash,
// retain_versions, previous_version_url, force_delete, and timeout settings.
//
// Parameters:
//   - ctx: execution context.
//   - request: schema request from the framework.
//   - response: schema response where the Schema definition is assigned.
//
// Returns:
//   - None
//
// Example:
//
//	r.Schema(ctx, req, resp)
func (r *ResourceF5XCApiDefinitionUploader) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: `# F5 XC API Definition Uploader

The f5xc_api_definition_uploader resource uploads raw OpenAPI or Swagger specifications to the F5 Distributed Cloud (F5 XC) Object Store. It exports the backend-generated versioned URL in metadata.url, which can be passed to f5xc_api_definition for API Discovery, API Inventory, and API Protection policies.

## Declarative Upload

Takes the raw specification through the string_value attribute. The value can come from a local file, a rendered template, decoded Base64 input, a remote HTTP response, cloud storage, or jsonencode(...).

## Object Store Integration

Uploads the specification with a PUT request to the F5 XC Object Store REST API:

	PUT /object_store/namespaces/{namespace}/stored_objects/swagger/{name}

## Lifecycle Management

The resource manages the object lifecycle:

- Create uploads the specification.
- Read retrieves the stored object through metadata.url.
- Update replaces the stored specification.
- Delete removes the stored object.

## Computed URL Export

Captures the backend-generated versioned URL in metadata.url for use by dependent f5xc_api_definition resources.`,
		Description: `f5xc_api_definition_uploader uploads raw OpenAPI or Swagger specifications to the F5 Distributed Cloud Object Store. It accepts the specification through string_value, manages create, read, update, and delete operations, and exports the backend-generated versioned URL in metadata.url for use by f5xc_api_definition.

Supported input sources include local files, rendered templates, decoded Base64 values, remote HTTP responses, cloud storage objects, and jsonencode(...).`,

		Attributes: map[string]schema.Attribute{
			"metadata": schema.SingleNestedAttribute{
				Description: "metadata Set schema update strategy for the API Inventory schemas management.",
				Required:    true,
				Optional:    false,
				Computed:    false,
				Attributes: map[string]schema.Attribute{
					"url": schema.StringAttribute{
						Description: "url The versioned URL of the stored OpenAPI specification in F5 XC Object Store.",
						Required:    false,
						Optional:    false,
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "name This is the name of configuration object. It has to be unique within the namespace.",
						Required:    true,
						Optional:    false,
						Computed:    false,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"version": schema.StringAttribute{
						Description: "version The version string assigned by F5 XC (e.g. v1-26-09-24).",
						Required:    false,
						Optional:    false,
						Computed:    true,
					},
					"namespace": schema.StringAttribute{
						Description: "namespace This defines the namespace within which the object is created.",
						Required:    true,
						Optional:    false,
						Computed:    false,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"description": schema.StringAttribute{
						Description: "description Human readable description for the object",
						Required:    false,
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString(""),
						Validators: []validator.String{
							bytesvalidator.BytesAtMost(512),
						},
					},
					"creation_timestamp": schema.StringAttribute{
						Description: "creation_timestamp Timestamp when the object version was created.",
						Required:    false,
						Optional:    false,
						Computed:    true,
					},
				},
			},
			"status": schema.StringAttribute{
				Description: "status Response status from the F5 XC object store create/update operation.",
				Required:    false,
				Optional:    false,
				Computed:    true,
			},
			"string_value": schema.StringAttribute{
				Description: "string_value Raw OpenAPI specification content (JSON or YAML).",
				Required:    true,
				Optional:    false,
				Computed:    false,
			},
			"content_hash": schema.StringAttribute{
				Description: "content_hash SHA-256 hash of string_value used to detect content modifications and prevent version churn.",
				Required:    false,
				Optional:    false,
				Computed:    true,
			},
			"retain_versions": schema.Int64Attribute{
				Description: "retain_versions Maximum number of historical versions to retain in the F5 XC object store. Defaults to 2 (current + previous version). Set to 0 to disable automated pruning.",
				Required:    false,
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2),
			},
			"previous_version_url": schema.StringAttribute{
				Description: "previous_version_url The URL of the immediately preceding version, available for rollback references.",
				Required:    false,
				Optional:    false,
				Computed:    true,
			},
			"force_delete": schema.BoolAttribute{
				Description: "force_delete When true, destroys all historical versions of the stored object upon resource deletion.",
				Required:    false,
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true, // activate timeouts.create
				Update: true, // activate timeouts.update
				Delete: true, // activate timeouts.delete
				Read:   true, // activate timeouts.read
			}),
		},
	}
}

// Configure supplies the API client from the provider configuration to the resource.
//
// Why: The resource needs access to client.APIClient to execute HTTP calls against F5 XC.
//
// How: Type-asserts ProviderData to *client.APIClient and stores it in the struct.
//
// Parameters:
//   - ctx: execution context.
//   - request: configure request containing ProviderData.
//   - response: configure response for reporting errors.
//
// Returns:
//   - None
//
// Example:
//
//	r.Configure(ctx, req, resp)
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

// ValidateConfig implements resource.ResourceWithValidateConfig.
//
// Why: Validates the configuration model before planning.
//
// How: Reads the config model and ensures no structural errors exist.
//
// Parameters:
//   - ctx: execution context.
//   - req: validate config request.
//   - resp: validate config response.
//
// Returns:
//   - None
//
// Example:
//
//	r.ValidateConfig(ctx, req, resp)
func (r *ResourceF5XCApiDefinitionUploader) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ResourceF5XCApiDefinitionUploaderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
//
// Why: Dynamically marks versioned attributes as unknown during an update plan when string_value changes,
// ensuring downstream resources (like f5xc_api_definition) accurately plan their updates.
//
// How: Inspects plan vs state; if string_value changes, sets metadata.url, metadata.version,
// metadata.creation_timestamp, content_hash, and status to unknown.
//
// Parameters:
//   - ctx: execution context.
//   - req: modify plan request containing prior state and proposed plan.
//   - resp: modify plan response where adjusted plan values are written.
//
// Returns:
//   - None
//
// Example:
//
//	r.ModifyPlan(ctx, req, resp)
func (r *ResourceF5XCApiDefinitionUploader) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		resp.Diagnostics.AddWarning(
			"Resource Destruction",
			"This will permanently delete the API Definition stored object from F5 XC.",
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
		return
	}

	var plan ResourceF5XCApiDefinitionUploaderModel
	var state ResourceF5XCApiDefinitionUploaderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If string_value is changing, mark versioned attributes as unknown so dependents plan for update
	if !plan.StringValue.IsUnknown() && !state.StringValue.IsNull() && plan.StringValue.ValueString() != state.StringValue.ValueString() {
		plan.Metadata.Url = types.StringUnknown()
		plan.Metadata.Version = types.StringUnknown()
		plan.Metadata.CreationTimestamp = types.StringUnknown()
		plan.ContentHash = types.StringUnknown()
		plan.Status = types.StringUnknown()
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

// Create uploads the OpenAPI spec to the F5 XC Object Store and saves the immutable version URL in state.
//
// Why: Initial creation of the stored object version in F5 XC.
//
// How: Computes SHA-256 of string_value, builds the JSON payload, executes HTTP PUT,
// decodes the response, and sets state attributes including url, version, and content_hash.
//
// Parameters:
//   - ctx: execution context.
//   - request: create request with plan data.
//   - response: create response where state is set or diagnostics reported.
//
// Returns:
//   - None
//
// Example:
//
//	r.Create(ctx, req, resp)
func (r *ResourceF5XCApiDefinitionUploader) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data ResourceF5XCApiDefinitionUploaderModel

	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	duration, err := time.ParseDuration("10m0s")
	if err != nil {
		response.Diagnostics.AddError(
			"Invalid Duration Format",
			fmt.Sprintf("Could not parse 'timeout' value %q: %s.", "10m0s", err.Error()),
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

	hash := computeSHA256(data.StringValue.ValueString())
	desc := data.Metadata.Description.ValueString()

	createReqMetadata := client.ResourceF5XCApiDefinitionUploaderMetadataApiResponse{
		Name:        data.Metadata.Name.ValueString(),
		Namespace:   data.Metadata.Namespace.ValueString(),
		Description: desc,
	}

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

	putURL := fmt.Sprintf("%s/object_store/namespaces/%s/stored_objects/swagger/%s",
		r.client.RestClient.URL,
		data.Metadata.Namespace.ValueString(),
		data.Metadata.Name.ValueString())

	tflog.Debug(ctx, fmt.Sprintf("PUT URL: %s", putURL))

	httpResp, err := r.client.RestClient.Execute(ctx, http.MethodPut, putURL, httpReqBody)
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
			"An unexpected error occurred while parsing the resource creation response: "+err.Error(),
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
	data.ContentHash = types.StringValue(hash)
	data.PreviousVersionUrl = types.StringNull()

	tflog.Info(ctx, fmt.Sprintf("Created Api definition uploader resource: %s version: %s", data.Metadata.Name.ValueString(), data.Metadata.Version.ValueString()))

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

// Read refreshes the resource state by querying the F5 XC Object Store.
//
// Why: Detects configuration drift and external deletion.
//
// How: Executes HTTP GET against metadata.url. If HTTP 404 is encountered, cleanly removes
// the resource from state. Otherwise, refreshes metadata, string_value, and content_hash.
//
// Parameters:
//   - ctx: execution context.
//   - request: read request containing prior state.
//   - response: read response where updated state is stored.
//
// Returns:
//   - None
//
// Example:
//
//	r.Read(ctx, req, resp)
func (r *ResourceF5XCApiDefinitionUploader) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data ResourceF5XCApiDefinitionUploaderModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	duration, err := time.ParseDuration("10m0s")
	if err != nil {
		response.Diagnostics.AddError(
			"Invalid Duration Format",
			fmt.Sprintf("Could not parse 'timeout' value %q: %s.", "10m0s", err.Error()),
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

	readURL := resolveAbsoluteURL(r.client.RestClient.URL, data.Metadata.Url.ValueString())
	httpResp, err := r.client.RestClient.Execute(ctx, http.MethodGet, readURL, nil)
	if err != nil {
		if utils.IsNotFoundError(err) ||
			strings.Contains(strings.ToLower(err.Error()), "404") ||
			strings.Contains(strings.ToLower(err.Error()), "not found") {
			tflog.Warn(ctx, fmt.Sprintf("Stored object version %s not found on F5 XC; removing from state", data.Metadata.Url.ValueString()))
			response.State.RemoveResource(ctx)
			return
		}

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
			"An unexpected error occurred while parsing the read response: "+err.Error(),
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
	data.ContentHash = types.StringValue(computeSHA256(readResp.StringValue))

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

// Update uploads a new version of the OpenAPI spec when content changes, and triggers retention pruning.
//
// Why: Enables automated rollout of new API definitions while avoiding version churn on unchanged specs
// and pruning stale historical versions.
//
// How: Compares SHA-256 hashes. If unchanged, skips upload. If changed, PUTs new version,
// captures new URL and version, sets previous_version_url, and calls pruneHistoricalVersions.
//
// Parameters:
//   - ctx: execution context.
//   - request: update request containing plan and prior state.
//   - response: update response where state is persisted.
//
// Returns:
//   - None
//
// Example:
//
//	r.Update(ctx, req, resp)
func (r *ResourceF5XCApiDefinitionUploader) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan ResourceF5XCApiDefinitionUploaderModel
	var state ResourceF5XCApiDefinitionUploaderModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	duration, err := time.ParseDuration("10m0s")
	if err != nil {
		response.Diagnostics.AddError(
			"Invalid Duration Format",
			fmt.Sprintf("Could not parse 'timeout' value %q: %s.", "10m0s", err.Error()),
		)
		return
	}

	updateTimeout, diags := plan.Timeouts.Update(ctx, duration)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	newHash := computeSHA256(plan.StringValue.ValueString())
	contentChanged := state.ContentHash.IsNull() ||
		state.ContentHash.ValueString() != newHash ||
		state.StringValue.ValueString() != plan.StringValue.ValueString()

	descChanged := plan.Metadata.Description.ValueString() != state.Metadata.Description.ValueString()

	// If neither content nor description changed, preserve current backend version and avoid version churn.
	// However, update configuration attributes (ForceDelete, RetainVersions, Timeouts) in state from plan.
	if !contentChanged && !descChanged {
		tflog.Debug(ctx, "Api definition uploader content unchanged; skipping upload to prevent version churn")
		state.ForceDelete = plan.ForceDelete
		state.RetainVersions = plan.RetainVersions
		state.Timeouts = plan.Timeouts

		if !plan.RetainVersions.IsNull() && !plan.RetainVersions.IsUnknown() && plan.RetainVersions.ValueInt64() > 0 {
			r.pruneHistoricalVersions(ctx, state.Metadata.Namespace.ValueString(), state.Metadata.Name.ValueString(), plan.RetainVersions.ValueInt64(), state.Metadata.Version.ValueString())
		}

		response.Diagnostics.Append(response.State.Set(ctx, &state)...)
		return
	}

	desc := plan.Metadata.Description.ValueString()

	updateReqMetadata := client.ResourceF5XCApiDefinitionUploaderMetadataApiResponse{
		Name:        plan.Metadata.Name.ValueString(),
		Namespace:   plan.Metadata.Namespace.ValueString(),
		Description: desc,
	}

	updateReq := client.ResourceF5XCApiDefinitionUploaderApiModel{
		Metadata:    updateReqMetadata,
		StringValue: plan.StringValue.ValueString(),
	}

	httpReqBody, err := json.Marshal(updateReq)
	if err != nil {
		response.Diagnostics.AddError(
			"Serialization Error",
			fmt.Sprintf("Unable to serialize request body: %s", err.Error()),
		)
		return
	}

	putURL := fmt.Sprintf("%s/object_store/namespaces/%s/stored_objects/swagger/%s",
		r.client.RestClient.URL,
		plan.Metadata.Namespace.ValueString(),
		plan.Metadata.Name.ValueString())

	tflog.Info(ctx, fmt.Sprintf("Uploading new version for Api definition uploader %s (%s)", plan.Metadata.Name.ValueString(), putURL))

	httpResp, err := r.client.RestClient.Execute(ctx, http.MethodPut, putURL, httpReqBody)
	if err != nil {
		response.Diagnostics.AddError(
			"Client Error", fmt.Sprintf("Unable to update %s, got error: %s", "Api definition uploader", err),
		)
		return
	}

	var updateResp apiDefinitionUploaderResponse
	err = json.NewDecoder(bytes.NewBuffer(httpResp)).Decode(&updateResp)
	if err != nil {
		response.Diagnostics.AddError(
			"Unable to Update Resource",
			"An unexpected error occurred while parsing the resource update response: "+err.Error(),
		)
		return
	}

	// Capture previous version URL for rollback visibility
	plan.PreviousVersionUrl = state.Metadata.Url
	plan.Status = types.StringValue(updateResp.Status)
	plan.Metadata.Url = types.StringValue(updateResp.Metadata.Url)
	plan.Metadata.Name = types.StringValue(updateResp.Metadata.Name)
	plan.Metadata.Version = types.StringValue(updateResp.Metadata.Version)
	plan.Metadata.Namespace = types.StringValue(updateResp.Metadata.Namespace)
	plan.Metadata.Description = types.StringValue(updateResp.Metadata.Description)
	plan.Metadata.CreationTimestamp = types.StringValue(updateResp.Metadata.CreationTimestamp)
	plan.ContentHash = types.StringValue(newHash)

	tflog.Info(ctx, fmt.Sprintf("Updated Api definition uploader resource: %s new version: %s (previous: %s)",
		plan.Metadata.Name.ValueString(), plan.Metadata.Version.ValueString(), state.Metadata.Version.ValueString()))

	// Perform retention pruning if configured
	retainCount := int64(2)
	if !plan.RetainVersions.IsNull() && !plan.RetainVersions.IsUnknown() {
		retainCount = plan.RetainVersions.ValueInt64()
	}
	r.pruneHistoricalVersions(ctx, plan.Metadata.Namespace.ValueString(), plan.Metadata.Name.ValueString(), retainCount, plan.Metadata.Version.ValueString())

	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
}

// Delete removes the stored object version (or all versions if force_delete is enabled).
//
// Why: Destroys the managed stored object upon terraform destroy.
//
// How: Invokes HTTP DELETE against metadata.url (or root object URL if force_delete = true).
// Ignores 404 Not Found errors during destroy.
//
// Parameters:
//   - ctx: execution context.
//   - request: delete request containing prior state.
//   - response: delete response for reporting diagnostics.
//
// Returns:
//   - None
//
// Example:
//
//	r.Delete(ctx, req, resp)
func (r *ResourceF5XCApiDefinitionUploader) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data ResourceF5XCApiDefinitionUploaderModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	duration, err := time.ParseDuration("10m0s")
	if err != nil {
		response.Diagnostics.AddError(
			"Invalid Duration Format",
			fmt.Sprintf("Could not parse 'timeout' value %q: %s.", "10m0s", err.Error()),
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

	var deleteURL string
	if !data.ForceDelete.IsNull() && data.ForceDelete.ValueBool() {
		// Delete the whole object including all historical versions
		deleteURL = fmt.Sprintf("%s/object_store/namespaces/%s/stored_objects/swagger/%s",
			r.client.RestClient.URL,
			data.Metadata.Namespace.ValueString(),
			data.Metadata.Name.ValueString())
	} else {
		// Delete only the managed version
		deleteURL = data.Metadata.Url.ValueString()
	}

	deleteURL = resolveAbsoluteURL(r.client.RestClient.URL, deleteURL)
	tflog.Info(ctx, fmt.Sprintf("Deleting Api definition uploader at %s", deleteURL))

	_, err = r.client.RestClient.Execute(ctx, http.MethodDelete, deleteURL, nil)
	if err != nil {
		if utils.IsNotFoundError(err) ||
			strings.Contains(strings.ToLower(err.Error()), "404") ||
			strings.Contains(strings.ToLower(err.Error()), "not found") {
			tflog.Debug(ctx, "Object already deleted in F5 XC; ignoring 404 on destroy")
			return
		}

		response.Diagnostics.AddError(
			"Client Error", fmt.Sprintf("Unable to delete %s, got error: %s", "Api definition uploader", err),
		)
		return
	}
}

// pruneHistoricalVersions prunes older versions of the stored object so that at most retainCount
// versions are preserved in the F5 XC Object Store.
//
// Why: Prevents unbounded accumulation of historical OpenAPI spec versions (garbage collection)
// while retaining recent versions (default 2) for rollback safety.
//
// How: Fetches the list of all versions via the ListObjects API, sorts them from newest to oldest
// based on creation timestamp and version sequence, and executes HTTP DELETE calls for any
// version beyond the retainCount threshold. Deletion failures are logged as warnings and do
// not fail the main operation.
//
// Parameters:
//   - ctx: context for HTTP execution and logging.
//   - namespace: namespace containing the stored object.
//   - name: name of the stored object.
//   - retainCount: maximum number of versions to keep (0 disables pruning).
//   - currentVersion: the active version just created, which must never be deleted.
//
// Returns:
//   - None
//
// Example:
//
//	r.pruneHistoricalVersions(ctx, "shared", "my-spec", 2, "v2-26-09-24")
func (r *ResourceF5XCApiDefinitionUploader) pruneHistoricalVersions(
	ctx context.Context,
	namespace, name string,
	retainCount int64,
	currentVersion string,
) {
	if retainCount <= 0 {
		return
	}

	listURL := fmt.Sprintf("%s/object_store/namespaces/%s/stored_objects/swagger?name=%s",
		r.client.RestClient.URL, namespace, name)

	respBytes, err := r.client.RestClient.Execute(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Failed to list stored object versions for pruning: %s", err))
		return
	}

	var listResp storedObjectListResponse
	if err := json.Unmarshal(respBytes, &listResp); err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Failed to decode stored object versions for pruning: %s", err))
		return
	}

	for _, item := range listResp.Items {
		if item.Name != name {
			continue
		}

		if int64(len(item.Versions)) <= retainCount {
			return
		}

		// Sort newest first
		versions := make([]storedObjectVersionItem, len(item.Versions))
		copy(versions, item.Versions)

		sort.Slice(versions, func(i, j int) bool {
			t1, err1 := time.Parse(time.RFC3339Nano, versions[i].CreationTimestamp)
			if err1 != nil {
				t1, err1 = time.Parse(time.RFC3339, versions[i].CreationTimestamp)
			}
			t2, err2 := time.Parse(time.RFC3339Nano, versions[j].CreationTimestamp)
			if err2 != nil {
				t2, err2 = time.Parse(time.RFC3339, versions[j].CreationTimestamp)
			}
			if err1 == nil && err2 == nil && !t1.Equal(t2) {
				return t1.After(t2)
			}
			return versions[i].Version > versions[j].Version
		})

		// Keep the first retainCount versions; delete the rest
		for i := retainCount; i < int64(len(versions)); i++ {
			v := versions[i]
			if v.Version == currentVersion {
				continue
			}

			delURL := v.Url
			if delURL == "" {
				delURL = fmt.Sprintf("%s/object_store/namespaces/%s/stored_objects/swagger/%s/%s",
					r.client.RestClient.URL, namespace, name, v.Version)
			} else {
				delURL = resolveAbsoluteURL(r.client.RestClient.URL, delURL)
			}

			tflog.Info(ctx, fmt.Sprintf("Pruning historical version %s (%s) of stored object %s",
				v.Version, delURL, name))

			if _, delErr := r.client.RestClient.Execute(ctx, http.MethodDelete, delURL, nil); delErr != nil {
				tflog.Warn(ctx, fmt.Sprintf("Failed to prune old version %s: %s", v.Version, delErr))
			}
		}
	}
}
