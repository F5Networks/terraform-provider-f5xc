//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package manual

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
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
	BytesValue         types.String                                   `tfsdk:"bytes_value"`
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
	BytesValue  string                                `json:"bytes_value"`
}

// storedObjectVersionItem represents an item in the ListObjects API response.
type storedObjectVersionItem struct {
	Version                string `json:"version"`
	Url                    string `json:"url"`
	Description            string `json:"description"`
	CreationTimestamp      string `json:"creationTimestamp"`
	CreationTimestampSnake string `json:"creation_timestamp"`
	LatestVersion          bool   `json:"latestVersion"`
	LatestVersionSnake     bool   `json:"latest_version"`
}

// effectiveCreationTimestamp returns the non-empty creation timestamp across camelCase and snake_case JSON fields.
//
// Why: F5 XC Object Store endpoints serialize creation timestamps as "creation_timestamp" (snake_case)
// on list responses, whereas other responses or mocks may use "creationTimestamp" (camelCase).
// Supporting both ensures reliable chronological sorting during version retention pruning.
//
// How: Checks CreationTimestamp first and falls back to CreationTimestampSnake when empty.
//
// Parameters:
//   - None (receiver method on storedObjectVersionItem)
//
// Returns:
//   - string: the timestamp string if present in either field, otherwise empty string
//
// Example:
//
//	ts := item.effectiveCreationTimestamp()
func (v storedObjectVersionItem) effectiveCreationTimestamp() string {
	if v.CreationTimestamp != "" {
		return v.CreationTimestamp
	}
	return v.CreationTimestampSnake
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

// maxStoredObjectSpecBytes defines the maximum OpenAPI specification payload size (5 MiB)
// enforced by F5 XC stored_object schema constraints.
const maxStoredObjectSpecBytes = 5242880

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ResourceF5XCApiDefinitionUploader{}
var _ resource.ResourceWithConfigure = &ResourceF5XCApiDefinitionUploader{}
var _ resource.ResourceWithModifyPlan = &ResourceF5XCApiDefinitionUploader{}
var _ resource.ResourceWithValidateConfig = &ResourceF5XCApiDefinitionUploader{}

// resolvedSpecContent holds the normalized API specification payload and derived
// metadata resolved from one of the write-only content attributes.
type resolvedSpecContent struct {
	attrName     string
	isBytes      bool
	stringValue  string
	bytesValue   string
	computedHash string
	present      bool
}

// computeSHA256Bytes calculates the lowercase hex-encoded SHA-256 digest of a byte slice.
//
// Why: Provides a uniform digest calculation for both plain-text string_value specs and
// Base64-decoded bytes_value specs so identical specifications yield identical digests.
//
// How: Computes the sha256 checksum of content and formats it as a lowercase hex string.
//
// Parameters:
//   - content: raw byte slice to hash.
//
// Returns:
//   - string: lowercase hex-encoded SHA-256 digest.
//
// Example:
//
//	h := computeSHA256Bytes([]byte(`{"openapi":"3.0.0"}`))
func computeSHA256Bytes(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// computeSHA256 calculates the lowercase hex-encoded SHA-256 digest of an input string.
//
// Why: Used to track content changes and prevent unnecessary version churn when
// uploading OpenAPI specifications to the F5 XC Object Store.
//
// How: Delegates to computeSHA256Bytes on the UTF-8 byte representation of content.
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
	return computeSHA256Bytes([]byte(content))
}

// decodeBase64Spec decodes a standard Base64-encoded OpenAPI specification string.
//
// Why: The write-only bytes_value attribute must be validated for syntactic correctness and
// decoded byte length (<= 5 MiB) as well as hashed over the raw specification bytes.
//
// How: Decodes encoded using base64.StdEncoding. Returns an error without echoing large input strings.
//
// Parameters:
//   - encoded: Base64-encoded specification string.
//
// Returns:
//   - []byte: decoded raw specification bytes.
//   - error: non-nil if encoded is not valid Base64.
//
// Example:
//
//	raw, err := decodeBase64Spec("eyJvcGVuYXBpIjoiMy4wLjAifQ==")
func decodeBase64Spec(encoded string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("value is not valid Base64 encoding: %w", err)
	}
	return decoded, nil
}

// applyWriteOnlyConfigOverlay copies configured write-only attributes from req.Config into target.
//
// Why: Terraform Plugin Framework guarantees that write-only attributes are stripped (null) in
// req.Plan and req.State, so Create, Update, and ModifyPlan must read them from req.Config.
//
// How: Returns early when cfg.Raw is null (preserving compatibility with unit tests that omit
// req.Config); otherwise reads the config model and overlays string_value and bytes_value onto
// target, returning whether content_hash was explicitly configured.
//
// Parameters:
//   - ctx: execution context.
//   - cfg: Terraform configuration object from request.Config.
//   - target: destination model receiving write-only attribute values.
//
// Returns:
//   - bool: true when content_hash is explicitly set (non-null) in cfg.
//   - diag.Diagnostics: any diagnostics encountered while decoding cfg.
//
// Example:
//
//	hasHash, diags := applyWriteOnlyConfigOverlay(ctx, req.Config, &plan)
func applyWriteOnlyConfigOverlay(ctx context.Context, cfg tfsdk.Config, target *ResourceF5XCApiDefinitionUploaderModel) (bool, diag.Diagnostics) {
	if cfg.Raw.IsNull() {
		return false, nil
	}

	var configModel ResourceF5XCApiDefinitionUploaderModel
	diags := cfg.Get(ctx, &configModel)
	if diags.HasError() {
		return false, diags
	}

	if !configModel.StringValue.IsNull() {
		target.StringValue = configModel.StringValue
	}
	if !configModel.BytesValue.IsNull() {
		target.BytesValue = configModel.BytesValue
	}

	return !configModel.ContentHash.IsNull(), diags
}

// clearWriteOnlyFields resets write-only attributes on model to null before persisting state or plan.
//
// Why: Ensures write-only OpenAPI specifications are never persisted in Terraform state or plan snapshots.
//
// How: Assigns types.StringNull() to StringValue and BytesValue.
//
// Parameters:
//   - model: state or plan model to sanitize.
//
// Returns:
//   - None
//
// Example:
//
//	clearWriteOnlyFields(&data)
func clearWriteOnlyFields(model *ResourceF5XCApiDefinitionUploaderModel) {
	model.StringValue = types.StringNull()
	model.BytesValue = types.StringNull()
}

// resolveSpecContent extracts the active API specification content and SHA-256 digest from model.
//
// Why: Centralizes handling across string_value and bytes_value so Create, Update, and ModifyPlan
// apply consistent encoding, size checking, and hashing rules.
//
// How: Checks StringValue and BytesValue, validates the 5 MiB limit for both string_value and
// decoded bytes_value (plus Base64 decoding for bytes_value), and returns a normalized
// resolvedSpecContent descriptor.
//
// Parameters:
//   - model: resource model populated from Config (or Plan in unit tests without Config).
//
// Returns:
//   - resolvedSpecContent: normalized content, wire mode, and computed SHA-256 digest.
//   - diag.Diagnostics: attribute errors if Base64 decoding fails or size exceeds 5 MiB.
//
// Example:
//
//	spec, diags := resolveSpecContent(&data)
func resolveSpecContent(model *ResourceF5XCApiDefinitionUploaderModel) (resolvedSpecContent, diag.Diagnostics) {
	var diags diag.Diagnostics

	if !model.StringValue.IsNull() && !model.StringValue.IsUnknown() {
		val := model.StringValue.ValueString()
		if len(val) > maxStoredObjectSpecBytes {
			diags.AddAttributeError(
				path.Root("string_value"),
				"Specification Exceeds Maximum Size",
				fmt.Sprintf("string_value size (%d bytes) exceeds maximum allowed size of %d bytes (5 MiB).", len(val), maxStoredObjectSpecBytes),
			)
			return resolvedSpecContent{}, diags
		}
		return resolvedSpecContent{
			attrName:     "string_value",
			stringValue:  val,
			computedHash: computeSHA256(val),
			present:      true,
		}, diags
	}

	if !model.BytesValue.IsNull() && !model.BytesValue.IsUnknown() {
		val := model.BytesValue.ValueString()
		decoded, err := decodeBase64Spec(val)
		if err != nil {
			diags.AddAttributeError(path.Root("bytes_value"), "Invalid Base64 Value", err.Error())
			return resolvedSpecContent{}, diags
		}
		if len(decoded) > maxStoredObjectSpecBytes {
			diags.AddAttributeError(
				path.Root("bytes_value"),
				"Specification Exceeds Maximum Size",
				fmt.Sprintf("Decoded bytes_value size (%d bytes) exceeds maximum allowed size of %d bytes (5 MiB).", len(decoded), maxStoredObjectSpecBytes),
			)
			return resolvedSpecContent{}, diags
		}
		return resolvedSpecContent{
			attrName:     "bytes_value",
			isBytes:      true,
			bytesValue:   val,
			computedHash: computeSHA256Bytes(decoded),
			present:      true,
		}, diags
	}

	return resolvedSpecContent{}, diags
}

// buildUploaderRequest constructs the F5 XC CreateObjectRequest payload for PUT operations.
//
// Why: Ensures only the selected oneof content field (string_value or bytes_value) is serialized
// on the wire when uploading or updating a stored OpenAPI specification.
//
// How: Sets Description and routes either bytesValue or stringValue based on spec.isBytes.
//
// Parameters:
//   - desc: object description string from metadata.description.
//   - spec: resolved API specification content from resolveSpecContent.
//
// Returns:
//   - client.ResourceF5XCApiDefinitionUploaderCreateRequest: payload ready for JSON marshaling.
//
// Example:
//
//	payload := buildUploaderRequest(desc, spec)
func buildUploaderRequest(desc string, spec resolvedSpecContent) client.ResourceF5XCApiDefinitionUploaderCreateRequest {
	req := client.ResourceF5XCApiDefinitionUploaderCreateRequest{
		Description: desc,
	}
	if spec.isBytes {
		req.BytesValue = spec.bytesValue
	} else {
		req.StringValue = spec.stringValue
	}
	return req
}

// resolveAbsoluteURL resolves a target API path against baseURL without duplicating an "/api" path prefix.
//
// Why: RestClient.URL typically includes a trailing "/api" segment (e.g. "https://tenant.console.ves.volterra.io/api"),
// while F5 XC stored object version metadata URLs already begin with "/api/object_store/...". Naive concatenation
// produced invalid "/api/api/object_store/..." URLs that failed DELETE requests silently.
//
// How: Returns targetURL unchanged if it already carries an HTTP(S) scheme. Otherwise normalizes slashes and,
// when trimmedBase ends with "/api" and targetURL is "/api" or starts with "/api/", strips the redundant "/api"
// prefix from targetURL before joining.
//
// Parameters:
//   - baseURL: base API URL configured on the REST client (e.g. "https://tenant.console.ves.volterra.io/api")
//   - targetURL: relative path or absolute URL returned by the API or constructed locally
//
// Returns:
//   - string: properly joined absolute URL without duplicate "/api" segments
//
// Example:
//
//	u := resolveAbsoluteURL("https://tenant.example.com/api", "/api/object_store/namespaces/shared/stored_objects/swagger/foo/v1")
//	// u == "https://tenant.example.com/api/object_store/namespaces/shared/stored_objects/swagger/foo/v1"
func resolveAbsoluteURL(baseURL, targetURL string) string {
	if strings.HasPrefix(targetURL, "http://") || strings.HasPrefix(targetURL, "https://") {
		return targetURL
	}

	trimmedBase := strings.TrimRight(baseURL, "/")
	normalizedTarget := "/" + strings.TrimLeft(targetURL, "/")

	if strings.HasSuffix(trimmedBase, "/api") {
		if normalizedTarget == "/api" {
			return trimmedBase
		}
		if strings.HasPrefix(normalizedTarget, "/api/") {
			normalizedTarget = strings.TrimPrefix(normalizedTarget, "/api")
		}
	}

	return trimmedBase + normalizedTarget
}

// matchesStoredObjectName checks whether a ListObjects item name refers to the target object in namespace.
//
// Why: F5 XC Object Store list responses format item.Name as "<namespace>/<name>" (e.g. "shared/my-api"),
// whereas some fixtures or endpoints return the bare "<name>". Matching both forms prevents silent
// pruning skips while isolating identically named objects across namespaces.
//
// How: Compares itemName against bare name first and, when namespace is non-empty, against "namespace/name".
//
// Parameters:
//   - itemName: the item.Name value returned by the stored objects list API
//   - namespace: target F5 XC namespace of the resource
//   - name: target stored object name (without namespace prefix)
//
// Returns:
//   - bool: true when itemName matches either exact "name" or "namespace/name", false otherwise
//
// Example:
//
//	ok := matchesStoredObjectName("shared/my-api", "shared", "my-api") // true
func matchesStoredObjectName(itemName, namespace, name string) bool {
	if itemName == name {
		return true
	}
	if namespace != "" && itemName == namespace+"/"+name {
		return true
	}
	return false
}

// parseStoredObjectVersionNumber extracts the leading numeric version component from an F5 XC stored object version tag.
//
// Why: Stored object versions follow formats like "v10-26-11-18" and "v2-26-11-18". When creation timestamps
// are absent or identical, plain lexicographical sorting orders "v9-..." after "v10-...", causing newer versions
// to be pruned instead of older ones.
//
// How: Strips a leading "v", splits on "-" to isolate the numeric counter token, and parses it with strconv.Atoi.
//
// Parameters:
//   - version: version identifier string (e.g. "v4-26-11-18" or "v12")
//
// Returns:
//   - int: parsed version counter when valid, otherwise 0
//   - bool: true if the version string matched the "v<digits>..." format and parsed cleanly, false otherwise
//
// Example:
//
//	n, ok := parseStoredObjectVersionNumber("v12-26-11-18") // n == 12, ok == true
func parseStoredObjectVersionNumber(version string) (int, bool) {
	if !strings.HasPrefix(version, "v") {
		return 0, false
	}
	rest := strings.TrimPrefix(version, "v")
	if idx := strings.IndexByte(rest, '-'); idx >= 0 {
		rest = rest[:idx]
	}
	if rest == "" {
		return 0, false
	}
	num, err := strconv.Atoi(rest)
	if err != nil {
		return 0, false
	}
	return num, true
}

// parseStoredObjectTimestamp parses an RFC3339 or RFC3339Nano timestamp string.
//
// Why: Keeps timestamp parsing consistent and avoids duplicating format fallback logic inside sort comparators.
//
// How: Attempts time.Parse with time.RFC3339Nano first, falling back to time.RFC3339 on failure.
//
// Parameters:
//   - ts: raw timestamp string from the API version item
//
// Returns:
//   - time.Time: parsed timestamp when valid, zero Time otherwise
//   - bool: true if parsing succeeded, false if empty or unparseable
//
// Example:
//
//	parsed, ok := parseStoredObjectTimestamp("2026-11-18T10:00:01Z")
func parseStoredObjectTimestamp(ts string) (time.Time, bool) {
	if ts == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.RFC3339, ts); err == nil {
		return t, true
	}
	return time.Time{}, false
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
// How: Constructs a framework schema.Schema mapping out metadata, write-only string_value and
// bytes_value inputs, content_hash, retain_versions, previous_version_url, force_delete, and
// timeout settings.
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

The ` + "`f5xc_api_definition_uploader`" + ` resource uploads raw or Base64-encoded OpenAPI or Swagger specifications to the F5 Distributed Cloud (F5 XC) Object Store. It exports the backend-generated versioned URL in ` + "`metadata.url`" + `, which can be passed to ` + "`f5xc_api_definition`" + ` for API Discovery, API Inventory, and API Protection policies.

## Declarative Upload & State Hygiene

Both ` + "`string_value`" + ` and ` + "`bytes_value`" + ` are write-only attributes (requires Terraform >= 1.11). Multi-megabyte OpenAPI payloads are sent only during ` + "`Create`" + ` and ` + "`Update`" + ` and are never persisted in Terraform state or plan files. Change detection is driven by ` + "`content_hash`" + ` (automatically computed as the SHA-256 digest when omitted, or explicitly set via e.g. ` + "`filesha256(...)`" + `).

## Object Store Integration

Uploads the specification with a PUT request to the F5 XC Object Store REST API:

	PUT /object_store/namespaces/{namespace}/stored_objects/swagger/{name}

## Lifecycle Management

The resource manages the object lifecycle:

- Create uploads the specification.
- Read retrieves the stored object metadata through ` + "`metadata.url`" + `.
- Update replaces the stored specification only when content or description changes.
- Delete removes the stored object.

## Computed URL Export

Captures the backend-generated versioned URL in ` + "`metadata.url`" + ` for use by dependent ` + "`f5xc_api_definition`" + ` resources.`,
		Description: `f5xc_api_definition_uploader uploads raw or Base64-encoded OpenAPI or Swagger specifications to the F5 Distributed Cloud Object Store using write-only attributes (string_value, bytes_value) alongside content_hash change detection (requires Terraform >= 1.11), manages create, read, update, and delete operations, and exports the backend-generated versioned URL in metadata.url for use by f5xc_api_definition.`,

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
				Description: "string_value Write-only raw OpenAPI specification content (JSON or YAML). Sent only during Create and Update operations and never stored in Terraform state or plans (requires Terraform >= 1.11).",
				Required:    false,
				Optional:    true,
				Computed:    false,
				WriteOnly:   true,
				Validators: []validator.String{
					bytesvalidator.BytesAtMost(maxStoredObjectSpecBytes),
				},
			},
			"bytes_value": schema.StringAttribute{
				Description: "bytes_value Write-only Base64-encoded OpenAPI specification content (e.g. filebase64(...)). Sent only during Create and Update operations and never stored in Terraform state or plans (requires Terraform >= 1.11).",
				Required:    false,
				Optional:    true,
				Computed:    false,
				WriteOnly:   true,
			},
			"content_hash": schema.StringAttribute{
				Description: "content_hash SHA-256 hex digest or user-supplied version trigger used to detect API specification modifications and prevent version churn. Computed automatically from the specification when omitted.",
				Required:    false,
				Optional:    true,
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
// Why: Validates mutual exclusivity between string_value and bytes_value, as well as Base64
// encoding and 5 MiB payload limits, before planning.
//
// How: Reads the configuration model, verifies that exactly one of string_value or bytes_value
// is configured, and validates Base64 decoding and byte size for bytes_value.
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

	// Delay mutual-exclusivity and presence checks if either write-only attribute is unknown
	// (e.g., when passed via module variables or computed expressions during static terraform validate),
	// while still validating any known payload attribute against encoding and 5 MiB size limits.
	if data.StringValue.IsUnknown() || data.BytesValue.IsUnknown() {
		if (!data.StringValue.IsNull() && !data.StringValue.IsUnknown()) || (!data.BytesValue.IsNull() && !data.BytesValue.IsUnknown()) {
			_, specDiags := resolveSpecContent(&data)
			resp.Diagnostics.Append(specDiags...)
		}
		return
	}

	configured := make([]string, 0, 2)
	if !data.StringValue.IsNull() {
		configured = append(configured, `"string_value"`)
	}
	if !data.BytesValue.IsNull() {
		configured = append(configured, `"bytes_value"`)
	}

	if len(configured) == 0 {
		resp.Diagnostics.AddError(
			"Missing API Specification Content",
			`Exactly one of "string_value" or "bytes_value" must be configured.`,
		)
		return
	}

	if len(configured) > 1 {
		resp.Diagnostics.AddError(
			"Conflicting API Specification Content",
			fmt.Sprintf(`Exactly one of "string_value" or "bytes_value" may be configured; found %s.`, strings.Join(configured, ", ")),
		)
		return
	}

	// Validate string_value size and bytes_value Base64 encoding and decoded size against the 5 MiB API limit.
	_, specDiags := resolveSpecContent(&data)
	resp.Diagnostics.Append(specDiags...)
}

// ModifyPlan implements resource.ResourceWithModifyPlan.
//
// Why: Dynamically marks versioned attributes as unknown during an update plan when content or
// description changes, preserves computed content_hash across no-op plans for write-only inputs,
// and ensures downstream resources (such as f5xc_api_definition) plan their updates accurately.
//
// How: Overlays write-only values from req.Config, evaluates changes via content_hash (user-supplied
// or computed SHA-256), marks metadata.url, metadata.version, metadata.creation_timestamp, and status
// as unknown when changed, and keeps write-only attributes null in the planned state.
//
// Parameters:
//   - ctx: execution context.
//   - req: modify plan request containing prior state, config, and proposed plan.
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

	effectivePlan := plan
	hasConfigContentHash, cfgDiags := applyWriteOnlyConfigOverlay(ctx, req.Config, &effectivePlan)
	resp.Diagnostics.Append(cfgDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	contentUnknown := effectivePlan.StringValue.IsUnknown() || effectivePlan.BytesValue.IsUnknown()

	var contentChanged bool
	if hasConfigContentHash {
		if plan.ContentHash.IsUnknown() {
			contentChanged = true
		} else {
			contentChanged = state.ContentHash.IsNull() || plan.ContentHash.ValueString() != state.ContentHash.ValueString()
		}
	} else if contentUnknown {
		contentChanged = true
	} else {
		spec, specDiags := resolveSpecContent(&effectivePlan)
		resp.Diagnostics.Append(specDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if spec.present {
			contentChanged = state.ContentHash.IsNull() || spec.computedHash != state.ContentHash.ValueString()
		}
	}

	descChanged := !plan.Metadata.Description.IsUnknown() && !state.Metadata.Description.IsNull() && plan.Metadata.Description.ValueString() != state.Metadata.Description.ValueString()

	planModified := false
	// If content or metadata.description is changing, mark versioned attributes as unknown so dependents plan for update.
	if contentChanged || descChanged {
		plan.Metadata.Url = types.StringUnknown()
		plan.Metadata.Version = types.StringUnknown()
		plan.Metadata.CreationTimestamp = types.StringUnknown()
		plan.PreviousVersionUrl = types.StringUnknown()
		if contentChanged && !hasConfigContentHash {
			plan.ContentHash = types.StringUnknown()
		}
		plan.Status = types.StringUnknown()
		planModified = true
	} else if !hasConfigContentHash && plan.ContentHash.IsUnknown() && !state.ContentHash.IsNull() {
		// Preserve state content_hash on unchanged write-only plans when content_hash is omitted from HCL.
		plan.ContentHash = state.ContentHash
		planModified = true
	}

	clearWriteOnlyFields(&plan)
	if planModified {
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

// Create uploads the OpenAPI spec to the F5 XC Object Store and saves the immutable version URL in state.
//
// Why: Initial creation of the stored object version in F5 XC.
//
// How: Overlays write-only attributes from req.Config, resolves the active specification payload
// (string_value or bytes_value), executes HTTP PUT, decodes the response, and sets state attributes
// while keeping write-only attributes null in state.
//
// Parameters:
//   - ctx: execution context.
//   - request: create request with plan and config data.
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

	_, cfgDiags := applyWriteOnlyConfigOverlay(ctx, request.Config, &data)
	response.Diagnostics.Append(cfgDiags...)
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

	spec, specDiags := resolveSpecContent(&data)
	response.Diagnostics.Append(specDiags...)
	if response.Diagnostics.HasError() {
		return
	}
	if !spec.present {
		response.Diagnostics.AddError(
			"Missing API Specification Content",
			`Exactly one of "string_value" or "bytes_value" must be provided.`,
		)
		return
	}

	effectiveHash := spec.computedHash
	if !data.ContentHash.IsNull() && !data.ContentHash.IsUnknown() {
		effectiveHash = data.ContentHash.ValueString()
	}

	desc := data.Metadata.Description.ValueString()
	createReq := buildUploaderRequest(desc, spec)

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
	data.ContentHash = types.StringValue(effectiveHash)
	data.PreviousVersionUrl = types.StringNull()
	clearWriteOnlyFields(&data)

	tflog.Info(ctx, fmt.Sprintf("Created Api definition uploader resource: %s version: %s", data.Metadata.Name.ValueString(), data.Metadata.Version.ValueString()))

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

// Read refreshes the resource state by querying the F5 XC Object Store.
//
// Why: Detects configuration drift and external deletion without persisting write-only specs in state.
//
// How: Executes HTTP GET against metadata.url. If HTTP 404 is encountered, cleanly removes the
// resource from state. Otherwise, refreshes metadata, leaves string_value and bytes_value null, and
// preserves existing content_hash (computing it from the response only if absent, e.g. on import).
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
	clearWriteOnlyFields(&data)

	if data.ContentHash.IsNull() || data.ContentHash.ValueString() == "" {
		if readResp.BytesValue != "" {
			if decoded, decErr := decodeBase64Spec(readResp.BytesValue); decErr == nil {
				data.ContentHash = types.StringValue(computeSHA256Bytes(decoded))
			} else {
				data.ContentHash = types.StringValue(computeSHA256(readResp.BytesValue))
			}
		} else if readResp.StringValue != "" {
			data.ContentHash = types.StringValue(computeSHA256(readResp.StringValue))
		}
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

// Update uploads a new version of the OpenAPI spec when content or description changes, and triggers retention pruning.
//
// Why: Enables automated rollout of new API definitions while avoiding version churn on unchanged specs
// and pruning stale historical versions.
//
// How: Overlays write-only values from req.Config, resolves the active payload and effective hash, and
// compares against prior state. If unchanged, skips HTTP PUT and updates non-content state fields.
// If changed, PUTs the new version, captures the new URL and version, sets previous_version_url,
// keeps write-only attributes null, and calls pruneHistoricalVersions.
//
// Parameters:
//   - ctx: execution context.
//   - request: update request containing plan, config, and prior state.
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

	effectivePlan := plan
	_, cfgDiags := applyWriteOnlyConfigOverlay(ctx, request.Config, &effectivePlan)
	response.Diagnostics.Append(cfgDiags...)
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

	spec, specDiags := resolveSpecContent(&effectivePlan)
	response.Diagnostics.Append(specDiags...)
	if response.Diagnostics.HasError() {
		return
	}
	if !spec.present {
		response.Diagnostics.AddError(
			"Missing API Specification Content",
			`Exactly one of "string_value" or "bytes_value" must be provided.`,
		)
		return
	}

	effectiveHash := spec.computedHash
	if !plan.ContentHash.IsNull() && !plan.ContentHash.IsUnknown() {
		effectiveHash = plan.ContentHash.ValueString()
	}

	contentChanged := state.ContentHash.IsNull() || state.ContentHash.ValueString() != effectiveHash
	descChanged := plan.Metadata.Description.ValueString() != state.Metadata.Description.ValueString()

	// If neither content nor description changed, preserve current backend version and avoid version churn.
	if !contentChanged && !descChanged {
		tflog.Debug(ctx, "Api definition uploader content unchanged; skipping upload to prevent version churn")
		state.ForceDelete = plan.ForceDelete
		state.RetainVersions = plan.RetainVersions
		state.Timeouts = plan.Timeouts
		clearWriteOnlyFields(&state)

		if !plan.RetainVersions.IsNull() && !plan.RetainVersions.IsUnknown() && plan.RetainVersions.ValueInt64() > 0 {
			response.Diagnostics.Append(r.pruneHistoricalVersions(ctx, state.Metadata.Namespace.ValueString(), state.Metadata.Name.ValueString(), plan.RetainVersions.ValueInt64(), state.Metadata.Version.ValueString())...)
		}

		response.Diagnostics.Append(response.State.Set(ctx, &state)...)
		return
	}

	desc := plan.Metadata.Description.ValueString()
	updateReq := buildUploaderRequest(desc, spec)

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
	plan.ContentHash = types.StringValue(effectiveHash)
	clearWriteOnlyFields(&plan)

	tflog.Info(ctx, fmt.Sprintf("Updated Api definition uploader resource: %s new version: %s (previous: %s)",
		plan.Metadata.Name.ValueString(), plan.Metadata.Version.ValueString(), state.Metadata.Version.ValueString()))

	// Perform retention pruning if configured
	retainCount := int64(2)
	if !plan.RetainVersions.IsNull() && !plan.RetainVersions.IsUnknown() {
		retainCount = plan.RetainVersions.ValueInt64()
	}
	response.Diagnostics.Append(r.pruneHistoricalVersions(ctx, plan.Metadata.Namespace.ValueString(), plan.Metadata.Name.ValueString(), retainCount, plan.Metadata.Version.ValueString())...)

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

// pruneHistoricalVersions lists stored versions for the object and deletes historical versions beyond retainCount.
//
// Why: Repeated updates to an f5xc_api_definition_uploader resource accumulate immutable object versions on the
// F5 XC Object Store backend. When retain_versions is positive, automated pruning prevents unbounded version growth
// while preserving the active currentVersion and the newest retainCount versions.
//
// How: Queries the namespace's stored swagger objects without the broken ?name= query parameter, matches the target
// object by bare or namespace-prefixed name, sorts versions newest-first (by creation timestamp, then currentVersion
// priority, then parsed numeric version counter, then string order), and sends DELETE requests for versions past
// retainCount using properly resolved URLs. Failures are logged and surfaced as non-fatal warning diagnostics.
//
// Parameters:
//   - ctx: context used for API requests and Terraform logging
//   - namespace: F5 XC namespace containing the stored object
//   - name: stored object name (without namespace prefix)
//   - retainCount: maximum number of newest versions to keep (<= 0 disables pruning)
//   - currentVersion: version identifier currently active in state so it is never deleted
//
// Returns:
//   - diag.Diagnostics: warning diagnostics if listing, decoding, or deleting historical versions fails
//
// Example:
//
//	diags := r.pruneHistoricalVersions(ctx, "shared", "my-api", 2, "v4-26-11-18")
func (r *ResourceF5XCApiDefinitionUploader) pruneHistoricalVersions(
	ctx context.Context,
	namespace, name string,
	retainCount int64,
	currentVersion string,
) diag.Diagnostics {
	var diags diag.Diagnostics
	if retainCount <= 0 {
		return diags
	}

	listURL := fmt.Sprintf("%s/object_store/namespaces/%s/stored_objects/swagger",
		r.client.RestClient.URL, namespace)

	respBytes, err := r.client.RestClient.Execute(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		msg := fmt.Sprintf("Failed to list stored object versions for pruning: %s", err)
		tflog.Warn(ctx, msg)
		diags.AddWarning("Failed to List Historical API Definition Versions", msg)
		return diags
	}

	var listResp storedObjectListResponse
	if unmarshalErr := json.Unmarshal(respBytes, &listResp); unmarshalErr != nil {
		msg := fmt.Sprintf("Failed to decode stored object versions for pruning: %s", unmarshalErr)
		tflog.Warn(ctx, msg)
		diags.AddWarning("Failed to Decode Historical API Definition Versions", msg)
		return diags
	}

	for _, item := range listResp.Items {
		if !matchesStoredObjectName(item.Name, namespace, name) {
			continue
		}

		if int64(len(item.Versions)) <= retainCount {
			return diags
		}

		// Sort newest first
		versions := make([]storedObjectVersionItem, len(item.Versions))
		copy(versions, item.Versions)

		sort.Slice(versions, func(i, j int) bool {
			t1, ok1 := parseStoredObjectTimestamp(versions[i].effectiveCreationTimestamp())
			t2, ok2 := parseStoredObjectTimestamp(versions[j].effectiveCreationTimestamp())
			if ok1 && ok2 && !t1.Equal(t2) {
				return t1.After(t2)
			}
			if currentVersion != "" && versions[i].Version != versions[j].Version {
				if versions[i].Version == currentVersion {
					return true
				}
				if versions[j].Version == currentVersion {
					return false
				}
			}
			n1, tok1 := parseStoredObjectVersionNumber(versions[i].Version)
			n2, tok2 := parseStoredObjectVersionNumber(versions[j].Version)
			if tok1 && tok2 && n1 != n2 {
				return n1 > n2
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
				msg := fmt.Sprintf("Failed to prune historical version %s of %s/%s: %s", v.Version, namespace, name, delErr)
				tflog.Warn(ctx, msg)
				diags.AddWarning("Failed to Prune Historical API Definition Version", msg)
			}
		}

		return diags
	}

	return diags
}
