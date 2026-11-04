//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package manual

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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
	assert.Contains(t, attrs, "content_hash")
	assert.Contains(t, attrs, "retain_versions")
	assert.Contains(t, attrs, "previous_version_url")
	assert.Contains(t, attrs, "force_delete")
	assert.Contains(t, attrs, "status")
	assert.Contains(t, attrs, "timeouts")
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
			resp := apiDefinitionUploaderResponse{
				Status: "STORED_OBJECT_STATUS_UPDATED",
				Metadata: apiDefinitionUploaderMetadataResponse{
					Url:               r.URL.String() + "/v2-26-09-24",
					Name:              "my-api",
					Version:           "v2-26-09-24",
					Namespace:         "shared",
					Description:       "",
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
			Url:       types.StringValue(srv.URL + "/object_store/namespaces/shared/stored_objects/swagger/my-api/v1-26-09-24"),
			Name:      types.StringValue("my-api"),
			Version:   types.StringValue("v1-26-09-24"),
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
	require.False(t, resp.Diagnostics.HasError())

	var updatedData ResourceF5XCApiDefinitionUploaderModel
	require.False(t, resp.State.Get(context.Background(), &updatedData).HasError())

	assert.Equal(t, 1, putCount, "PUT must be executed when content changes")
	assert.Equal(t, "v2-26-09-24", updatedData.Metadata.Version.ValueString())
	assert.Equal(t, computeSHA256(newSpec), updatedData.ContentHash.ValueString())
	assert.Equal(t, srv.URL+"/object_store/namespaces/shared/stored_objects/swagger/my-api/v1-26-09-24", updatedData.PreviousVersionUrl.ValueString())
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
		var req client.ResourceF5XCApiDefinitionUploaderApiModel
		_ = json.NewDecoder(r.Body).Decode(&req)
		receivedDesc = req.Metadata.Description

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
