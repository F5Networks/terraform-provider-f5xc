//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package data

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
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

func newKnownLabelDataTestClient(t *testing.T, handler http.HandlerFunc) *client.APIClient {
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

func knownLabelDataEmptyValue(t *testing.T, ctx context.Context, s schema.Schema) tftypes.Value {
	t.Helper()
	objType, ok := s.Type().TerraformType(ctx).(tftypes.Object)
	require.True(t, ok)
	vals := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for k, attrType := range objType.AttributeTypes {
		vals[k] = tftypes.NewValue(attrType, nil)
	}
	return tftypes.NewValue(objType, vals)
}

func knownLabelDataTestTimeouts() timeouts.Value {
	return timeouts.Value{
		Object: types.ObjectNull(map[string]attr.Type{}),
	}
}

func writeKnownLabelDataProtoJSON(t *testing.T, w http.ResponseWriter, msg proto.Message) {
	t.Helper()
	out, err := codec.ToJSON(msg, codec.ToWithUseProtoFieldName())
	require.NoError(t, err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(out))
	require.NoError(t, err)
}

// TestDataSourceF5XCKnownLabel_CompositeKeyAndFallbackRead verifies that
// DataSourceF5XCKnownLabel filters strictly by (key, value) when value is configured
// while preserving key-only compatibility when value is omitted.
func TestDataSourceF5XCKnownLabel_CompositeKeyAndFallbackRead(t *testing.T) {
	ctx := context.Background()

	labelsResp := &ves_io_schema_known_label.GetResponse{
		Label: []*ves_io_schema_known_label.LabelType{
			{
				Key:         "site",
				Value:       "aws-1",
				Description: "AWS",
			},
			{
				Key:         "site",
				Value:       "azure-2",
				Description: "Azure",
			},
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeKnownLabelDataProtoJSON(t, w, labelsResp)
	})

	d := &DataSourceF5XCKnownLabel{client: newKnownLabelDataTestClient(t, handler)}
	var schemaResp datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())

	runRead := func(t *testing.T, cfgModel DataSourceF5XCKnownLabelModel) (DataSourceF5XCKnownLabelModel, datasource.ReadResponse) {
		t.Helper()
		tmpState := tfsdk.State{Schema: schemaResp.Schema, Raw: knownLabelDataEmptyValue(t, ctx, schemaResp.Schema)}
		diags := tmpState.Set(ctx, &cfgModel)
		require.False(t, diags.HasError(), "tmpState.Set diagnostics: %v", diags)
		cfg := tfsdk.Config{Schema: schemaResp.Schema, Raw: tmpState.Raw}
		readResp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: knownLabelDataEmptyValue(t, ctx, schemaResp.Schema)}}
		d.Read(ctx, datasource.ReadRequest{Config: cfg}, &readResp)
		var out DataSourceF5XCKnownLabelModel
		if !readResp.Diagnostics.HasError() {
			require.False(t, readResp.State.Get(ctx, &out).HasError())
		}
		return out, readResp
	}

	t.Run("exact-composite-key-and-value", func(t *testing.T) {
		out, resp := runRead(t, DataSourceF5XCKnownLabelModel{
			Namespace: types.StringValue("shared"),
			Key:       types.StringValue("site"),
			Value:     types.StringValue("azure-2"),
			Timeouts:  knownLabelDataTestTimeouts(),
		})
		require.False(t, resp.Diagnostics.HasError(), "Read diagnostics: %v", resp.Diagnostics)
		assert.Equal(t, "azure-2", out.Value.ValueString())
		assert.Equal(t, "Azure", out.Description.ValueString())
	})

	t.Run("key-only-when-value-null-preserves-compatibility", func(t *testing.T) {
		out, resp := runRead(t, DataSourceF5XCKnownLabelModel{
			Namespace: types.StringValue("shared"),
			Key:       types.StringValue("site"),
			Value:     types.StringNull(),
			Timeouts:  knownLabelDataTestTimeouts(),
		})
		require.False(t, resp.Diagnostics.HasError(), "Read diagnostics: %v", resp.Diagnostics)
		assert.Equal(t, "aws-1", out.Value.ValueString())
		assert.Equal(t, "AWS", out.Description.ValueString())
	})

	t.Run("composite-mismatch-returns-not-found", func(t *testing.T) {
		_, resp := runRead(t, DataSourceF5XCKnownLabelModel{
			Namespace: types.StringValue("shared"),
			Key:       types.StringValue("site"),
			Value:     types.StringValue("gcp-3"),
			Timeouts:  knownLabelDataTestTimeouts(),
		})
		assert.True(t, resp.Diagnostics.HasError())
	})
}
