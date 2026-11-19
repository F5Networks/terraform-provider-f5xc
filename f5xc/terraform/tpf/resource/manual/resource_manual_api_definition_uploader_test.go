//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package manual

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/client"
)

// mockAuthenticator is a minimal client.Authenticator for testing with httptest.Server.
type mockAuthenticator struct {
	httpClient *http.Client
}

func (a *mockAuthenticator) ApplyAuth(*http.Request) error { return nil }
func (a *mockAuthenticator) GetClient() *http.Client       { return a.httpClient }
func (a *mockAuthenticator) GetClientData() string         { return "" }

// newMockAPIClient creates an APIClient backed by an in-process httptest.Server.
func newMockAPIClient(srv *httptest.Server) *client.APIClient {
	auth := &mockAuthenticator{httpClient: srv.Client()}
	return &client.APIClient{RestClient: client.NewAuthClient(srv.URL, auth, "test-agent")}
}

func testTimeoutsValue() timeouts.Value {
	return timeouts.Value{
		Object: types.ObjectNull(map[string]attr.Type{
			"create": types.StringType,
			"read":   types.StringType,
			"update": types.StringType,
			"delete": types.StringType,
		}),
	}
}

func TestComputeSHA256(t *testing.T) {
	content := `{"openapi":"3.0.0","info":{"title":"Sample API","version":"1.0.0"}}`
	hash1 := computeSHA256(content)
	hash2 := computeSHA256(content)

	assert.NotEmpty(t, hash1)
	assert.Equal(t, hash1, hash2, "Identical content must produce identical hash")

	differentContent := `{"openapi":"3.0.0","info":{"title":"Sample API","version":"1.0.1"}}`
	hash3 := computeSHA256(differentContent)
	assert.NotEqual(t, hash1, hash3, "Different content must produce different hash")
}

func TestResourceF5XCApiDefinitionUploader_Schema(t *testing.T) {
	res := NewResourceF5XCApiDefinitionUploader()
	var resp resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	require.False(t, resp.Diagnostics.HasError())
	require.NotNil(t, resp.Schema)

	attrs := resp.Schema.Attributes
	assert.Contains(t, attrs, "metadata")
	assert.Contains(t, attrs, "string_value")
	assert.Contains(t, attrs, "bytes_value")
	assert.NotContains(t, attrs, "string_value_wo")
	assert.NotContains(t, attrs, "bytes_value_wo")
	assert.Contains(t, attrs, "content_hash")
	assert.Contains(t, attrs, "retain_versions")
	assert.Contains(t, attrs, "previous_version_url")
	assert.Contains(t, attrs, "force_delete")
	assert.Contains(t, attrs, "status")
	assert.Contains(t, attrs, "timeouts")

	stringValAttr, ok := attrs["string_value"].(rschema.StringAttribute)
	require.True(t, ok)
	assert.True(t, stringValAttr.Optional)
	assert.False(t, stringValAttr.Required)
	assert.False(t, stringValAttr.Computed)
	assert.True(t, stringValAttr.WriteOnly)
	assert.Empty(t, stringValAttr.DeprecationMessage)

	bytesValAttr, ok := attrs["bytes_value"].(rschema.StringAttribute)
	require.True(t, ok)
	assert.True(t, bytesValAttr.Optional)
	assert.False(t, bytesValAttr.Required)
	assert.False(t, bytesValAttr.Computed)
	assert.True(t, bytesValAttr.WriteOnly)
	assert.Empty(t, bytesValAttr.DeprecationMessage)

	contentHashAttr, ok := attrs["content_hash"].(rschema.StringAttribute)
	require.True(t, ok)
	assert.True(t, contentHashAttr.Optional)
	assert.True(t, contentHashAttr.Computed)
}

func TestResourceF5XCApiDefinitionUploader_Create(t *testing.T) {
	specContent := `{"openapi":"3.0.0","info":{"title":"Test","version":"1.0.0"}}`
	expectedHash := computeSHA256(specContent)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.True(t, strings.HasSuffix(r.URL.Path, "/object_store/namespaces/shared/stored_objects/swagger/my-api"))

		resp := apiDefinitionUploaderResponse{
			Status: "STORED_OBJECT_STATUS_CREATED",
			Metadata: apiDefinitionUploaderMetadataResponse{
				Url:               r.URL.String() + "/v1-26-09-24",
				Name:              "my-api",
				Version:           "v1-26-09-24",
				Namespace:         "shared",
				Description:       "",
				CreationTimestamp: "2026-09-24T12:00:00Z",
			},
			StringValue: specContent,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	res := &ResourceF5XCApiDefinitionUploader{client: newMockAPIClient(srv)}

	var schemaResp resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	planData := ResourceF5XCApiDefinitionUploaderModel{
		Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
			Name:        types.StringValue("my-api"),
			Namespace:   types.StringValue("shared"),
			Description: types.StringValue(""),
		},
		StringValue:    types.StringValue(specContent),
		RetainVersions: types.Int64Value(2),
		ForceDelete:    types.BoolValue(false),
		Timeouts:       testTimeoutsValue(),
	}

	planState := tfsdk.State{Schema: schemaResp.Schema}
	diags := planState.Set(context.Background(), &planData)
	require.False(t, diags.HasError())

	req := resource.CreateRequest{Plan: tfsdk.Plan(planState)}
	var resp resource.CreateResponse
	resp.State = tfsdk.State{Schema: schemaResp.Schema}

	res.Create(context.Background(), req, &resp)
	require.False(t, resp.Diagnostics.HasError())

	var createdData ResourceF5XCApiDefinitionUploaderModel
	diags = resp.State.Get(context.Background(), &createdData)
	require.False(t, diags.HasError())

	assert.Equal(t, "v1-26-09-24", createdData.Metadata.Version.ValueString())
	assert.Equal(t, expectedHash, createdData.ContentHash.ValueString())
	assert.Contains(t, createdData.Metadata.Url.ValueString(), "/v1-26-09-24")
	assert.Equal(t, "", createdData.Metadata.Description.ValueString())
	assert.True(t, createdData.PreviousVersionUrl.IsNull())
}

func TestResourceF5XCApiDefinitionUploader_Read_Success(t *testing.T) {
	specContent := `{"openapi":"3.0.0"}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)

		resp := apiDefinitionUploaderResponse{
			Metadata: apiDefinitionUploaderMetadataResponse{
				Url:               r.URL.String(),
				Name:              "my-api",
				Version:           "v1-26-09-24",
				Namespace:         "shared",
				Description:       "",
				CreationTimestamp: "2026-09-24T12:00:00Z",
			},
			StringValue: specContent,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	res := &ResourceF5XCApiDefinitionUploader{client: newMockAPIClient(srv)}

	var schemaResp resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	stateData := ResourceF5XCApiDefinitionUploaderModel{
		Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
			Url:       types.StringValue(srv.URL + "/object_store/namespaces/shared/stored_objects/swagger/my-api/v1-26-09-24"),
			Name:      types.StringValue("my-api"),
			Namespace: types.StringValue("shared"),
		},
		StringValue: types.StringValue(specContent),
		Timeouts:    testTimeoutsValue(),
	}

	state := tfsdk.State{Schema: schemaResp.Schema}
	diags := state.Set(context.Background(), &stateData)
	require.False(t, diags.HasError())

	req := resource.ReadRequest{State: state}
	var resp resource.ReadResponse
	resp.State = state

	res.Read(context.Background(), req, &resp)
	require.False(t, resp.Diagnostics.HasError())

	var refreshed ResourceF5XCApiDefinitionUploaderModel
	diags = resp.State.Get(context.Background(), &refreshed)
	require.False(t, diags.HasError())

	assert.True(t, refreshed.StringValue.IsNull(), "Write-only string_value must not be stored in state after Read")
	assert.Equal(t, "v1-26-09-24", refreshed.Metadata.Version.ValueString())
	assert.Equal(t, computeSHA256(specContent), refreshed.ContentHash.ValueString())
}

func TestResourceF5XCApiDefinitionUploader_Read_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code": 5, "message": "Object not found"}`))
	}))
	defer srv.Close()

	res := &ResourceF5XCApiDefinitionUploader{client: newMockAPIClient(srv)}

	var schemaResp resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	stateData := ResourceF5XCApiDefinitionUploaderModel{
		Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
			Url:       types.StringValue(srv.URL + "/object_store/namespaces/shared/stored_objects/swagger/my-api/v1-26-09-24"),
			Name:      types.StringValue("my-api"),
			Namespace: types.StringValue("shared"),
		},
		Timeouts: testTimeoutsValue(),
	}

	state := tfsdk.State{Schema: schemaResp.Schema}
	diags := state.Set(context.Background(), &stateData)
	require.False(t, diags.HasError())

	req := resource.ReadRequest{State: state}
	var resp resource.ReadResponse
	resp.State = state

	res.Read(context.Background(), req, &resp)
	require.False(t, resp.Diagnostics.HasError())
	assert.True(t, resp.State.Raw.IsNull(), "Resource must be removed from state on 404")
}

func TestResourceF5XCApiDefinitionUploader_Update_ChangedContent(t *testing.T) {
	oldSpec := `{"openapi":"3.0.0","version":"1.0"}`
	newSpec := `{"openapi":"3.0.0","version":"2.0"}`

	var mu sync.Mutex
	putCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if r.Method == http.MethodPut {
			putCount++
			var rawBody map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&rawBody))
			assert.NotContains(t, rawBody, "metadata", "CreateObjectRequest must not wrap fields inside metadata")
			descVal, _ := rawBody["description"].(string)
			assert.Equal(t, "Content update preserves description", descVal)

			resp := apiDefinitionUploaderResponse{
				Status: "STORED_OBJECT_STATUS_UPDATED",
				Metadata: apiDefinitionUploaderMetadataResponse{
					Url:               r.URL.String() + "/v2-26-09-24",
					Name:              "my-api",
					Version:           "v2-26-09-24",
					Namespace:         "shared",
					Description:       descVal,
					CreationTimestamp: "2026-09-24T13:00:00Z",
				},
				StringValue: newSpec,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/stored_objects/swagger") {
			// ListObjects for retention pruning
			listResp := storedObjectListResponse{
				Items: []storedObjectListItem{
					{
						Name: "my-api",
						Versions: []storedObjectVersionItem{
							{Version: "v2-26-09-24", CreationTimestamp: "2026-09-24T13:00:00Z"},
							{Version: "v1-26-09-24", CreationTimestamp: "2026-09-24T12:00:00Z"},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(listResp)
			return
		}
	}))
	defer srv.Close()

	res := &ResourceF5XCApiDefinitionUploader{client: newMockAPIClient(srv)}

	var schemaResp resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	stateData := ResourceF5XCApiDefinitionUploaderModel{
		Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
			Url:         types.StringValue(srv.URL + "/object_store/namespaces/shared/stored_objects/swagger/my-api/v1-26-09-24"),
			Name:        types.StringValue("my-api"),
			Version:     types.StringValue("v1-26-09-24"),
			Namespace:   types.StringValue("shared"),
			Description: types.StringValue("Content update preserves description"),
		},
		StringValue:    types.StringValue(oldSpec),
		ContentHash:    types.StringValue(computeSHA256(oldSpec)),
		RetainVersions: types.Int64Value(2),
		Timeouts:       testTimeoutsValue(),
	}

	planData := ResourceF5XCApiDefinitionUploaderModel{
		Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
			Name:        types.StringValue("my-api"),
			Namespace:   types.StringValue("shared"),
			Description: types.StringValue("Content update preserves description"),
		},
		StringValue:    types.StringValue(newSpec),
		RetainVersions: types.Int64Value(2),
		Timeouts:       testTimeoutsValue(),
	}

	state := tfsdk.State{Schema: schemaResp.Schema}
	require.False(t, state.Set(context.Background(), &stateData).HasError())

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	require.False(t, plan.Set(context.Background(), &planData).HasError())

	req := resource.UpdateRequest{Plan: plan, State: state}
	var resp resource.UpdateResponse
	resp.State = state

	res.Update(context.Background(), req, &resp)
	require.False(t, resp.Diagnostics.HasError())

	var updatedData ResourceF5XCApiDefinitionUploaderModel
	require.False(t, resp.State.Get(context.Background(), &updatedData).HasError())

	assert.Equal(t, 1, putCount, "PUT must be executed when content changes")
	assert.Equal(t, "v2-26-09-24", updatedData.Metadata.Version.ValueString())
	assert.Equal(t, "Content update preserves description", updatedData.Metadata.Description.ValueString())
	assert.Equal(t, computeSHA256(newSpec), updatedData.ContentHash.ValueString())
	assert.Equal(t, srv.URL+"/object_store/namespaces/shared/stored_objects/swagger/my-api/v1-26-09-24", updatedData.PreviousVersionUrl.ValueString())
}

func TestResourceF5XCApiDefinitionUploader_Update_ChangedDescription(t *testing.T) {
	spec := `{"openapi":"3.0.0","version":"1.0"}`
	hash := computeSHA256(spec)

	var mu sync.Mutex
	putCount := 0
	var receivedDesc string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if r.Method == http.MethodPut {
			putCount++
			var rawBody map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&rawBody))
			assert.NotContains(t, rawBody, "metadata", "CreateObjectRequest must not wrap fields inside metadata")
			descVal, _ := rawBody["description"].(string)
			receivedDesc = descVal

			resp := apiDefinitionUploaderResponse{
				Status: "STORED_OBJECT_STATUS_UPDATED",
				Metadata: apiDefinitionUploaderMetadataResponse{
					Url:               r.URL.String() + "/v2",
					Name:              "my-api",
					Version:           "v2",
					Namespace:         "shared",
					Description:       receivedDesc,
					CreationTimestamp: "2026-09-24T13:00:00Z",
				},
				StringValue: spec,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/stored_objects/swagger") {
			listResp := storedObjectListResponse{}
			_ = json.NewEncoder(w).Encode(listResp)
			return
		}
	}))
	defer srv.Close()

	res := &ResourceF5XCApiDefinitionUploader{client: newMockAPIClient(srv)}

	var schemaResp resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	stateData := ResourceF5XCApiDefinitionUploaderModel{
		Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
			Url:         types.StringValue(srv.URL + "/object_store/namespaces/shared/stored_objects/swagger/my-api/v1"),
			Name:        types.StringValue("my-api"),
			Version:     types.StringValue("v1"),
			Namespace:   types.StringValue("shared"),
			Description: types.StringValue("v1 desc"),
		},
		StringValue:    types.StringValue(spec),
		ContentHash:    types.StringValue(hash),
		RetainVersions: types.Int64Value(2),
		Timeouts:       testTimeoutsValue(),
	}

	planData := ResourceF5XCApiDefinitionUploaderModel{
		Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
			Name:        types.StringValue("my-api"),
			Namespace:   types.StringValue("shared"),
			Description: types.StringValue("v2 desc"),
		},
		StringValue:    types.StringValue(spec),
		RetainVersions: types.Int64Value(2),
		Timeouts:       testTimeoutsValue(),
	}

	state := tfsdk.State{Schema: schemaResp.Schema}
	require.False(t, state.Set(context.Background(), &stateData).HasError())

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	require.False(t, plan.Set(context.Background(), &planData).HasError())

	req := resource.UpdateRequest{Plan: plan, State: state}
	var resp resource.UpdateResponse
	resp.State = state

	res.Update(context.Background(), req, &resp)
	require.False(t, resp.Diagnostics.HasError())

	var updatedData ResourceF5XCApiDefinitionUploaderModel
	require.False(t, resp.State.Get(context.Background(), &updatedData).HasError())

	assert.Equal(t, 1, putCount, "PUT must be executed when description changes")
	assert.Equal(t, "v2 desc", receivedDesc)
	assert.Equal(t, "v2 desc", updatedData.Metadata.Description.ValueString())
	assert.Equal(t, "v2", updatedData.Metadata.Version.ValueString())
}

func TestResourceF5XCApiDefinitionUploader_Update_ClearDescription(t *testing.T) {
	spec := `{"openapi":"3.0.0","version":"1.0"}`
	hash := computeSHA256(spec)

	var mu sync.Mutex
	putCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if r.Method == http.MethodPut {
			putCount++
			var rawBody map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&rawBody))
			assert.NotContains(t, rawBody, "metadata", "CreateObjectRequest must not wrap fields inside metadata")
			descVal, _ := rawBody["description"].(string)
			assert.Empty(t, descVal, "Cleared description must be empty or omitted in PUT body")

			resp := apiDefinitionUploaderResponse{
				Status: "STORED_OBJECT_STATUS_UPDATED",
				Metadata: apiDefinitionUploaderMetadataResponse{
					Url:               r.URL.String() + "/v2",
					Name:              "my-api",
					Version:           "v2",
					Namespace:         "shared",
					Description:       "",
					CreationTimestamp: "2026-09-24T13:00:00Z",
				},
				StringValue: spec,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/stored_objects/swagger") {
			listResp := storedObjectListResponse{}
			_ = json.NewEncoder(w).Encode(listResp)
			return
		}
	}))
	defer srv.Close()

	res := &ResourceF5XCApiDefinitionUploader{client: newMockAPIClient(srv)}

	var schemaResp resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	stateData := ResourceF5XCApiDefinitionUploaderModel{
		Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
			Url:         types.StringValue(srv.URL + "/object_store/namespaces/shared/stored_objects/swagger/my-api/v1"),
			Name:        types.StringValue("my-api"),
			Version:     types.StringValue("v1"),
			Namespace:   types.StringValue("shared"),
			Description: types.StringValue("Temporary desc"),
		},
		StringValue:    types.StringValue(spec),
		ContentHash:    types.StringValue(hash),
		RetainVersions: types.Int64Value(2),
		Timeouts:       testTimeoutsValue(),
	}

	planData := ResourceF5XCApiDefinitionUploaderModel{
		Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
			Name:        types.StringValue("my-api"),
			Namespace:   types.StringValue("shared"),
			Description: types.StringValue(""),
		},
		StringValue:    types.StringValue(spec),
		RetainVersions: types.Int64Value(2),
		Timeouts:       testTimeoutsValue(),
	}

	state := tfsdk.State{Schema: schemaResp.Schema}
	require.False(t, state.Set(context.Background(), &stateData).HasError())

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	require.False(t, plan.Set(context.Background(), &planData).HasError())

	req := resource.UpdateRequest{Plan: plan, State: state}
	var resp resource.UpdateResponse
	resp.State = state

	res.Update(context.Background(), req, &resp)
	require.False(t, resp.Diagnostics.HasError())

	var updatedData ResourceF5XCApiDefinitionUploaderModel
	require.False(t, resp.State.Get(context.Background(), &updatedData).HasError())

	assert.Equal(t, 1, putCount, "PUT must be executed when clearing description")
	assert.Equal(t, "", updatedData.Metadata.Description.ValueString())
}

func TestResourceF5XCApiDefinitionUploader_Update_UnchangedContent_NoOp(t *testing.T) {
	spec := `{"openapi":"3.0.0"}`
	hash := computeSHA256(spec)

	putExecuted := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			putExecuted = true
		}
	}))
	defer srv.Close()

	res := &ResourceF5XCApiDefinitionUploader{client: newMockAPIClient(srv)}

	var schemaResp resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	model := ResourceF5XCApiDefinitionUploaderModel{
		Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
			Url:       types.StringValue("https://example.com/spec/v1"),
			Name:      types.StringValue("my-api"),
			Version:   types.StringValue("v1"),
			Namespace: types.StringValue("shared"),
		},
		StringValue:    types.StringValue(spec),
		ContentHash:    types.StringValue(hash),
		RetainVersions: types.Int64Value(2),
		Timeouts:       testTimeoutsValue(),
	}

	state := tfsdk.State{Schema: schemaResp.Schema}
	require.False(t, state.Set(context.Background(), &model).HasError())

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	require.False(t, plan.Set(context.Background(), &model).HasError())

	req := resource.UpdateRequest{Plan: plan, State: state}
	var resp resource.UpdateResponse
	resp.State = state

	res.Update(context.Background(), req, &resp)
	require.False(t, resp.Diagnostics.HasError())

	assert.False(t, putExecuted, "PUT must not be executed when content is unchanged (prevents version churn)")
}

func TestResourceF5XCApiDefinitionUploader_Update_ConfigAttributeOnly(t *testing.T) {
	spec := `{"openapi":"3.0.0"}`
	hash := computeSHA256(spec)

	putExecuted := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			putExecuted = true
		}
	}))
	defer srv.Close()

	res := &ResourceF5XCApiDefinitionUploader{client: newMockAPIClient(srv)}

	var schemaResp resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	stateModel := ResourceF5XCApiDefinitionUploaderModel{
		Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
			Url:         types.StringValue("https://example.com/spec/v1"),
			Name:        types.StringValue("my-api"),
			Version:     types.StringValue("v1"),
			Namespace:   types.StringValue("shared"),
			Description: types.StringValue(""),
		},
		StringValue:    types.StringValue(spec),
		ContentHash:    types.StringValue(hash),
		RetainVersions: types.Int64Value(2),
		ForceDelete:    types.BoolValue(false),
		Timeouts:       testTimeoutsValue(),
	}

	planModel := ResourceF5XCApiDefinitionUploaderModel{
		Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
			Name:        types.StringValue("my-api"),
			Namespace:   types.StringValue("shared"),
			Description: types.StringValue(""),
		},
		StringValue:    types.StringValue(spec),
		RetainVersions: types.Int64Value(5),
		ForceDelete:    types.BoolValue(true),
		Timeouts:       testTimeoutsValue(),
	}

	state := tfsdk.State{Schema: schemaResp.Schema}
	require.False(t, state.Set(context.Background(), &stateModel).HasError())

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	require.False(t, plan.Set(context.Background(), &planModel).HasError())

	req := resource.UpdateRequest{Plan: plan, State: state}
	var resp resource.UpdateResponse
	resp.State = state

	res.Update(context.Background(), req, &resp)
	require.False(t, resp.Diagnostics.HasError())

	assert.False(t, putExecuted, "PUT must not be executed when only config attributes change")

	var updatedModel ResourceF5XCApiDefinitionUploaderModel
	require.False(t, resp.State.Get(context.Background(), &updatedModel).HasError())

	assert.True(t, updatedModel.ForceDelete.ValueBool(), "force_delete must be updated in state")
	assert.Equal(t, int64(5), updatedModel.RetainVersions.ValueInt64(), "retain_versions must be updated in state")
	assert.Equal(t, "v1", updatedModel.Metadata.Version.ValueString(), "version must remain unchanged")
	assert.Equal(t, hash, updatedModel.ContentHash.ValueString(), "content_hash must remain unchanged")
}

func TestResourceF5XCApiDefinitionUploader_Create_CustomDescription(t *testing.T) {
	specContent := `{"openapi":"3.0.0"}`
	var receivedDesc string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var rawBody map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&rawBody))
		assert.NotContains(t, rawBody, "metadata", "CreateObjectRequest must not wrap fields inside metadata")
		descVal, _ := rawBody["description"].(string)
		receivedDesc = descVal

		resp := apiDefinitionUploaderResponse{
			Status: "STORED_OBJECT_STATUS_CREATED",
			Metadata: apiDefinitionUploaderMetadataResponse{
				Url:         r.URL.String() + "/v1",
				Name:        "my-api",
				Version:     "v1",
				Namespace:   "shared",
				Description: receivedDesc,
			},
			StringValue: specContent,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	res := &ResourceF5XCApiDefinitionUploader{client: newMockAPIClient(srv)}

	var schemaResp resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	planData := ResourceF5XCApiDefinitionUploaderModel{
		Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
			Name:        types.StringValue("my-api"),
			Namespace:   types.StringValue("shared"),
			Description: types.StringValue("Production API v1"),
		},
		StringValue:    types.StringValue(specContent),
		RetainVersions: types.Int64Value(2),
		ForceDelete:    types.BoolValue(false),
		Timeouts:       testTimeoutsValue(),
	}

	planState := tfsdk.State{Schema: schemaResp.Schema}
	require.False(t, planState.Set(context.Background(), &planData).HasError())

	req := resource.CreateRequest{Plan: tfsdk.Plan(planState)}
	var resp resource.CreateResponse
	resp.State = tfsdk.State{Schema: schemaResp.Schema}

	res.Create(context.Background(), req, &resp)
	require.False(t, resp.Diagnostics.HasError())

	assert.Equal(t, "Production API v1", receivedDesc)

	var createdData ResourceF5XCApiDefinitionUploaderModel
	require.False(t, resp.State.Get(context.Background(), &createdData).HasError())
	assert.Equal(t, "Production API v1", createdData.Metadata.Description.ValueString())
}

func TestResourceF5XCApiDefinitionUploader_RetentionPruning(t *testing.T) {
	var deletedVersions []string
	var mu sync.Mutex

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/stored_objects/swagger") {
			listResp := storedObjectListResponse{
				Items: []storedObjectListItem{
					{
						Name: "test-api",
						Versions: []storedObjectVersionItem{
							{Version: "v3", CreationTimestamp: "2026-09-24T14:00:00Z", Url: "/swagger/test-api/v3"},
							{Version: "v2", CreationTimestamp: "2026-09-24T13:00:00Z", Url: "/swagger/test-api/v2"},
							{Version: "v1", CreationTimestamp: "2026-09-24T12:00:00Z", Url: "/swagger/test-api/v1"},
							{Version: "v0", CreationTimestamp: "2026-09-24T11:00:00Z", Url: "/swagger/test-api/v0"},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(listResp)
			return
		}

		if r.Method == http.MethodDelete {
			parts := strings.Split(r.URL.Path, "/")
			deletedVersions = append(deletedVersions, parts[len(parts)-1])
			w.WriteHeader(http.StatusOK)
			return
		}
	}))
	defer srv.Close()

	res := &ResourceF5XCApiDefinitionUploader{client: newMockAPIClient(srv)}

	// Retain 2 versions (v3 and v2 should be kept, v1 and v0 pruned)
	res.pruneHistoricalVersions(context.Background(), "shared", "test-api", 2, "v3")

	mu.Lock()
	defer mu.Unlock()

	assert.Len(t, deletedVersions, 2, "Should prune exactly 2 historical versions")
	assert.Contains(t, deletedVersions, "v1")
	assert.Contains(t, deletedVersions, "v0")
	assert.NotContains(t, deletedVersions, "v2")
	assert.NotContains(t, deletedVersions, "v3")
}

func TestResourceF5XCApiDefinitionUploader_Delete(t *testing.T) {
	deleteCalled := false
	deletedURL := ""

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deleteCalled = true
			deletedURL = r.URL.Path
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	res := &ResourceF5XCApiDefinitionUploader{client: newMockAPIClient(srv)}

	var schemaResp resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	stateData := ResourceF5XCApiDefinitionUploaderModel{
		Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
			Url:       types.StringValue(srv.URL + "/object_store/namespaces/shared/stored_objects/swagger/my-api/v1"),
			Name:      types.StringValue("my-api"),
			Namespace: types.StringValue("shared"),
		},
		ForceDelete: types.BoolValue(false),
		Timeouts:    testTimeoutsValue(),
	}

	state := tfsdk.State{Schema: schemaResp.Schema}
	require.False(t, state.Set(context.Background(), &stateData).HasError())

	req := resource.DeleteRequest{State: state}
	var resp resource.DeleteResponse

	res.Delete(context.Background(), req, &resp)
	require.False(t, resp.Diagnostics.HasError())

	assert.True(t, deleteCalled)
	assert.Equal(t, "/object_store/namespaces/shared/stored_objects/swagger/my-api/v1", deletedURL)
}

func TestResourceF5XCApiDefinitionUploader_ModifyPlan_ChangedContent(t *testing.T) {
	res := &ResourceF5XCApiDefinitionUploader{}

	var schemaResp resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	stateData := ResourceF5XCApiDefinitionUploaderModel{
		Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
			Url:       types.StringValue("https://example.com/v1"),
			Name:      types.StringValue("my-api"),
			Version:   types.StringValue("v1"),
			Namespace: types.StringValue("shared"),
		},
		StringValue: types.StringValue(`{"openapi":"3.0.0","version":"1"}`),
		ContentHash: types.StringValue("oldhash"),
		Timeouts:    testTimeoutsValue(),
	}

	planData := ResourceF5XCApiDefinitionUploaderModel{
		Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
			Name:      types.StringValue("my-api"),
			Namespace: types.StringValue("shared"),
		},
		StringValue: types.StringValue(`{"openapi":"3.0.0","version":"2"}`),
		Timeouts:    testTimeoutsValue(),
	}

	state := tfsdk.State{Schema: schemaResp.Schema}
	require.False(t, state.Set(context.Background(), &stateData).HasError())

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	require.False(t, plan.Set(context.Background(), &planData).HasError())

	req := resource.ModifyPlanRequest{
		State: state,
		Plan:  plan,
	}
	resp := resource.ModifyPlanResponse{
		Plan: plan,
	}

	res.ModifyPlan(context.Background(), req, &resp)
	require.False(t, resp.Diagnostics.HasError())

	var planned ResourceF5XCApiDefinitionUploaderModel
	require.False(t, resp.Plan.Get(context.Background(), &planned).HasError())

	assert.True(t, planned.Metadata.Url.IsUnknown(), "URL must be unknown when string_value changes")
	assert.True(t, planned.Metadata.Version.IsUnknown(), "Version must be unknown when string_value changes")
	assert.True(t, planned.ContentHash.IsUnknown(), "ContentHash must be unknown when string_value changes")
}

func TestResourceF5XCApiDefinitionUploader_RetentionPruning_LivePayloadShape(t *testing.T) {
	var mu sync.Mutex
	var listQueries []string
	var deletedPaths []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if r.Method == http.MethodGet && r.URL.Path == "/api/object_store/namespaces/shared/stored_objects/swagger" {
			listQueries = append(listQueries, r.URL.RawQuery)
			w.Header().Set("Content-Type", "application/json")
			// Simulate F5 XC live object_store list payload:
			// - item.name is namespace-prefixed ("<namespace>/<name>")
			// - timestamps use snake_case ("creation_timestamp")
			// - version urls include the "/api/..." prefix
			payload := map[string]any{
				"items": []map[string]any{
					{
						"name": "shared/xc35500-tc07-ret",
						"versions": []map[string]any{
							{
								"version":            "v1-26-11-18",
								"url":                "/api/object_store/namespaces/shared/stored_objects/swagger/xc35500-tc07-ret/v1-26-11-18",
								"creation_timestamp": "2026-11-18T10:00:01Z",
							},
							{
								"version":            "v3-26-11-18",
								"url":                "/api/object_store/namespaces/shared/stored_objects/swagger/xc35500-tc07-ret/v3-26-11-18",
								"creation_timestamp": "2026-11-18T10:00:03Z",
							},
							{
								"version":            "v2-26-11-18",
								"url":                "/api/object_store/namespaces/shared/stored_objects/swagger/xc35500-tc07-ret/v2-26-11-18",
								"creation_timestamp": "2026-11-18T10:00:02Z",
							},
							{
								"version":            "v4-26-11-18",
								"url":                "/api/object_store/namespaces/shared/stored_objects/swagger/xc35500-tc07-ret/v4-26-11-18",
								"creation_timestamp": "2026-11-18T10:00:04Z",
								"latest_version":     true,
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(payload)
			return
		}

		if r.Method == http.MethodDelete {
			deletedPaths = append(deletedPaths, r.URL.Path)
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	mockClient := newMockAPIClient(srv)
	mockClient.RestClient.URL = srv.URL + "/api"
	res := &ResourceF5XCApiDefinitionUploader{client: mockClient}

	res.pruneHistoricalVersions(context.Background(), "shared", "xc35500-tc07-ret", 2, "v4-26-11-18")

	mu.Lock()
	defer mu.Unlock()

	require.Len(t, listQueries, 1, "ListObjects endpoint must be queried once without ?name= filter")
	assert.NotContains(t, listQueries[0], "name=", "ListObjects query must not send broken ?name= filter")

	expectedDeleted := []string{
		"/api/object_store/namespaces/shared/stored_objects/swagger/xc35500-tc07-ret/v2-26-11-18",
		"/api/object_store/namespaces/shared/stored_objects/swagger/xc35500-tc07-ret/v1-26-11-18",
	}
	assert.ElementsMatch(t, expectedDeleted, deletedPaths, "Oldest 2 versions must be deleted without double /api/api prefix")
}

func TestResourceF5XCApiDefinitionUploader_RetentionPruning_EmptyTimestampsVersionSort(t *testing.T) {
	var mu sync.Mutex
	var deletedVersions []string

	versions := make([]storedObjectVersionItem, 0, 11)
	for i := 1; i <= 11; i++ {
		ver := fmt.Sprintf("v%d-26-11-18", i)
		versions = append(versions, storedObjectVersionItem{
			Version: ver,
			Url:     fmt.Sprintf("/api/object_store/namespaces/shared/stored_objects/swagger/multi-ver/%s", ver),
		})
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/stored_objects/swagger") {
			listResp := storedObjectListResponse{
				Items: []storedObjectListItem{
					{
						Name:     "shared/multi-ver",
						Versions: versions,
					},
				},
			}
			_ = json.NewEncoder(w).Encode(listResp)
			return
		}

		if r.Method == http.MethodDelete {
			parts := strings.Split(r.URL.Path, "/")
			deletedVersions = append(deletedVersions, parts[len(parts)-1])
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	mockClient := newMockAPIClient(srv)
	mockClient.RestClient.URL = srv.URL + "/api"
	res := &ResourceF5XCApiDefinitionUploader{client: mockClient}

	res.pruneHistoricalVersions(context.Background(), "shared", "multi-ver", 2, "v11-26-11-18")

	mu.Lock()
	defer mu.Unlock()

	assert.Len(t, deletedVersions, 9, "All 9 older versions (v1..v9) must be pruned")
	assert.NotContains(t, deletedVersions, "v11-26-11-18", "Newest version v11 must be retained")
	assert.NotContains(t, deletedVersions, "v10-26-11-18", "Second-newest version v10 must be retained")
	for i := 1; i <= 9; i++ {
		assert.Contains(t, deletedVersions, fmt.Sprintf("v%d-26-11-18", i))
	}
}

func TestResourceF5XCApiDefinitionUploader_RetentionPruning_MultiTenantNamespaces(t *testing.T) {
	var mu sync.Mutex
	var deletedPaths []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/stored_objects/swagger") {
			listResp := storedObjectListResponse{
				Items: []storedObjectListItem{
					{
						Name: "shared/my-api",
						Versions: []storedObjectVersionItem{
							{Version: "v1-26-11-18", CreationTimestamp: "2026-11-18T10:00:01Z", Url: "/api/object_store/namespaces/shared/stored_objects/swagger/my-api/v1-26-11-18"},
							{Version: "v2-26-11-18", CreationTimestamp: "2026-11-18T10:00:02Z", Url: "/api/object_store/namespaces/shared/stored_objects/swagger/my-api/v2-26-11-18"},
							{Version: "v3-26-11-18", CreationTimestamp: "2026-11-18T10:00:03Z", Url: "/api/object_store/namespaces/shared/stored_objects/swagger/my-api/v3-26-11-18"},
						},
					},
					{
						Name: "shared/my-api-other",
						Versions: []storedObjectVersionItem{
							{Version: "v1-26-11-18", CreationTimestamp: "2026-11-18T10:00:01Z", Url: "/api/object_store/namespaces/shared/stored_objects/swagger/my-api-other/v1-26-11-18"},
							{Version: "v2-26-11-18", CreationTimestamp: "2026-11-18T10:00:02Z", Url: "/api/object_store/namespaces/shared/stored_objects/swagger/my-api-other/v2-26-11-18"},
						},
					},
					{
						Name: "other-ns/my-api",
						Versions: []storedObjectVersionItem{
							{Version: "v1-26-11-18", CreationTimestamp: "2026-11-18T10:00:01Z", Url: "/api/object_store/namespaces/other-ns/stored_objects/swagger/my-api/v1-26-11-18"},
							{Version: "v2-26-11-18", CreationTimestamp: "2026-11-18T10:00:02Z", Url: "/api/object_store/namespaces/other-ns/stored_objects/swagger/my-api/v2-26-11-18"},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(listResp)
			return
		}

		if r.Method == http.MethodDelete {
			deletedPaths = append(deletedPaths, r.URL.Path)
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	mockClient := newMockAPIClient(srv)
	mockClient.RestClient.URL = srv.URL + "/api"
	res := &ResourceF5XCApiDefinitionUploader{client: mockClient}

	res.pruneHistoricalVersions(context.Background(), "shared", "my-api", 1, "v3-26-11-18")

	mu.Lock()
	defer mu.Unlock()

	expectedDeleted := []string{
		"/api/object_store/namespaces/shared/stored_objects/swagger/my-api/v2-26-11-18",
		"/api/object_store/namespaces/shared/stored_objects/swagger/my-api/v1-26-11-18",
	}
	assert.ElementsMatch(t, expectedDeleted, deletedPaths, "Only shared/my-api historical versions must be deleted")
}

func TestResourceF5XCApiDefinitionUploader_RetentionPruning_DeleteFailureWarning(t *testing.T) {
	oldSpec := `{"openapi":"3.0.0","version":"1.0"}`
	newSpec := `{"openapi":"3.0.0","version":"2.0"}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			resp := apiDefinitionUploaderResponse{
				Status: "STORED_OBJECT_STATUS_UPDATED",
				Metadata: apiDefinitionUploaderMetadataResponse{
					Url:               "/api/object_store/namespaces/shared/stored_objects/swagger/my-api/v3-26-11-18",
					Name:              "my-api",
					Version:           "v3-26-11-18",
					Namespace:         "shared",
					CreationTimestamp: "2026-11-18T12:00:00Z",
				},
				StringValue: newSpec,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/stored_objects/swagger") {
			listResp := storedObjectListResponse{
				Items: []storedObjectListItem{
					{
						Name: "shared/my-api",
						Versions: []storedObjectVersionItem{
							{Version: "v1-26-11-18", CreationTimestamp: "2026-11-18T10:00:00Z", Url: "/api/object_store/namespaces/shared/stored_objects/swagger/my-api/v1-26-11-18"},
							{Version: "v2-26-11-18", CreationTimestamp: "2026-11-18T11:00:00Z", Url: "/api/object_store/namespaces/shared/stored_objects/swagger/my-api/v2-26-11-18"},
							{Version: "v3-26-11-18", CreationTimestamp: "2026-11-18T12:00:00Z", Url: "/api/object_store/namespaces/shared/stored_objects/swagger/my-api/v3-26-11-18"},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(listResp)
			return
		}

		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"delete failed"}`))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	mockClient := newMockAPIClient(srv)
	mockClient.RestClient.URL = srv.URL + "/api"
	res := &ResourceF5XCApiDefinitionUploader{client: mockClient}

	var schemaResp resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	stateData := ResourceF5XCApiDefinitionUploaderModel{
		Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
			Url:       types.StringValue(srv.URL + "/api/object_store/namespaces/shared/stored_objects/swagger/my-api/v2-26-11-18"),
			Name:      types.StringValue("my-api"),
			Version:   types.StringValue("v2-26-11-18"),
			Namespace: types.StringValue("shared"),
		},
		StringValue:    types.StringValue(oldSpec),
		ContentHash:    types.StringValue(computeSHA256(oldSpec)),
		RetainVersions: types.Int64Value(2),
		Timeouts:       testTimeoutsValue(),
	}

	planData := ResourceF5XCApiDefinitionUploaderModel{
		Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
			Name:      types.StringValue("my-api"),
			Namespace: types.StringValue("shared"),
		},
		StringValue:    types.StringValue(newSpec),
		RetainVersions: types.Int64Value(2),
		Timeouts:       testTimeoutsValue(),
	}

	state := tfsdk.State{Schema: schemaResp.Schema}
	require.False(t, state.Set(context.Background(), &stateData).HasError())

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	require.False(t, plan.Set(context.Background(), &planData).HasError())

	req := resource.UpdateRequest{Plan: plan, State: state}
	var resp resource.UpdateResponse
	resp.State = state

	res.Update(context.Background(), req, &resp)

	assert.False(t, resp.Diagnostics.HasError(), "Pruning DELETE failure must be non-fatal")
	assert.Positive(t, resp.Diagnostics.WarningsCount(), "Pruning DELETE failure must surface a warning diagnostic")

	var updatedData ResourceF5XCApiDefinitionUploaderModel
	require.False(t, resp.State.Get(context.Background(), &updatedData).HasError())
	assert.Equal(t, "v3-26-11-18", updatedData.Metadata.Version.ValueString())
}

func TestResourceF5XCApiDefinitionUploader_ResolveAbsoluteURL(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		targetURL string
		expected  string
	}{
		{
			name:      "api-base-with-api-prefixed-target",
			baseURL:   "https://tenant.console.ves.volterra.io/api",
			targetURL: "/api/object_store/namespaces/shared/stored_objects/swagger/foo/v1",
			expected:  "https://tenant.console.ves.volterra.io/api/object_store/namespaces/shared/stored_objects/swagger/foo/v1",
		},
		{
			name:      "api-base-trailing-slash-with-api-prefixed-target",
			baseURL:   "https://tenant.console.ves.volterra.io/api/",
			targetURL: "/api/object_store/namespaces/shared/stored_objects/swagger/foo/v1",
			expected:  "https://tenant.console.ves.volterra.io/api/object_store/namespaces/shared/stored_objects/swagger/foo/v1",
		},
		{
			name:      "api-base-with-relative-target",
			baseURL:   "https://tenant.console.ves.volterra.io/api",
			targetURL: "/object_store/namespaces/shared/stored_objects/swagger/foo/v1",
			expected:  "https://tenant.console.ves.volterra.io/api/object_store/namespaces/shared/stored_objects/swagger/foo/v1",
		},
		{
			name:      "already-absolute-https",
			baseURL:   "https://tenant.console.ves.volterra.io/api",
			targetURL: "https://other.example.com/api/foo",
			expected:  "https://other.example.com/api/foo",
		},
		{
			name:      "root-base-with-relative-target",
			baseURL:   "http://127.0.0.1:12345",
			targetURL: "/swagger/test-api/v1",
			expected:  "http://127.0.0.1:12345/swagger/test-api/v1",
		},
		{
			name:      "api-base-matching-api-target",
			baseURL:   "https://tenant.console.ves.volterra.io/api",
			targetURL: "/api",
			expected:  "https://tenant.console.ves.volterra.io/api",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, resolveAbsoluteURL(tc.baseURL, tc.targetURL))
		})
	}
}

func TestResourceF5XCApiDefinitionUploader_ModifyPlan_ChangedDescription(t *testing.T) {
	res := &ResourceF5XCApiDefinitionUploader{}

	var schemaResp resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	spec := `{"openapi":"3.0.0","version":"1"}`
	hash := computeSHA256(spec)

	stateData := ResourceF5XCApiDefinitionUploaderModel{
		Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
			Url:               types.StringValue("https://example.com/v1"),
			Name:              types.StringValue("my-api"),
			Version:           types.StringValue("v1"),
			Namespace:         types.StringValue("shared"),
			Description:       types.StringValue("old description"),
			CreationTimestamp: types.StringValue("2026-09-24T12:00:00Z"),
		},
		Status:      types.StringValue("STORED_OBJECT_STATUS_CREATED"),
		StringValue: types.StringValue(spec),
		ContentHash: types.StringValue(hash),
		Timeouts:    testTimeoutsValue(),
	}

	planData := ResourceF5XCApiDefinitionUploaderModel{
		Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
			Name:        types.StringValue("my-api"),
			Namespace:   types.StringValue("shared"),
			Description: types.StringValue("new description"),
		},
		StringValue: types.StringValue(spec),
		ContentHash: types.StringValue(hash),
		Timeouts:    testTimeoutsValue(),
	}

	state := tfsdk.State{Schema: schemaResp.Schema}
	require.False(t, state.Set(context.Background(), &stateData).HasError())

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	require.False(t, plan.Set(context.Background(), &planData).HasError())

	req := resource.ModifyPlanRequest{
		State: state,
		Plan:  plan,
	}
	resp := resource.ModifyPlanResponse{
		Plan: plan,
	}

	res.ModifyPlan(context.Background(), req, &resp)
	require.False(t, resp.Diagnostics.HasError())

	var planned ResourceF5XCApiDefinitionUploaderModel
	require.False(t, resp.Plan.Get(context.Background(), &planned).HasError())

	assert.True(t, planned.Metadata.Url.IsUnknown(), "URL must be unknown when metadata.description changes")
	assert.True(t, planned.Metadata.Version.IsUnknown(), "Version must be unknown when metadata.description changes")
	assert.True(t, planned.Metadata.CreationTimestamp.IsUnknown(), "CreationTimestamp must be unknown when metadata.description changes")
	assert.True(t, planned.Status.IsUnknown(), "Status must be unknown when metadata.description changes")
	assert.False(t, planned.ContentHash.IsUnknown(), "ContentHash must remain known when string_value is unchanged")
	assert.Equal(t, hash, planned.ContentHash.ValueString())
}

// buildUploaderConfig constructs a tfsdk.Config populated with model using schema.
//
// Why: Simplifies building Terraform request.Config inputs for ValidateConfig, ModifyPlan, Create, and Update unit tests.
//
// How: Encodes model into a tfsdk.State backed by schema and wraps its Raw value in tfsdk.Config.
//
// Parameters:
//   - t: test handle for assertions.
//   - schemaDef: resource schema definition.
//   - model: populated resource model.
//
// Returns:
//   - tfsdk.Config: constructed Terraform configuration object.
//
// Example:
//
//	cfg := buildUploaderConfig(t, schemaResp.Schema, configModel)
func buildUploaderConfig(t *testing.T, schemaDef rschema.Schema, model ResourceF5XCApiDefinitionUploaderModel) tfsdk.Config {
	t.Helper()
	holder := tfsdk.State{Schema: schemaDef}
	require.False(t, holder.Set(context.Background(), &model).HasError())
	return tfsdk.Config{Schema: schemaDef, Raw: holder.Raw}
}

// TestResourceF5XCApiDefinitionUploader_ValidateConfig verifies mutual exclusivity, Base64 decoding,
// and 5 MiB payload limits across write-only specification attributes.
//
// Why: Ensures invalid attribute combinations and oversized or malformed Base64 payloads fail early
// during config validation before any API call occurs.
//
// How: Invokes ValidateConfig across valid, conflicting, missing, invalid Base64, oversized, and
// unknown configuration scenarios and asserts expected diagnostics.
//
// Parameters:
//   - t: test handle.
//
// Returns:
//   - None
//
// Example:
//
//	go test -run TestResourceF5XCApiDefinitionUploader_ValidateConfig
func TestResourceF5XCApiDefinitionUploader_ValidateConfig(t *testing.T) {
	res := &ResourceF5XCApiDefinitionUploader{}
	var schemaResp resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	baseMetadata := ResourceF5XCApiDefinitionUploaderMetadataModel{
		Name:      types.StringValue("my-api"),
		Namespace: types.StringValue("shared"),
	}

	validSpec := `{"openapi":"3.0.0"}`
	validB64 := base64.StdEncoding.EncodeToString([]byte(validSpec))
	oversizedB64 := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte("a"), maxStoredObjectSpecBytes+1))

	tests := []struct {
		name            string
		model           ResourceF5XCApiDefinitionUploaderModel
		expectError     bool
		expectedSummary string
	}{
		{
			name: "missing-all-content",
			model: ResourceF5XCApiDefinitionUploaderModel{
				Metadata: baseMetadata,
				Timeouts: testTimeoutsValue(),
			},
			expectError:     true,
			expectedSummary: "Missing API Specification Content",
		},
		{
			name: "conflicting-string-value-and-bytes-value",
			model: ResourceF5XCApiDefinitionUploaderModel{
				Metadata:    baseMetadata,
				StringValue: types.StringValue(validSpec),
				BytesValue:  types.StringValue(validB64),
				Timeouts:    testTimeoutsValue(),
			},
			expectError:     true,
			expectedSummary: "Conflicting API Specification Content",
		},
		{
			name: "valid-string-value",
			model: ResourceF5XCApiDefinitionUploaderModel{
				Metadata:    baseMetadata,
				StringValue: types.StringValue(validSpec),
				Timeouts:    testTimeoutsValue(),
			},
			expectError: false,
		},
		{
			name: "valid-bytes-value",
			model: ResourceF5XCApiDefinitionUploaderModel{
				Metadata:   baseMetadata,
				BytesValue: types.StringValue(validB64),
				Timeouts:   testTimeoutsValue(),
			},
			expectError: false,
		},
		{
			name: "invalid-base64-bytes-value",
			model: ResourceF5XCApiDefinitionUploaderModel{
				Metadata:   baseMetadata,
				BytesValue: types.StringValue("not!valid!base64"),
				Timeouts:   testTimeoutsValue(),
			},
			expectError:     true,
			expectedSummary: "Invalid Base64 Value",
		},
		{
			name: "oversized-bytes-value",
			model: ResourceF5XCApiDefinitionUploaderModel{
				Metadata:   baseMetadata,
				BytesValue: types.StringValue(oversizedB64),
				Timeouts:   testTimeoutsValue(),
			},
			expectError:     true,
			expectedSummary: "Specification Exceeds Maximum Size",
		},
		{
			name: "oversized-string-value",
			model: ResourceF5XCApiDefinitionUploaderModel{
				Metadata:    baseMetadata,
				StringValue: types.StringValue(strings.Repeat("a", maxStoredObjectSpecBytes+1)),
				Timeouts:    testTimeoutsValue(),
			},
			expectError:     true,
			expectedSummary: "Specification Exceeds Maximum Size",
		},
		{
			name: "unknown-bytes-value",
			model: ResourceF5XCApiDefinitionUploaderModel{
				Metadata:   baseMetadata,
				BytesValue: types.StringUnknown(),
				Timeouts:   testTimeoutsValue(),
			},
			expectError: false,
		},
		{
			name: "unknown-both-string-and-bytes",
			model: ResourceF5XCApiDefinitionUploaderModel{
				Metadata:    baseMetadata,
				StringValue: types.StringUnknown(),
				BytesValue:  types.StringUnknown(),
				Timeouts:    testTimeoutsValue(),
			},
			expectError: false,
		},
		{
			name: "known-string-unknown-bytes",
			model: ResourceF5XCApiDefinitionUploaderModel{
				Metadata:    baseMetadata,
				StringValue: types.StringValue(validSpec),
				BytesValue:  types.StringUnknown(),
				Timeouts:    testTimeoutsValue(),
			},
			expectError: false,
		},
		{
			name: "known-invalid-bytes-unknown-string",
			model: ResourceF5XCApiDefinitionUploaderModel{
				Metadata:    baseMetadata,
				StringValue: types.StringUnknown(),
				BytesValue:  types.StringValue("not!valid!base64"),
				Timeouts:    testTimeoutsValue(),
			},
			expectError:     true,
			expectedSummary: "Invalid Base64 Value",
		},
		{
			name: "known-oversized-string-unknown-bytes",
			model: ResourceF5XCApiDefinitionUploaderModel{
				Metadata:    baseMetadata,
				StringValue: types.StringValue(strings.Repeat("a", maxStoredObjectSpecBytes+1)),
				BytesValue:  types.StringUnknown(),
				Timeouts:    testTimeoutsValue(),
			},
			expectError:     true,
			expectedSummary: "Specification Exceeds Maximum Size",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := buildUploaderConfig(t, schemaResp.Schema, tc.model)
			req := resource.ValidateConfigRequest{Config: cfg}
			var resp resource.ValidateConfigResponse
			res.ValidateConfig(context.Background(), req, &resp)

			if tc.expectError {
				require.True(t, resp.Diagnostics.HasError())
				assert.Equal(t, tc.expectedSummary, resp.Diagnostics[0].Summary())
			} else {
				assert.False(t, resp.Diagnostics.HasError())
			}
		})
	}
}

// TestResourceF5XCApiDefinitionUploader_Create_WriteOnlyString verifies Create behavior for write-only string_value
// and confirms the specification is never persisted into Terraform state.
//
// Why: Guarantees state remains compact and excludes multi-megabyte OpenAPI documents.
//
// How: Passes string_value via req.Config while keeping req.Plan write-only fields null, intercepts the HTTP PUT
// payload, and checks both wire serialization and final state attributes.
//
// Parameters:
//   - t: test handle.
//
// Returns:
//   - None
//
// Example:
//
//	go test -run TestResourceF5XCApiDefinitionUploader_Create_WriteOnlyString
func TestResourceF5XCApiDefinitionUploader_Create_WriteOnlyString(t *testing.T) {
	specContent := `{"openapi":"3.0.0","info":{"title":"WO API","version":"1.0.0"}}`
	expectedHash := computeSHA256(specContent)
	var receivedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&receivedBody))

		resp := apiDefinitionUploaderResponse{
			Status: "STORED_OBJECT_STATUS_CREATED",
			Metadata: apiDefinitionUploaderMetadataResponse{
				Url:               r.URL.String() + "/v1",
				Name:              "wo-api",
				Version:           "v1",
				Namespace:         "shared",
				Description:       "",
				CreationTimestamp: "2026-10-01T10:00:00Z",
			},
			StringValue: specContent,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	res := &ResourceF5XCApiDefinitionUploader{client: newMockAPIClient(srv)}
	var schemaResp resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	configModel := ResourceF5XCApiDefinitionUploaderModel{
		Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
			Name:        types.StringValue("wo-api"),
			Namespace:   types.StringValue("shared"),
			Description: types.StringValue(""),
		},
		StringValue:    types.StringValue(specContent),
		RetainVersions: types.Int64Value(2),
		ForceDelete:    types.BoolValue(false),
		Timeouts:       testTimeoutsValue(),
	}

	planModel := configModel
	planModel.StringValue = types.StringNull()
	planModel.ContentHash = types.StringUnknown()

	planState := tfsdk.State{Schema: schemaResp.Schema}
	require.False(t, planState.Set(context.Background(), &planModel).HasError())

	req := resource.CreateRequest{
		Config: buildUploaderConfig(t, schemaResp.Schema, configModel),
		Plan:   tfsdk.Plan(planState),
	}
	var resp resource.CreateResponse
	resp.State = tfsdk.State{Schema: schemaResp.Schema}

	res.Create(context.Background(), req, &resp)
	require.False(t, resp.Diagnostics.HasError())

	assert.Equal(t, specContent, receivedBody["string_value"])
	assert.NotContains(t, receivedBody, "bytes_value")

	var created ResourceF5XCApiDefinitionUploaderModel
	require.False(t, resp.State.Get(context.Background(), &created).HasError())

	assert.True(t, created.StringValue.IsNull(), "string_value must remain null in state")
	assert.True(t, created.BytesValue.IsNull(), "bytes_value must remain null in state")
	assert.Equal(t, expectedHash, created.ContentHash.ValueString())
	assert.Equal(t, "v1", created.Metadata.Version.ValueString())
}

// TestResourceF5XCApiDefinitionUploader_Create_WriteOnlyBytes verifies Create behavior for write-only bytes_value
// and explicit content_hash preservation.
//
// Why: Ensures Base64 write-only specifications are serialized as bytes_value, hashed consistently on raw bytes,
// and excluded from state while respecting user-provided content_hash values.
//
// How: Exercises both auto-computed SHA-256 and user-supplied content_hash via bytes_value.
//
// Parameters:
//   - t: test handle.
//
// Returns:
//   - None
//
// Example:
//
//	go test -run TestResourceF5XCApiDefinitionUploader_Create_WriteOnlyBytes
func TestResourceF5XCApiDefinitionUploader_Create_WriteOnlyBytes(t *testing.T) {
	rawSpec := []byte(`{"openapi":"3.0.0","info":{"title":"Bytes API","version":"1.0.0"}}`)
	encodedSpec := base64.StdEncoding.EncodeToString(rawSpec)
	expectedRawHash := computeSHA256Bytes(rawSpec)

	t.Run("auto-computed-hash", func(t *testing.T) {
		var receivedBody map[string]any

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodPut, r.Method)
			require.NoError(t, json.NewDecoder(r.Body).Decode(&receivedBody))

			resp := apiDefinitionUploaderResponse{
				Status: "STORED_OBJECT_STATUS_CREATED",
				Metadata: apiDefinitionUploaderMetadataResponse{
					Url:               r.URL.String() + "/v1",
					Name:              "bytes-api",
					Version:           "v1",
					Namespace:         "shared",
					Description:       "",
					CreationTimestamp: "2026-10-01T10:00:00Z",
				},
				BytesValue: encodedSpec,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		res := &ResourceF5XCApiDefinitionUploader{client: newMockAPIClient(srv)}
		var schemaResp resource.SchemaResponse
		res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

		configModel := ResourceF5XCApiDefinitionUploaderModel{
			Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
				Name:        types.StringValue("bytes-api"),
				Namespace:   types.StringValue("shared"),
				Description: types.StringValue(""),
			},
			BytesValue:     types.StringValue(encodedSpec),
			RetainVersions: types.Int64Value(2),
			ForceDelete:    types.BoolValue(false),
			Timeouts:       testTimeoutsValue(),
		}

		planModel := configModel
		planModel.BytesValue = types.StringNull()
		planModel.ContentHash = types.StringUnknown()

		planState := tfsdk.State{Schema: schemaResp.Schema}
		require.False(t, planState.Set(context.Background(), &planModel).HasError())

		req := resource.CreateRequest{
			Config: buildUploaderConfig(t, schemaResp.Schema, configModel),
			Plan:   tfsdk.Plan(planState),
		}
		var resp resource.CreateResponse
		resp.State = tfsdk.State{Schema: schemaResp.Schema}

		res.Create(context.Background(), req, &resp)
		require.False(t, resp.Diagnostics.HasError())

		assert.Equal(t, encodedSpec, receivedBody["bytes_value"])
		assert.NotContains(t, receivedBody, "string_value")

		var created ResourceF5XCApiDefinitionUploaderModel
		require.False(t, resp.State.Get(context.Background(), &created).HasError())
		assert.Equal(t, expectedRawHash, created.ContentHash.ValueString())
		assert.True(t, created.StringValue.IsNull())
		assert.True(t, created.BytesValue.IsNull())
	})

	t.Run("explicit-content-hash-preserved", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := apiDefinitionUploaderResponse{
				Status: "STORED_OBJECT_STATUS_CREATED",
				Metadata: apiDefinitionUploaderMetadataResponse{
					Url:               r.URL.String() + "/v1",
					Name:              "bytes-api",
					Version:           "v1",
					Namespace:         "shared",
					Description:       "",
					CreationTimestamp: "2026-10-01T10:00:00Z",
				},
				BytesValue: encodedSpec,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		res := &ResourceF5XCApiDefinitionUploader{client: newMockAPIClient(srv)}
		var schemaResp resource.SchemaResponse
		res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

		customHash := "custom-release-v2.1.0"
		configModel := ResourceF5XCApiDefinitionUploaderModel{
			Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
				Name:        types.StringValue("bytes-api"),
				Namespace:   types.StringValue("shared"),
				Description: types.StringValue(""),
			},
			BytesValue:     types.StringValue(encodedSpec),
			ContentHash:    types.StringValue(customHash),
			RetainVersions: types.Int64Value(2),
			ForceDelete:    types.BoolValue(false),
			Timeouts:       testTimeoutsValue(),
		}

		planModel := configModel
		planModel.BytesValue = types.StringNull()

		planState := tfsdk.State{Schema: schemaResp.Schema}
		require.False(t, planState.Set(context.Background(), &planModel).HasError())

		req := resource.CreateRequest{
			Config: buildUploaderConfig(t, schemaResp.Schema, configModel),
			Plan:   tfsdk.Plan(planState),
		}
		var resp resource.CreateResponse
		resp.State = tfsdk.State{Schema: schemaResp.Schema}

		res.Create(context.Background(), req, &resp)
		require.False(t, resp.Diagnostics.HasError())

		var created ResourceF5XCApiDefinitionUploaderModel
		require.False(t, resp.State.Get(context.Background(), &created).HasError())
		assert.Equal(t, customHash, created.ContentHash.ValueString())
		assert.True(t, created.StringValue.IsNull())
		assert.True(t, created.BytesValue.IsNull())
	})
}

// TestResourceF5XCApiDefinitionUploader_Read_WriteOnly verifies Read refreshes metadata while keeping
// specification content null in write-only state snapshots.
//
// Why: Prevents Read from leaking server-returned OpenAPI specs back into Terraform state.
//
// How: Runs Read with prior state having null spec fields and checks both preserved custom hashes and
// import backfill when ContentHash starts null.
//
// Parameters:
//   - t: test handle.
//
// Returns:
//   - None
//
// Example:
//
//	go test -run TestResourceF5XCApiDefinitionUploader_Read_WriteOnly
func TestResourceF5XCApiDefinitionUploader_Read_WriteOnly(t *testing.T) {
	specContent := `{"openapi":"3.0.0","info":{"title":"Large API","version":"1.0.0"}}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		resp := apiDefinitionUploaderResponse{
			Metadata: apiDefinitionUploaderMetadataResponse{
				Url:               r.URL.String(),
				Name:              "wo-api",
				Version:           "v1",
				Namespace:         "shared",
				Description:       "updated desc",
				CreationTimestamp: "2026-10-01T10:00:00Z",
			},
			StringValue: specContent,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	res := &ResourceF5XCApiDefinitionUploader{client: newMockAPIClient(srv)}
	var schemaResp resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	t.Run("preserves-null-spec-and-existing-hash", func(t *testing.T) {
		stateData := ResourceF5XCApiDefinitionUploaderModel{
			Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
				Url:       types.StringValue(srv.URL + "/object_store/namespaces/shared/stored_objects/swagger/wo-api/v1"),
				Name:      types.StringValue("wo-api"),
				Namespace: types.StringValue("shared"),
			},
			ContentHash: types.StringValue("custom-digest"),
			Timeouts:    testTimeoutsValue(),
		}

		state := tfsdk.State{Schema: schemaResp.Schema}
		require.False(t, state.Set(context.Background(), &stateData).HasError())

		req := resource.ReadRequest{State: state}
		resp := resource.ReadResponse{State: state}
		res.Read(context.Background(), req, &resp)
		require.False(t, resp.Diagnostics.HasError())

		var refreshed ResourceF5XCApiDefinitionUploaderModel
		require.False(t, resp.State.Get(context.Background(), &refreshed).HasError())

		assert.True(t, refreshed.StringValue.IsNull(), "Read must not write string_value into state")
		assert.True(t, refreshed.BytesValue.IsNull(), "Read must not write bytes_value into state")
		assert.Equal(t, "custom-digest", refreshed.ContentHash.ValueString())
		assert.Equal(t, "updated desc", refreshed.Metadata.Description.ValueString())
	})

	t.Run("import-backfills-content-hash-without-persisting-spec", func(t *testing.T) {
		stateData := ResourceF5XCApiDefinitionUploaderModel{
			Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
				Url:       types.StringValue(srv.URL + "/object_store/namespaces/shared/stored_objects/swagger/wo-api/v1"),
				Name:      types.StringValue("wo-api"),
				Namespace: types.StringValue("shared"),
			},
			Timeouts: testTimeoutsValue(),
		}

		state := tfsdk.State{Schema: schemaResp.Schema}
		require.False(t, state.Set(context.Background(), &stateData).HasError())

		req := resource.ReadRequest{State: state}
		resp := resource.ReadResponse{State: state}
		res.Read(context.Background(), req, &resp)
		require.False(t, resp.Diagnostics.HasError())

		var refreshed ResourceF5XCApiDefinitionUploaderModel
		require.False(t, resp.State.Get(context.Background(), &refreshed).HasError())

		assert.True(t, refreshed.StringValue.IsNull())
		assert.True(t, refreshed.BytesValue.IsNull())
		assert.Equal(t, computeSHA256(specContent), refreshed.ContentHash.ValueString())
	})
}

// TestResourceF5XCApiDefinitionUploader_ModifyPlan_WriteOnly verifies plan-time change detection across
// unchanged write-only configs, updated specs, and user-supplied content_hash triggers.
//
// Why: Ensures accurate Terraform diffs and prevents spurious no-op diffs when content_hash is omitted.
//
// How: Executes ModifyPlan with realistic Config/Plan/State models and inspects planned attribute states.
//
// Parameters:
//   - t: test handle.
//
// Returns:
//   - None
//
// Example:
//
//	go test -run TestResourceF5XCApiDefinitionUploader_ModifyPlan_WriteOnly
func TestResourceF5XCApiDefinitionUploader_ModifyPlan_WriteOnly(t *testing.T) {
	res := &ResourceF5XCApiDefinitionUploader{}
	var schemaResp resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	specV1 := `{"openapi":"3.0.0","version":"1"}`
	hashV1 := computeSHA256(specV1)
	specV2 := `{"openapi":"3.0.0","version":"2"}`

	t.Run("unchanged-write-only-preserves-content-hash", func(t *testing.T) {
		stateData := ResourceF5XCApiDefinitionUploaderModel{
			Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
				Url:               types.StringValue("https://example.com/v1"),
				Name:              types.StringValue("my-api"),
				Version:           types.StringValue("v1"),
				Namespace:         types.StringValue("shared"),
				Description:       types.StringValue(""),
				CreationTimestamp: types.StringValue("2026-10-01T10:00:00Z"),
			},
			Status:      types.StringValue("STORED_OBJECT_STATUS_CREATED"),
			ContentHash: types.StringValue(hashV1),
			Timeouts:    testTimeoutsValue(),
		}

		configData := ResourceF5XCApiDefinitionUploaderModel{
			Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
				Name:        types.StringValue("my-api"),
				Namespace:   types.StringValue("shared"),
				Description: types.StringValue(""),
			},
			StringValue: types.StringValue(specV1),
			Timeouts:    testTimeoutsValue(),
		}

		planData := configData
		planData.StringValue = types.StringNull()
		planData.ContentHash = types.StringUnknown()
		planData.Metadata.Url = stateData.Metadata.Url
		planData.Metadata.Version = stateData.Metadata.Version
		planData.Metadata.CreationTimestamp = stateData.Metadata.CreationTimestamp
		planData.Status = stateData.Status

		state := tfsdk.State{Schema: schemaResp.Schema}
		require.False(t, state.Set(context.Background(), &stateData).HasError())
		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		require.False(t, plan.Set(context.Background(), &planData).HasError())

		req := resource.ModifyPlanRequest{
			Config: buildUploaderConfig(t, schemaResp.Schema, configData),
			State:  state,
			Plan:   plan,
		}
		resp := resource.ModifyPlanResponse{Plan: plan}
		res.ModifyPlan(context.Background(), req, &resp)
		require.False(t, resp.Diagnostics.HasError())

		var planned ResourceF5XCApiDefinitionUploaderModel
		require.False(t, resp.Plan.Get(context.Background(), &planned).HasError())
		assert.Equal(t, hashV1, planned.ContentHash.ValueString())
		assert.False(t, planned.Metadata.Url.IsUnknown())
		assert.False(t, planned.Metadata.Version.IsUnknown())
		assert.False(t, planned.PreviousVersionUrl.IsUnknown())
		assert.True(t, planned.StringValue.IsNull())
	})

	t.Run("changed-write-only-marks-versioned-fields-unknown", func(t *testing.T) {
		stateData := ResourceF5XCApiDefinitionUploaderModel{
			Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
				Url:               types.StringValue("https://example.com/v1"),
				Name:              types.StringValue("my-api"),
				Version:           types.StringValue("v1"),
				Namespace:         types.StringValue("shared"),
				Description:       types.StringValue(""),
				CreationTimestamp: types.StringValue("2026-10-01T10:00:00Z"),
			},
			Status:      types.StringValue("STORED_OBJECT_STATUS_CREATED"),
			ContentHash: types.StringValue(hashV1),
			Timeouts:    testTimeoutsValue(),
		}

		configData := ResourceF5XCApiDefinitionUploaderModel{
			Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
				Name:        types.StringValue("my-api"),
				Namespace:   types.StringValue("shared"),
				Description: types.StringValue(""),
			},
			StringValue: types.StringValue(specV2),
			Timeouts:    testTimeoutsValue(),
		}

		planData := configData
		planData.StringValue = types.StringNull()
		planData.ContentHash = types.StringUnknown()
		planData.Metadata.Url = stateData.Metadata.Url
		planData.Metadata.Version = stateData.Metadata.Version
		planData.Metadata.CreationTimestamp = stateData.Metadata.CreationTimestamp
		planData.Status = stateData.Status

		state := tfsdk.State{Schema: schemaResp.Schema}
		require.False(t, state.Set(context.Background(), &stateData).HasError())
		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		require.False(t, plan.Set(context.Background(), &planData).HasError())

		req := resource.ModifyPlanRequest{
			Config: buildUploaderConfig(t, schemaResp.Schema, configData),
			State:  state,
			Plan:   plan,
		}
		resp := resource.ModifyPlanResponse{Plan: plan}
		res.ModifyPlan(context.Background(), req, &resp)
		require.False(t, resp.Diagnostics.HasError())

		var planned ResourceF5XCApiDefinitionUploaderModel
		require.False(t, resp.Plan.Get(context.Background(), &planned).HasError())
		assert.True(t, planned.Metadata.Url.IsUnknown())
		assert.True(t, planned.Metadata.Version.IsUnknown())
		assert.True(t, planned.Metadata.CreationTimestamp.IsUnknown())
		assert.True(t, planned.PreviousVersionUrl.IsUnknown())
		assert.True(t, planned.Status.IsUnknown())
		assert.True(t, planned.ContentHash.IsUnknown())
		assert.True(t, planned.StringValue.IsNull())
	})

	t.Run("changed-explicit-content-hash-preserves-user-hash", func(t *testing.T) {
		stateData := ResourceF5XCApiDefinitionUploaderModel{
			Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
				Url:               types.StringValue("https://example.com/v1"),
				Name:              types.StringValue("my-api"),
				Version:           types.StringValue("v1"),
				Namespace:         types.StringValue("shared"),
				Description:       types.StringValue(""),
				CreationTimestamp: types.StringValue("2026-10-01T10:00:00Z"),
			},
			Status:      types.StringValue("STORED_OBJECT_STATUS_CREATED"),
			ContentHash: types.StringValue("v1-custom"),
			Timeouts:    testTimeoutsValue(),
		}

		configData := ResourceF5XCApiDefinitionUploaderModel{
			Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
				Name:        types.StringValue("my-api"),
				Namespace:   types.StringValue("shared"),
				Description: types.StringValue(""),
			},
			StringValue: types.StringValue(specV2),
			ContentHash: types.StringValue("v2-custom"),
			Timeouts:    testTimeoutsValue(),
		}

		planData := configData
		planData.StringValue = types.StringNull()
		planData.Metadata.Url = stateData.Metadata.Url
		planData.Metadata.Version = stateData.Metadata.Version
		planData.Metadata.CreationTimestamp = stateData.Metadata.CreationTimestamp
		planData.Status = stateData.Status

		state := tfsdk.State{Schema: schemaResp.Schema}
		require.False(t, state.Set(context.Background(), &stateData).HasError())
		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		require.False(t, plan.Set(context.Background(), &planData).HasError())

		req := resource.ModifyPlanRequest{
			Config: buildUploaderConfig(t, schemaResp.Schema, configData),
			State:  state,
			Plan:   plan,
		}
		resp := resource.ModifyPlanResponse{Plan: plan}
		res.ModifyPlan(context.Background(), req, &resp)
		require.False(t, resp.Diagnostics.HasError())

		var planned ResourceF5XCApiDefinitionUploaderModel
		require.False(t, resp.Plan.Get(context.Background(), &planned).HasError())
		assert.True(t, planned.Metadata.Url.IsUnknown())
		assert.Equal(t, "v2-custom", planned.ContentHash.ValueString())
	})
}

// TestResourceF5XCApiDefinitionUploader_Update_WriteOnlyBytesAndNoOp verifies Update behavior for
// bytes_value and ensures unchanged write-only configs skip HTTP PUT without persisting spec content.
//
// Why: Guarantees version churn prevention and clean wire encoding when updating specs in write-only mode.
//
// How: Exercises both a no-op write-only Update and a changed bytes_value Update via req.Config.
//
// Parameters:
//   - t: test handle.
//
// Returns:
//   - None
//
// Example:
//
//	go test -run TestResourceF5XCApiDefinitionUploader_Update_WriteOnlyBytesAndNoOp
func TestResourceF5XCApiDefinitionUploader_Update_WriteOnlyBytesAndNoOp(t *testing.T) {
	specV1 := []byte(`{"openapi":"3.0.0","version":"1.0"}`)
	hashV1 := computeSHA256Bytes(specV1)
	b64V1 := base64.StdEncoding.EncodeToString(specV1)

	specV2 := []byte(`{"openapi":"3.0.0","version":"2.0"}`)
	hashV2 := computeSHA256Bytes(specV2)
	b64V2 := base64.StdEncoding.EncodeToString(specV2)

	t.Run("unchanged-write-only-skips-put", func(t *testing.T) {
		putCount := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPut {
				putCount++
			}
			if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/stored_objects/swagger") {
				_ = json.NewEncoder(w).Encode(storedObjectListResponse{})
			}
		}))
		defer srv.Close()

		res := &ResourceF5XCApiDefinitionUploader{client: newMockAPIClient(srv)}
		var schemaResp resource.SchemaResponse
		res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

		stateData := ResourceF5XCApiDefinitionUploaderModel{
			Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
				Url:         types.StringValue(srv.URL + "/object_store/namespaces/shared/stored_objects/swagger/my-api/v1"),
				Name:        types.StringValue("my-api"),
				Version:     types.StringValue("v1"),
				Namespace:   types.StringValue("shared"),
				Description: types.StringValue("desc"),
			},
			Status:         types.StringValue("STORED_OBJECT_STATUS_CREATED"),
			ContentHash:    types.StringValue(hashV1),
			RetainVersions: types.Int64Value(2),
			ForceDelete:    types.BoolValue(false),
			Timeouts:       testTimeoutsValue(),
		}

		configData := ResourceF5XCApiDefinitionUploaderModel{
			Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
				Name:        types.StringValue("my-api"),
				Namespace:   types.StringValue("shared"),
				Description: types.StringValue("desc"),
			},
			BytesValue:     types.StringValue(b64V1),
			RetainVersions: types.Int64Value(3),
			ForceDelete:    types.BoolValue(true),
			Timeouts:       testTimeoutsValue(),
		}

		planData := configData
		planData.BytesValue = types.StringNull()
		planData.ContentHash = types.StringValue(hashV1)

		state := tfsdk.State{Schema: schemaResp.Schema}
		require.False(t, state.Set(context.Background(), &stateData).HasError())
		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		require.False(t, plan.Set(context.Background(), &planData).HasError())

		req := resource.UpdateRequest{
			Config: buildUploaderConfig(t, schemaResp.Schema, configData),
			Plan:   plan,
			State:  state,
		}
		resp := resource.UpdateResponse{State: state}
		res.Update(context.Background(), req, &resp)
		require.False(t, resp.Diagnostics.HasError())

		var updated ResourceF5XCApiDefinitionUploaderModel
		require.False(t, resp.State.Get(context.Background(), &updated).HasError())

		assert.Equal(t, 0, putCount)
		assert.True(t, updated.StringValue.IsNull())
		assert.True(t, updated.BytesValue.IsNull())
		assert.Equal(t, "v1", updated.Metadata.Version.ValueString())
		assert.Equal(t, int64(3), updated.RetainVersions.ValueInt64())
		assert.True(t, updated.ForceDelete.ValueBool())
		assert.Equal(t, hashV1, updated.ContentHash.ValueString())
	})

	t.Run("changed-bytes-value-executes-put", func(t *testing.T) {
		var receivedBody map[string]any
		putCount := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPut {
				putCount++
				require.NoError(t, json.NewDecoder(r.Body).Decode(&receivedBody))
				resp := apiDefinitionUploaderResponse{
					Status: "STORED_OBJECT_STATUS_UPDATED",
					Metadata: apiDefinitionUploaderMetadataResponse{
						Url:               r.URL.String() + "/v2",
						Name:              "my-api",
						Version:           "v2",
						Namespace:         "shared",
						Description:       "desc",
						CreationTimestamp: "2026-10-01T11:00:00Z",
					},
					BytesValue: b64V2,
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(resp)
				return
			}
			if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/stored_objects/swagger") {
				_ = json.NewEncoder(w).Encode(storedObjectListResponse{})
			}
		}))
		defer srv.Close()

		res := &ResourceF5XCApiDefinitionUploader{client: newMockAPIClient(srv)}
		var schemaResp resource.SchemaResponse
		res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

		stateData := ResourceF5XCApiDefinitionUploaderModel{
			Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
				Url:         types.StringValue(srv.URL + "/object_store/namespaces/shared/stored_objects/swagger/my-api/v1"),
				Name:        types.StringValue("my-api"),
				Version:     types.StringValue("v1"),
				Namespace:   types.StringValue("shared"),
				Description: types.StringValue("desc"),
			},
			Status:         types.StringValue("STORED_OBJECT_STATUS_CREATED"),
			ContentHash:    types.StringValue(hashV1),
			RetainVersions: types.Int64Value(2),
			Timeouts:       testTimeoutsValue(),
		}

		configData := ResourceF5XCApiDefinitionUploaderModel{
			Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
				Name:        types.StringValue("my-api"),
				Namespace:   types.StringValue("shared"),
				Description: types.StringValue("desc"),
			},
			BytesValue:     types.StringValue(b64V2),
			RetainVersions: types.Int64Value(2),
			Timeouts:       testTimeoutsValue(),
		}

		planData := configData
		planData.BytesValue = types.StringNull()
		planData.ContentHash = types.StringUnknown()

		state := tfsdk.State{Schema: schemaResp.Schema}
		require.False(t, state.Set(context.Background(), &stateData).HasError())
		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		require.False(t, plan.Set(context.Background(), &planData).HasError())

		req := resource.UpdateRequest{
			Config: buildUploaderConfig(t, schemaResp.Schema, configData),
			Plan:   plan,
			State:  state,
		}
		resp := resource.UpdateResponse{State: state}
		res.Update(context.Background(), req, &resp)
		require.False(t, resp.Diagnostics.HasError())

		assert.Equal(t, 1, putCount)
		assert.Equal(t, b64V2, receivedBody["bytes_value"])
		assert.NotContains(t, receivedBody, "string_value")

		var updated ResourceF5XCApiDefinitionUploaderModel
		require.False(t, resp.State.Get(context.Background(), &updated).HasError())
		assert.Equal(t, "v2", updated.Metadata.Version.ValueString())
		assert.Equal(t, hashV2, updated.ContentHash.ValueString())
		assert.True(t, updated.StringValue.IsNull())
		assert.True(t, updated.BytesValue.IsNull())
	})
}

// TestResourceF5XCApiDefinitionUploader_LargeSpec_WriteOnlyStateMinimal verifies that uploading
// a 4 MiB specification with write-only string_value leaves state free of the multi-megabyte payload.
//
// Why: Directly validates the core state-overhead fix for large OpenAPI specifications.
//
// How: Sends a 4 MiB OpenAPI JSON payload via string_value during Create, verifies the backend
// receives the full 4 MiB payload, and asserts all state content fields are null while ContentHash is 64 chars.
//
// Parameters:
//   - t: test handle.
//
// Returns:
//   - None
//
// Example:
//
//	go test -run TestResourceF5XCApiDefinitionUploader_LargeSpec_WriteOnlyStateMinimal
func TestResourceF5XCApiDefinitionUploader_LargeSpec_WriteOnlyStateMinimal(t *testing.T) {
	largePadding := strings.Repeat("x", 4*1024*1024)
	largeSpec := `{"openapi":"3.0.0","info":{"title":"Large","description":"` + largePadding + `"}}`
	expectedHash := computeSHA256(largeSpec)
	receivedLen := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody client.ResourceF5XCApiDefinitionUploaderCreateRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&reqBody))
		receivedLen = len(reqBody.StringValue)

		resp := apiDefinitionUploaderResponse{
			Status: "STORED_OBJECT_STATUS_CREATED",
			Metadata: apiDefinitionUploaderMetadataResponse{
				Url:               r.URL.String() + "/v1",
				Name:              "large-api",
				Version:           "v1",
				Namespace:         "shared",
				Description:       "",
				CreationTimestamp: "2026-10-01T10:00:00Z",
			},
			StringValue: largeSpec,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	res := &ResourceF5XCApiDefinitionUploader{client: newMockAPIClient(srv)}
	var schemaResp resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	configModel := ResourceF5XCApiDefinitionUploaderModel{
		Metadata: ResourceF5XCApiDefinitionUploaderMetadataModel{
			Name:        types.StringValue("large-api"),
			Namespace:   types.StringValue("shared"),
			Description: types.StringValue(""),
		},
		StringValue:    types.StringValue(largeSpec),
		RetainVersions: types.Int64Value(2),
		ForceDelete:    types.BoolValue(false),
		Timeouts:       testTimeoutsValue(),
	}

	planModel := configModel
	planModel.StringValue = types.StringNull()
	planModel.ContentHash = types.StringUnknown()

	planState := tfsdk.State{Schema: schemaResp.Schema}
	require.False(t, planState.Set(context.Background(), &planModel).HasError())

	req := resource.CreateRequest{
		Config: buildUploaderConfig(t, schemaResp.Schema, configModel),
		Plan:   tfsdk.Plan(planState),
	}
	resp := resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	res.Create(context.Background(), req, &resp)
	require.False(t, resp.Diagnostics.HasError())

	assert.Equal(t, len(largeSpec), receivedLen)

	var created ResourceF5XCApiDefinitionUploaderModel
	require.False(t, resp.State.Get(context.Background(), &created).HasError())
	assert.Len(t, created.ContentHash.ValueString(), 64)
	assert.Equal(t, expectedHash, created.ContentHash.ValueString())
	assert.True(t, created.StringValue.IsNull())
	assert.True(t, created.BytesValue.IsNull())
}
