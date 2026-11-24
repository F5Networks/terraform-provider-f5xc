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
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gopkg.volterra.us/stdlib/client/vesapi"
	"gopkg.volterra.us/stdlib/codec"
	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/client"
	ves_io_schema_combined "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/combined"
	ves_io_schema_known_label "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/known_label"
)

type recordedKnownLabelHTTPRequest struct {
	method string
	path   string
	body   string
}

func newKnownLabelTestClient(t *testing.T, handler http.HandlerFunc) *client.APIClient {
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

func knownLabelTestTimeouts() timeouts.Value {
	return timeouts.Value{
		Object: types.ObjectNull(map[string]attr.Type{
			"create": types.StringType,
			"read":   types.StringType,
			"delete": types.StringType,
		}),
	}
}

func knownLabelEmptyValue(t *testing.T, ctx context.Context, s schema.Schema) tftypes.Value {
	t.Helper()
	objType, ok := s.Type().TerraformType(ctx).(tftypes.Object)
	require.True(t, ok)
	vals := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for k, attrType := range objType.AttributeTypes {
		vals[k] = tftypes.NewValue(attrType, nil)
	}
	return tftypes.NewValue(objType, vals)
}

func writeKnownLabelProtoJSON(t *testing.T, w http.ResponseWriter, msg proto.Message) {
	t.Helper()
	out, err := codec.ToJSON(msg, codec.ToWithUseProtoFieldName())
	require.NoError(t, err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(out))
	require.NoError(t, err)
}

// TestResourceF5XCKnownLabel_CompositeKeyReadCreateDelete verifies that f5xc_known_label
// matches entries by the composite (key, value) tuple in Read(), preventing state corruption
// when multiple labels share the same key.
func TestResourceF5XCKnownLabel_CompositeKeyReadCreateDelete(t *testing.T) {
	ctx := context.Background()

	var (
		mu       sync.Mutex
		requests []recordedKnownLabelHTTPRequest
	)

	labelsResp := &ves_io_schema_known_label.GetResponse{
		Label: []*ves_io_schema_known_label.LabelType{
			{
				Key:         "site",
				Value:       "aws-1",
				Description: "AWS region label",
			},
			{
				Key:         "site",
				Value:       "azure-2",
				Description: "Azure region label",
			},
			{
				Key:         "empty-val",
				Value:       "",
				Description: "Label with empty value",
			},
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var bodyBytes []byte
		if r.Body != nil {
			var err error
			bodyBytes, err = io.ReadAll(r.Body)
			require.NoError(t, err)
		}

		mu.Lock()
		requests = append(requests, recordedKnownLabelHTTPRequest{
			method: r.Method,
			path:   r.URL.Path,
			body:   string(bodyBytes),
		})
		mu.Unlock()

		switch r.URL.Path {
		case "/public/namespaces/shared/known_label/create":
			writeKnownLabelProtoJSON(t, w, &ves_io_schema_known_label.CreateResponse{})
		case "/public/namespaces/shared/known_labels":
			writeKnownLabelProtoJSON(t, w, labelsResp)
		case "/public/namespaces/shared/known_label/delete":
			writeKnownLabelProtoJSON(t, w, &ves_io_schema_known_label.DeleteResponse{})
		default:
			t.Errorf("unexpected HTTP request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})

	r := &ResourceF5XCKnownLabel{client: newKnownLabelTestClient(t, handler)}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())

	t.Run("create-and-read-second-entry-with-same-key", func(t *testing.T) {
		mu.Lock()
		requests = nil
		mu.Unlock()

		model := ResourceF5XCKnownLabelModel{
			Namespace:   types.StringValue("shared"),
			Key:         types.StringValue("site"),
			Value:       types.StringValue("azure-2"),
			Description: types.StringValue("Azure region label"),
			Timeouts:    knownLabelTestTimeouts(),
		}

		plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: knownLabelEmptyValue(t, ctx, schemaResp.Schema)}
		require.False(t, plan.Set(ctx, &model).HasError())
		config := tfsdk.Config{Schema: schemaResp.Schema, Raw: plan.Raw}
		createResp := resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: knownLabelEmptyValue(t, ctx, schemaResp.Schema)}}

		r.Create(ctx, resource.CreateRequest{Config: config, Plan: plan}, &createResp)
		require.False(t, createResp.Diagnostics.HasError(), "Create diagnostics: %v", createResp.Diagnostics)

		var state ResourceF5XCKnownLabelModel
		require.False(t, createResp.State.Get(ctx, &state).HasError())
		assert.Equal(t, "site", state.Key.ValueString())
		assert.Equal(t, "azure-2", state.Value.ValueString())
		assert.Equal(t, "Azure region label", state.Description.ValueString())
	})

	t.Run("read-null-value-matches-only-empty-backend-value", func(t *testing.T) {
		missingEmpty := ResourceF5XCKnownLabelModel{
			Namespace:   types.StringValue("shared"),
			Key:         types.StringValue("site"),
			Value:       types.StringNull(),
			Description: types.StringValue(""),
			Timeouts:    knownLabelTestTimeouts(),
		}
		state := tfsdk.State{Schema: schemaResp.Schema, Raw: knownLabelEmptyValue(t, ctx, schemaResp.Schema)}
		require.False(t, state.Set(ctx, &missingEmpty).HasError())
		readResp := resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: state.Raw}}

		r.Read(ctx, resource.ReadRequest{State: state}, &readResp)
		require.False(t, readResp.Diagnostics.HasError())
		assert.True(t, readResp.State.Raw.IsNull(), "null value must not match non-empty 'aws-1' entry")

		presentEmpty := ResourceF5XCKnownLabelModel{
			Namespace:   types.StringValue("shared"),
			Key:         types.StringValue("empty-val"),
			Value:       types.StringNull(),
			Description: types.StringValue(""),
			Timeouts:    knownLabelTestTimeouts(),
		}
		state2 := tfsdk.State{Schema: schemaResp.Schema, Raw: knownLabelEmptyValue(t, ctx, schemaResp.Schema)}
		require.False(t, state2.Set(ctx, &presentEmpty).HasError())
		readResp2 := resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: state2.Raw}}

		r.Read(ctx, resource.ReadRequest{State: state2}, &readResp2)
		require.False(t, readResp2.Diagnostics.HasError())
		require.False(t, readResp2.State.Raw.IsNull())

		var got ResourceF5XCKnownLabelModel
		require.False(t, readResp2.State.Get(ctx, &got).HasError())
		assert.Equal(t, "empty-val", got.Key.ValueString())
		assert.True(t, got.Value.IsNull())
		assert.Equal(t, "Label with empty value", got.Description.ValueString())
	})

	t.Run("delete-sends-composite-key-and-value", func(t *testing.T) {
		mu.Lock()
		requests = nil
		mu.Unlock()

		model := ResourceF5XCKnownLabelModel{
			Namespace:   types.StringValue("shared"),
			Key:         types.StringValue("site"),
			Value:       types.StringValue("azure-2"),
			Description: types.StringValue("Azure region label"),
			Timeouts:    knownLabelTestTimeouts(),
		}
		state := tfsdk.State{Schema: schemaResp.Schema, Raw: knownLabelEmptyValue(t, ctx, schemaResp.Schema)}
		require.False(t, state.Set(ctx, &model).HasError())

		delResp := resource.DeleteResponse{}
		r.Delete(ctx, resource.DeleteRequest{State: state}, &delResp)
		require.False(t, delResp.Diagnostics.HasError(), "Delete diagnostics: %v", delResp.Diagnostics)

		mu.Lock()
		defer mu.Unlock()
		require.Len(t, requests, 1)
		assert.Equal(t, http.MethodPost, requests[0].method)
		var bodyMap map[string]string
		require.NoError(t, json.Unmarshal([]byte(requests[0].body), &bodyMap))
		assert.Equal(t, "site", bodyMap["key"])
		assert.Equal(t, "azure-2", bodyMap["value"])
	})

	t.Run("import-composite-key-with-slash", func(t *testing.T) {
		importResp := resource.ImportStateResponse{
			State: tfsdk.State{Schema: schemaResp.Schema, Raw: knownLabelEmptyValue(t, ctx, schemaResp.Schema)},
		}
		r.ImportState(ctx, resource.ImportStateRequest{ID: "shared/ves.io/region=ves-io-par"}, &importResp)
		require.False(t, importResp.Diagnostics.HasError(), "Import diagnostics: %v", importResp.Diagnostics)

		var ns, key, val types.String
		require.False(t, importResp.State.GetAttribute(ctx, path.Root("namespace"), &ns).HasError())
		require.False(t, importResp.State.GetAttribute(ctx, path.Root("key"), &key).HasError())
		require.False(t, importResp.State.GetAttribute(ctx, path.Root("value"), &val).HasError())
		assert.Equal(t, "shared", ns.ValueString())
		assert.Equal(t, "ves.io/region", key.ValueString())
		assert.Equal(t, "ves-io-par", val.ValueString())
	})
}
