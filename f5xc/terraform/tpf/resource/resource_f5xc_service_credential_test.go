//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package resource

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gopkg.volterra.us/stdlib/client/vesapi"
	"gopkg.volterra.us/stdlib/codec"
	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/client"
	ves_io_schema_api_credential "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/api_credential"
	ves_io_schema_combined "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/combined"
)

type recordedServiceCredentialHTTPRequest struct {
	method string
	path   string
	body   string
}

func newServiceCredentialTestClient(t *testing.T, handler http.HandlerFunc) *client.APIClient {
	t.Helper()
	vesClient, err := vesapi.NewAPIClient(
		"https://mock.volterra.io",
		ves_io_schema_combined.MDR,
		vesapi.WithCfgApiToken("test-token"),
		vesapi.WithCfgRoundTripperFn(func(req *http.Request, _ http.RoundTripper) (*http.Response, error) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			return rec.Result(), nil
		}),
	)
	require.NoError(t, err)
	return &client.APIClient{
		APIClient: vesClient,
		Config:    &client.ClientTimeouts{},
	}
}

func serviceCredentialTestTimeouts() timeouts.Value {
	return timeouts.Value{
		Object: types.ObjectNull(map[string]attr.Type{
			"create": types.StringType,
			"read":   types.StringType,
			"update": types.StringType,
			"delete": types.StringType,
		}),
	}
}

func writeServiceCredentialProtoJSON(t *testing.T, w http.ResponseWriter, msg proto.Message) {
	t.Helper()
	out, err := codec.ToJSON(msg, codec.ToWithUseProtoFieldName())
	require.NoError(t, err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(out))
	require.NoError(t, err)
}

// TestResourceF5XCServiceCredential_VK8sKubeConfigLifecycle verifies XC 10.0 RBAC routing
// for vk8s_kubeconfig service credentials: Create targets "system", while Read, Update
// (URI + ReplaceServiceCredentialsRequest.Namespace + pre-update hook), and Delete target
// service_credential_choice.vk8s_kubeconfig.vk8s_namespace without state drift.
func TestResourceF5XCServiceCredential_VK8sKubeConfigLifecycle(t *testing.T) {
	ctx := context.Background()

	var (
		mu             sync.Mutex
		requests       []recordedServiceCredentialHTTPRequest
		userGroupNames []string
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var bodyBytes []byte
		if r.Body != nil {
			var err error
			bodyBytes, err = io.ReadAll(r.Body)
			require.NoError(t, err)
		}

		mu.Lock()
		requests = append(requests, recordedServiceCredentialHTTPRequest{
			method: r.Method,
			path:   r.URL.Path,
			body:   string(bodyBytes),
		})
		currentGroups := append([]string(nil), userGroupNames...)
		mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/public/namespaces/system/service_credentials":
			writeServiceCredentialProtoJSON(t, w, &ves_io_schema_api_credential.CreateResponse{
				Name: "svc-cred-xyz",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/public/namespaces/my-vk8s-ns/service_credentials":
			writeServiceCredentialProtoJSON(t, w, &ves_io_schema_api_credential.ListServiceCredentialsResponse{
				Items: []*ves_io_schema_api_credential.ListServiceCredentialsResponseItem{
					{
						Name:           "svc-cred-xyz",
						Namespace:      "my-vk8s-ns",
						UserGroupNames: currentGroups,
					},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/public/namespaces/my-vk8s-ns/service_credentials/svc-cred-xyz":
			writeServiceCredentialProtoJSON(t, w, &ves_io_schema_api_credential.GetServiceCredentialsResponse{
				Active: true,
			})
		case r.Method == http.MethodPut && r.URL.Path == "/public/namespaces/my-vk8s-ns/service_credentials/svc-cred-xyz":
			var replaceReq ves_io_schema_api_credential.ReplaceServiceCredentialsRequest
			require.NoError(t, codec.FromJSON(string(bodyBytes), &replaceReq))
			mu.Lock()
			userGroupNames = append([]string(nil), replaceReq.GetUserGroupNames()...)
			mu.Unlock()
			writeServiceCredentialProtoJSON(t, w, &ves_io_schema_api_credential.ReplaceServiceCredentialsResponse{})
		case r.Method == http.MethodPost && r.URL.Path == "/public/namespaces/my-vk8s-ns/revoke/service_credentials":
			writeServiceCredentialProtoJSON(t, w, &ves_io_schema_api_credential.StatusResponse{})
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":5,"message":"unexpected namespace or endpoint"}`))
		}
	})

	res := &ResourceF5XCServiceCredential{client: newServiceCredentialTestClient(t, handler)}

	var schemaResp resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())

	vk8sObj, diags := types.ObjectValue(
		ServiceCredentialServiceCredentialChoiceVk8SKubeconfigModelAttributeMap,
		map[string]attr.Value{
			"vk8s_cluster_name": types.StringValue("vk8s-1"),
			"vk8s_namespace":    types.StringValue("my-vk8s-ns"),
		},
	)
	require.False(t, diags.HasError())

	choiceObj, diags := types.ObjectValue(
		ServiceCredentialServiceCredentialChoiceModelAttributeMap,
		map[string]attr.Value{
			"api_certificate": types.ObjectNull(ServiceCredentialServiceCredentialChoiceApiCertificateModelAttributeMap),
			"api_token":       types.BoolNull(),
			"site_kubeconfig": types.ObjectNull(ServiceCredentialServiceCredentialChoiceSiteKubeconfigModelAttributeMap),
			"vk8s_kubeconfig": vk8sObj,
		},
	)
	require.False(t, diags.HasError())

	roleElemType := types.ObjectType{AttrTypes: ServiceCredentialNamespaceRolesModelAttributeMap}
	roleObj, diags := types.ObjectValue(ServiceCredentialNamespaceRolesModelAttributeMap, map[string]attr.Value{
		"namespace": types.StringValue("my-vk8s-ns"),
		"role":      types.StringValue("ves-io-admin-role"),
	})
	require.False(t, diags.HasError())
	namespaceRolesList, diags := types.ListValue(roleElemType, []attr.Value{roleObj})
	require.False(t, diags.HasError())

	planModel := ResourceF5XCServiceCredentialModel{
		Name:                    types.StringValue("my-svc-cred"),
		Namespace:               types.StringValue("system"),
		ExpirationDays:          types.Int64Value(30),
		NamespaceRoles:          namespaceRolesList,
		ServiceCredentialChoice: choiceObj,
		UserGroupNames:          types.ListNull(types.StringType),
		CredentialName:          types.StringUnknown(),
		RenewTrigger:            types.BoolValue(false),
		Timeouts:                serviceCredentialTestTimeouts(),
	}

	// 1. Create (POST /public/namespaces/system/service_credentials -> GET /public/namespaces/my-vk8s-ns/service_credentials)
	planState := tfsdk.State{Schema: schemaResp.Schema}
	require.False(t, planState.Set(ctx, &planModel).HasError())

	var createResp resource.CreateResponse
	createResp.State = tfsdk.State{Schema: schemaResp.Schema}
	res.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan(planState)}, &createResp)
	require.False(t, createResp.Diagnostics.HasError(), "Create diagnostics: %v", createResp.Diagnostics)

	var stateAfterCreate ResourceF5XCServiceCredentialModel
	require.False(t, createResp.State.Get(ctx, &stateAfterCreate).HasError())
	assert.Equal(t, "svc-cred-xyz", stateAfterCreate.CredentialName.ValueString())
	assert.Equal(t, "system", stateAfterCreate.Namespace.ValueString(), "state.Namespace must stay 'system' and not drift to vK8s namespace")
	assert.Equal(t, "my-svc-cred", stateAfterCreate.Name.ValueString())

	// 2. Read (GET /public/namespaces/my-vk8s-ns/service_credentials)
	var readResp resource.ReadResponse
	readResp.State = createResp.State
	res.Read(ctx, resource.ReadRequest{State: createResp.State}, &readResp)
	require.False(t, readResp.Diagnostics.HasError(), "Read diagnostics: %v", readResp.Diagnostics)

	var stateAfterRead ResourceF5XCServiceCredentialModel
	require.False(t, readResp.State.Get(ctx, &stateAfterRead).HasError())
	assert.Equal(t, "system", stateAfterRead.Namespace.ValueString())
	assert.Equal(t, "svc-cred-xyz", stateAfterRead.CredentialName.ValueString())

	// 3. Update (pre-update hook Get + PUT ReplaceServiceCredentials + Read in my-vk8s-ns)
	updatedGroups, diags := types.ListValueFrom(ctx, types.StringType, []string{"ug-ops"})
	require.False(t, diags.HasError())

	updatePlanModel := stateAfterRead
	updatePlanModel.UserGroupNames = updatedGroups
	updatePlanModel.RenewTrigger = types.BoolValue(true)

	updatePlanState := tfsdk.State{Schema: schemaResp.Schema}
	require.False(t, updatePlanState.Set(ctx, &updatePlanModel).HasError())

	var updateResp resource.UpdateResponse
	updateResp.State = tfsdk.State{Schema: schemaResp.Schema}
	res.Update(ctx, resource.UpdateRequest{
		State: readResp.State,
		Plan:  tfsdk.Plan(updatePlanState),
	}, &updateResp)
	require.False(t, updateResp.Diagnostics.HasError(), "Update diagnostics: %v", updateResp.Diagnostics)

	var stateAfterUpdate ResourceF5XCServiceCredentialModel
	require.False(t, updateResp.State.Get(ctx, &stateAfterUpdate).HasError())
	assert.Equal(t, "system", stateAfterUpdate.Namespace.ValueString())
	var gotGroups []string
	require.False(t, stateAfterUpdate.UserGroupNames.ElementsAs(ctx, &gotGroups, false).HasError())
	assert.Equal(t, []string{"ug-ops"}, gotGroups)

	// 4. Delete (POST /public/namespaces/my-vk8s-ns/revoke/service_credentials)
	var deleteResp resource.DeleteResponse
	res.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, &deleteResp)
	require.False(t, deleteResp.Diagnostics.HasError(), "Delete diagnostics: %v", deleteResp.Diagnostics)

	mu.Lock()
	recorded := append([]recordedServiceCredentialHTTPRequest(nil), requests...)
	mu.Unlock()

	require.Len(t, recorded, 7)
	assert.Equal(t, http.MethodPost, recorded[0].method)
	assert.Equal(t, "/public/namespaces/system/service_credentials", recorded[0].path)

	assert.Equal(t, http.MethodGet, recorded[1].method)
	assert.Equal(t, "/public/namespaces/my-vk8s-ns/service_credentials", recorded[1].path)

	assert.Equal(t, http.MethodGet, recorded[2].method)
	assert.Equal(t, "/public/namespaces/my-vk8s-ns/service_credentials", recorded[2].path)

	assert.Equal(t, http.MethodGet, recorded[3].method)
	assert.Equal(t, "/public/namespaces/my-vk8s-ns/service_credentials/svc-cred-xyz", recorded[3].path)

	assert.Equal(t, http.MethodPut, recorded[4].method)
	assert.Equal(t, "/public/namespaces/my-vk8s-ns/service_credentials/svc-cred-xyz", recorded[4].path)
	var putBody map[string]any
	require.NoError(t, json.Unmarshal([]byte(recorded[4].body), &putBody))
	assert.Equal(t, "my-vk8s-ns", putBody["namespace"], "ReplaceServiceCredentialsRequest.namespace must be set to effective vK8s namespace")
	assert.Equal(t, "svc-cred-xyz", putBody["name"], "ReplaceServiceCredentialsRequest.name must be set to backend credential_name")

	assert.Equal(t, http.MethodGet, recorded[5].method)
	assert.Equal(t, "/public/namespaces/my-vk8s-ns/service_credentials", recorded[5].path)

	assert.Equal(t, http.MethodPost, recorded[6].method)
	assert.Equal(t, "/public/namespaces/my-vk8s-ns/revoke/service_credentials", recorded[6].path)
	var revokeBody map[string]any
	require.NoError(t, json.Unmarshal([]byte(recorded[6].body), &revokeBody))
	assert.Equal(t, "svc-cred-xyz", revokeBody["name"])
}

// TestResourceF5XCServiceCredential_SystemNamespaceLifecycle ensures non-vK8s service
// credentials continue targeting "system" across Create, Read, Update, and Delete.
func TestResourceF5XCServiceCredential_SystemNamespaceLifecycle(t *testing.T) {
	ctx := context.Background()

	var (
		mu       sync.Mutex
		requests []recordedServiceCredentialHTTPRequest
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var bodyBytes []byte
		if r.Body != nil {
			var err error
			bodyBytes, err = io.ReadAll(r.Body)
			require.NoError(t, err)
		}

		mu.Lock()
		requests = append(requests, recordedServiceCredentialHTTPRequest{
			method: r.Method,
			path:   r.URL.Path,
			body:   string(bodyBytes),
		})
		mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/public/namespaces/system/service_credentials":
			writeServiceCredentialProtoJSON(t, w, &ves_io_schema_api_credential.CreateResponse{
				Name: "svc-tok-1",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/public/namespaces/system/service_credentials":
			writeServiceCredentialProtoJSON(t, w, &ves_io_schema_api_credential.ListServiceCredentialsResponse{
				Items: []*ves_io_schema_api_credential.ListServiceCredentialsResponseItem{
					{
						Name:      "svc-tok-1",
						Namespace: "system",
					},
				},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/public/namespaces/system/service_credentials/svc-tok-1":
			writeServiceCredentialProtoJSON(t, w, &ves_io_schema_api_credential.ReplaceServiceCredentialsResponse{})
		case r.Method == http.MethodPost && r.URL.Path == "/public/namespaces/system/revoke/service_credentials":
			writeServiceCredentialProtoJSON(t, w, &ves_io_schema_api_credential.StatusResponse{})
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":5,"message":"unexpected endpoint"}`))
		}
	})

	res := &ResourceF5XCServiceCredential{client: newServiceCredentialTestClient(t, handler)}

	var schemaResp resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())

	choiceObj, diags := types.ObjectValue(
		ServiceCredentialServiceCredentialChoiceModelAttributeMap,
		map[string]attr.Value{
			"api_certificate": types.ObjectNull(ServiceCredentialServiceCredentialChoiceApiCertificateModelAttributeMap),
			"api_token":       types.BoolValue(true),
			"site_kubeconfig": types.ObjectNull(ServiceCredentialServiceCredentialChoiceSiteKubeconfigModelAttributeMap),
			"vk8s_kubeconfig": types.ObjectNull(ServiceCredentialServiceCredentialChoiceVk8SKubeconfigModelAttributeMap),
		},
	)
	require.False(t, diags.HasError())

	roleElemType := types.ObjectType{AttrTypes: ServiceCredentialNamespaceRolesModelAttributeMap}

	planModel := ResourceF5XCServiceCredentialModel{
		Name:                    types.StringValue("my-svc-tok"),
		Namespace:               types.StringValue("system"),
		ExpirationDays:          types.Int64Value(7),
		NamespaceRoles:          types.ListNull(roleElemType),
		ServiceCredentialChoice: choiceObj,
		UserGroupNames:          types.ListNull(types.StringType),
		CredentialName:          types.StringUnknown(),
		RenewTrigger:            types.BoolValue(false),
		Timeouts:                serviceCredentialTestTimeouts(),
	}

	planState := tfsdk.State{Schema: schemaResp.Schema}
	require.False(t, planState.Set(ctx, &planModel).HasError())

	var createResp resource.CreateResponse
	createResp.State = tfsdk.State{Schema: schemaResp.Schema}
	res.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan(planState)}, &createResp)
	require.False(t, createResp.Diagnostics.HasError(), "Create diagnostics: %v", createResp.Diagnostics)

	var createdModel ResourceF5XCServiceCredentialModel
	require.False(t, createResp.State.Get(ctx, &createdModel).HasError())
	createdModel.ExpirationDays = types.Int64Value(14)

	updatePlanState := tfsdk.State{Schema: schemaResp.Schema}
	require.False(t, updatePlanState.Set(ctx, &createdModel).HasError())

	var updateResp resource.UpdateResponse
	updateResp.State = tfsdk.State{Schema: schemaResp.Schema}
	res.Update(ctx, resource.UpdateRequest{
		State: createResp.State,
		Plan:  tfsdk.Plan(updatePlanState),
	}, &updateResp)
	require.False(t, updateResp.Diagnostics.HasError(), "Update diagnostics: %v", updateResp.Diagnostics)

	var deleteResp resource.DeleteResponse
	res.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, &deleteResp)
	require.False(t, deleteResp.Diagnostics.HasError(), "Delete diagnostics: %v", deleteResp.Diagnostics)

	mu.Lock()
	recorded := append([]recordedServiceCredentialHTTPRequest(nil), requests...)
	mu.Unlock()

	require.Len(t, recorded, 5)
	assert.Equal(t, "/public/namespaces/system/service_credentials", recorded[0].path)
	assert.Equal(t, "/public/namespaces/system/service_credentials", recorded[1].path)
	assert.Equal(t, "/public/namespaces/system/service_credentials/svc-tok-1", recorded[2].path)
	assert.Equal(t, "/public/namespaces/system/service_credentials", recorded[3].path)
	assert.Equal(t, "/public/namespaces/system/revoke/service_credentials", recorded[4].path)
}

func TestResourceF5XCServiceCredential_ImportStateAndRead(t *testing.T) {
	ctx := context.Background()

	var (
		mu       sync.Mutex
		requests []recordedServiceCredentialHTTPRequest
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, recordedServiceCredentialHTTPRequest{
			method: r.Method,
			path:   r.URL.Path,
		})
		mu.Unlock()

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/public/namespaces/my-vk8s-ns/service_credentials":
			writeServiceCredentialProtoJSON(t, w, &ves_io_schema_api_credential.ListServiceCredentialsResponse{
				Items: []*ves_io_schema_api_credential.ListServiceCredentialsResponseItem{
					{
						Name:      "svc-cred-xyz",
						Namespace: "my-vk8s-ns",
					},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/public/namespaces/system/service_credentials":
			writeServiceCredentialProtoJSON(t, w, &ves_io_schema_api_credential.ListServiceCredentialsResponse{
				Items: []*ves_io_schema_api_credential.ListServiceCredentialsResponseItem{
					{
						Name:      "svc-tok-1",
						Namespace: "system",
					},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":5,"message":"unexpected endpoint"}`))
		}
	})

	res := &ResourceF5XCServiceCredential{client: newServiceCredentialTestClient(t, handler)}

	var schemaResp resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())

	emptyState := func() tfsdk.State {
		objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
		vals := make(map[string]tftypes.Value, len(objType.AttributeTypes))
		for k, attrType := range objType.AttributeTypes {
			vals[k] = tftypes.NewValue(attrType, nil)
		}
		return tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(objType, vals),
		}
	}

	// 1. Import vk8s_kubeconfig credential via "<vk8s_namespace>/<credential_name>" -> Read
	importResp := resource.ImportStateResponse{State: emptyState()}
	res.ImportState(ctx, resource.ImportStateRequest{ID: "my-vk8s-ns/svc-cred-xyz"}, &importResp)
	require.False(t, importResp.Diagnostics.HasError(), "ImportState diagnostics: %v", importResp.Diagnostics)

	readResp := resource.ReadResponse{State: importResp.State}
	res.Read(ctx, resource.ReadRequest{State: importResp.State}, &readResp)
	require.False(t, readResp.Diagnostics.HasError(), "Read diagnostics: %v", readResp.Diagnostics)

	var importedModel ResourceF5XCServiceCredentialModel
	require.False(t, readResp.State.Get(ctx, &importedModel).HasError())
	assert.Equal(t, "system", importedModel.Namespace.ValueString())
	assert.Equal(t, "svc-cred-xyz", importedModel.CredentialName.ValueString())

	// 2. Import system credential via "system/<credential_name>" -> Read
	sysImportResp := resource.ImportStateResponse{State: emptyState()}
	res.ImportState(ctx, resource.ImportStateRequest{ID: "system/svc-tok-1"}, &sysImportResp)
	require.False(t, sysImportResp.Diagnostics.HasError(), "ImportState diagnostics: %v", sysImportResp.Diagnostics)

	sysReadResp := resource.ReadResponse{State: sysImportResp.State}
	res.Read(ctx, resource.ReadRequest{State: sysImportResp.State}, &sysReadResp)
	require.False(t, sysReadResp.Diagnostics.HasError(), "Read diagnostics: %v", sysReadResp.Diagnostics)

	mu.Lock()
	recorded := append([]recordedServiceCredentialHTTPRequest(nil), requests...)
	mu.Unlock()

	require.Len(t, recorded, 2)
	assert.Equal(t, "/public/namespaces/my-vk8s-ns/service_credentials", recorded[0].path)
	assert.Equal(t, "/public/namespaces/system/service_credentials", recorded[1].path)
}

// TestResourceF5XCServiceCredential_ImportThreePartThenDelete verifies that a 3-part import ID
// ("system/<vk8s_namespace>/<credential_name>") routes both the follow-up Read and
// the Delete (revoke) to the vK8s namespace with the backend credential name.
func TestResourceF5XCServiceCredential_ImportThreePartThenDelete(t *testing.T) {
	ctx := context.Background()

	var (
		mu       sync.Mutex
		requests []recordedServiceCredentialHTTPRequest
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var bodyBytes []byte
		if r.Body != nil {
			var err error
			bodyBytes, err = io.ReadAll(r.Body)
			require.NoError(t, err)
		}

		mu.Lock()
		requests = append(requests, recordedServiceCredentialHTTPRequest{
			method: r.Method,
			path:   r.URL.Path,
			body:   string(bodyBytes),
		})
		mu.Unlock()

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/public/namespaces/my-vk8s-ns/service_credentials":
			writeServiceCredentialProtoJSON(t, w, &ves_io_schema_api_credential.ListServiceCredentialsResponse{
				Items: []*ves_io_schema_api_credential.ListServiceCredentialsResponseItem{
					{
						Name:      "svc-cred-789",
						Namespace: "my-vk8s-ns",
					},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/public/namespaces/my-vk8s-ns/revoke/service_credentials":
			writeServiceCredentialProtoJSON(t, w, &ves_io_schema_api_credential.StatusResponse{})
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":5,"message":"unexpected endpoint"}`))
		}
	})

	res := &ResourceF5XCServiceCredential{client: newServiceCredentialTestClient(t, handler)}

	var schemaResp resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())

	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	vals := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for k, attrType := range objType.AttributeTypes {
		vals[k] = tftypes.NewValue(attrType, nil)
	}
	emptyState := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, vals)}

	importResp := resource.ImportStateResponse{State: emptyState}
	res.ImportState(ctx, resource.ImportStateRequest{ID: "system/my-vk8s-ns/svc-cred-789"}, &importResp)
	require.False(t, importResp.Diagnostics.HasError(), "ImportState diagnostics: %v", importResp.Diagnostics)

	readResp := resource.ReadResponse{State: importResp.State}
	res.Read(ctx, resource.ReadRequest{State: importResp.State}, &readResp)
	require.False(t, readResp.Diagnostics.HasError(), "Read diagnostics: %v", readResp.Diagnostics)

	var importedModel ResourceF5XCServiceCredentialModel
	require.False(t, readResp.State.Get(ctx, &importedModel).HasError())
	assert.Equal(t, "system", importedModel.Namespace.ValueString())
	assert.Equal(t, "svc-cred-789", importedModel.CredentialName.ValueString())

	var deleteResp resource.DeleteResponse
	res.Delete(ctx, resource.DeleteRequest{State: readResp.State}, &deleteResp)
	require.False(t, deleteResp.Diagnostics.HasError(), "Delete diagnostics: %v", deleteResp.Diagnostics)

	mu.Lock()
	recorded := append([]recordedServiceCredentialHTTPRequest(nil), requests...)
	mu.Unlock()

	require.Len(t, recorded, 2)
	assert.Equal(t, http.MethodGet, recorded[0].method)
	assert.Equal(t, "/public/namespaces/my-vk8s-ns/service_credentials", recorded[0].path)
	assert.Equal(t, http.MethodPost, recorded[1].method)
	assert.Equal(t, "/public/namespaces/my-vk8s-ns/revoke/service_credentials", recorded[1].path)
	var revokeBody map[string]any
	require.NoError(t, json.Unmarshal([]byte(recorded[1].body), &revokeBody))
	assert.Equal(t, "svc-cred-789", revokeBody["name"])
}

// TestResourceF5XCServiceCredential_ReadNotFoundRemovesResource verifies that Read removes the
// resource from state when the credential is no longer in the vK8s namespace list.
func TestResourceF5XCServiceCredential_ReadNotFoundRemovesResource(t *testing.T) {
	ctx := context.Background()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/public/namespaces/my-vk8s-ns/service_credentials" {
			writeServiceCredentialProtoJSON(t, w, &ves_io_schema_api_credential.ListServiceCredentialsResponse{
				Items: []*ves_io_schema_api_credential.ListServiceCredentialsResponseItem{
					{
						Name:      "some-other-credential",
						Namespace: "my-vk8s-ns",
					},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":5,"message":"unexpected endpoint"}`))
	})

	res := &ResourceF5XCServiceCredential{client: newServiceCredentialTestClient(t, handler)}

	var schemaResp resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())

	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	vals := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for k, attrType := range objType.AttributeTypes {
		vals[k] = tftypes.NewValue(attrType, nil)
	}
	emptyState := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, vals)}

	importResp := resource.ImportStateResponse{State: emptyState}
	res.ImportState(ctx, resource.ImportStateRequest{ID: "my-vk8s-ns/svc-cred-789"}, &importResp)
	require.False(t, importResp.Diagnostics.HasError(), "ImportState diagnostics: %v", importResp.Diagnostics)

	readResp := resource.ReadResponse{State: importResp.State}
	res.Read(ctx, resource.ReadRequest{State: importResp.State}, &readResp)
	require.False(t, readResp.Diagnostics.HasError(), "Read diagnostics: %v", readResp.Diagnostics)
	assert.True(t, readResp.State.Raw.IsNull(), "Read must remove the resource from state when the credential is not found")
}
