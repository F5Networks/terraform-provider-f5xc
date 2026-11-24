//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package hook

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKnownLabelOverrideImportHook(t *testing.T) {
	testSchema := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"namespace": schema.StringAttribute{Optional: true},
			"key":       schema.StringAttribute{Optional: true},
			"value":     schema.StringAttribute{Optional: true},
		},
	}

	t.Run("simple-key-and-value", func(t *testing.T) {
		ctx := context.Background()
		resp := &resource.ImportStateResponse{State: newEmptyStateForSchema(testSchema)}
		KnownLabelOverrideImportHook(ctx, resource.ImportStateRequest{ID: "shared/env=prod"}, resp)
		require.False(t, resp.Diagnostics.HasError())

		var ns, key, val types.String
		mustGetAttribute(t, ctx, resp.State, path.Root("namespace"), &ns)
		mustGetAttribute(t, ctx, resp.State, path.Root("key"), &key)
		mustGetAttribute(t, ctx, resp.State, path.Root("value"), &val)
		assert.Equal(t, "shared", ns.ValueString())
		assert.Equal(t, "env", key.ValueString())
		assert.Equal(t, "prod", val.ValueString())
	})

	t.Run("slash-in-label-key", func(t *testing.T) {
		ctx := context.Background()
		resp := &resource.ImportStateResponse{State: newEmptyStateForSchema(testSchema)}
		KnownLabelOverrideImportHook(ctx, resource.ImportStateRequest{ID: "shared/ves.io/region=ves-io-par"}, resp)
		require.False(t, resp.Diagnostics.HasError())

		var ns, key, val types.String
		mustGetAttribute(t, ctx, resp.State, path.Root("namespace"), &ns)
		mustGetAttribute(t, ctx, resp.State, path.Root("key"), &key)
		mustGetAttribute(t, ctx, resp.State, path.Root("value"), &val)
		assert.Equal(t, "shared", ns.ValueString())
		assert.Equal(t, "ves.io/region", key.ValueString())
		assert.Equal(t, "ves-io-par", val.ValueString())
	})

	t.Run("empty-value-leaves-state-value-null", func(t *testing.T) {
		ctx := context.Background()
		resp := &resource.ImportStateResponse{State: newEmptyStateForSchema(testSchema)}
		KnownLabelOverrideImportHook(ctx, resource.ImportStateRequest{ID: "shared/my-key="}, resp)
		require.False(t, resp.Diagnostics.HasError())

		var ns, key, val types.String
		mustGetAttribute(t, ctx, resp.State, path.Root("namespace"), &ns)
		mustGetAttribute(t, ctx, resp.State, path.Root("key"), &key)
		mustGetAttribute(t, ctx, resp.State, path.Root("value"), &val)
		assert.Equal(t, "shared", ns.ValueString())
		assert.Equal(t, "my-key", key.ValueString())
		assert.True(t, val.IsNull())
	})

	for _, invalidID := range []string{
		"",
		"shared",
		"shared/env",
		"/env=prod",
		"shared/=prod",
	} {
		t.Run("invalid-"+invalidID, func(t *testing.T) {
			ctx := context.Background()
			resp := &resource.ImportStateResponse{State: newEmptyStateForSchema(testSchema)}
			KnownLabelOverrideImportHook(ctx, resource.ImportStateRequest{ID: invalidID}, resp)
			assert.True(t, resp.Diagnostics.HasError())
		})
	}
}
