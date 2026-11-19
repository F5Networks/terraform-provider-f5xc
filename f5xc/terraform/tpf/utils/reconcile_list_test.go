//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package utils_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/utils"
)

func TestReconcileListForApply(t *testing.T) {
	ctx := context.Background()

	t.Run("string-list-permutation-preserves-plan-order", func(t *testing.T) {
		planned, diags := types.ListValue(types.StringType, []attr.Value{
			types.StringValue("96.86.173.186/32"),
			types.StringValue("104.129.180.10/24"),
			types.StringValue("192.168.1.0/24"),
		})
		require.False(t, diags.HasError())

		api, diags := types.ListValue(types.StringType, []attr.Value{
			types.StringValue("104.129.180.10/24"),
			types.StringValue("192.168.1.0/24"),
			types.StringValue("96.86.173.186/32"),
		})
		require.False(t, diags.HasError())

		reconciled, rDiags := utils.ReconcileListForApply(ctx, planned, api)
		require.False(t, rDiags.HasError())
		assert.True(t, planned.Equal(reconciled), "reconciled list should exactly match planned list on pure permutation")
		assert.Equal(t, planned.Elements(), reconciled.Elements())
	})

	t.Run("int64-list-permutation-preserves-plan-order", func(t *testing.T) {
		planned, diags := types.ListValue(types.Int64Type, []attr.Value{
			types.Int64Value(65002),
			types.Int64Value(65001),
			types.Int64Value(65003),
		})
		require.False(t, diags.HasError())

		api, diags := types.ListValue(types.Int64Type, []attr.Value{
			types.Int64Value(65001),
			types.Int64Value(65002),
			types.Int64Value(65003),
		})
		require.False(t, diags.HasError())

		reconciled, rDiags := utils.ReconcileListForApply(ctx, planned, api)
		require.False(t, rDiags.HasError())
		assert.True(t, planned.Equal(reconciled))
		assert.Equal(t, planned.Elements(), reconciled.Elements())
	})

	t.Run("int32-list-permutation-preserves-plan-order", func(t *testing.T) {
		planned, diags := types.ListValue(types.Int32Type, []attr.Value{
			types.Int32Value(8080),
			types.Int32Value(443),
			types.Int32Value(80),
		})
		require.False(t, diags.HasError())

		api, diags := types.ListValue(types.Int32Type, []attr.Value{
			types.Int32Value(80),
			types.Int32Value(443),
			types.Int32Value(8080),
		})
		require.False(t, diags.HasError())

		reconciled, rDiags := utils.ReconcileListForApply(ctx, planned, api)
		require.False(t, rDiags.HasError())
		assert.True(t, planned.Equal(reconciled))
		assert.Equal(t, planned.Elements(), reconciled.Elements())
	})

	t.Run("out-of-band-addition-preserves-planned-and-appends-new", func(t *testing.T) {
		planned, diags := types.ListValue(types.StringType, []attr.Value{
			types.StringValue("10.0.0.0/24"),
			types.StringValue("192.168.1.0/24"),
		})
		require.False(t, diags.HasError())

		// API returns elements in sorted order plus an extra item added out-of-band
		api, diags := types.ListValue(types.StringType, []attr.Value{
			types.StringValue("10.0.0.0/24"),
			types.StringValue("172.16.0.0/16"),
			types.StringValue("192.168.1.0/24"),
		})
		require.False(t, diags.HasError())

		reconciled, rDiags := utils.ReconcileListForApply(ctx, planned, api)
		require.False(t, rDiags.HasError())

		expected, diags := types.ListValue(types.StringType, []attr.Value{
			types.StringValue("10.0.0.0/24"),
			types.StringValue("192.168.1.0/24"),
			types.StringValue("172.16.0.0/16"),
		})
		require.False(t, diags.HasError())
		assert.Equal(t, expected.Elements(), reconciled.Elements())
	})

	t.Run("out-of-band-deletion-preserves-remaining-planned-order", func(t *testing.T) {
		planned, diags := types.ListValue(types.StringType, []attr.Value{
			types.StringValue("10.0.0.0/24"),
			types.StringValue("172.16.0.0/16"),
			types.StringValue("192.168.1.0/24"),
		})
		require.False(t, diags.HasError())

		// 172.16.0.0/16 was removed out-of-band
		api, diags := types.ListValue(types.StringType, []attr.Value{
			types.StringValue("192.168.1.0/24"),
			types.StringValue("10.0.0.0/24"),
		})
		require.False(t, diags.HasError())

		reconciled, rDiags := utils.ReconcileListForApply(ctx, planned, api)
		require.False(t, rDiags.HasError())

		expected, diags := types.ListValue(types.StringType, []attr.Value{
			types.StringValue("10.0.0.0/24"),
			types.StringValue("192.168.1.0/24"),
		})
		require.False(t, diags.HasError())
		assert.Equal(t, expected.Elements(), reconciled.Elements())
	})

	t.Run("both-addition-and-deletion", func(t *testing.T) {
		planned, diags := types.ListValue(types.StringType, []attr.Value{
			types.StringValue("A"),
			types.StringValue("B"),
			types.StringValue("C"),
		})
		require.False(t, diags.HasError())

		// B removed, D added, remaining A and C returned in reversed order
		api, diags := types.ListValue(types.StringType, []attr.Value{
			types.StringValue("D"),
			types.StringValue("C"),
			types.StringValue("A"),
		})
		require.False(t, diags.HasError())

		reconciled, rDiags := utils.ReconcileListForApply(ctx, planned, api)
		require.False(t, rDiags.HasError())

		expected, diags := types.ListValue(types.StringType, []attr.Value{
			types.StringValue("A"),
			types.StringValue("C"),
			types.StringValue("D"),
		})
		require.False(t, diags.HasError())
		assert.Equal(t, expected.Elements(), reconciled.Elements())
	})

	t.Run("null-and-unknown-handling", func(t *testing.T) {
		api, _ := types.ListValue(types.StringType, []attr.Value{types.StringValue("A")})

		// Planned is null -> return API
		r1, d1 := utils.ReconcileListForApply(ctx, types.ListNull(types.StringType), api)
		require.False(t, d1.HasError())
		assert.Equal(t, api, r1)

		// Planned is unknown -> return API
		r2, d2 := utils.ReconcileListForApply(ctx, types.ListUnknown(types.StringType), api)
		require.False(t, d2.HasError())
		assert.Equal(t, api, r2)

		// API is null -> return planned
		planned, _ := types.ListValue(types.StringType, []attr.Value{types.StringValue("A")})
		r3, d3 := utils.ReconcileListForApply(ctx, planned, types.ListNull(types.StringType))
		require.False(t, d3.HasError())
		assert.Equal(t, planned, r3)

		// API is unknown -> return planned
		r4, d4 := utils.ReconcileListForApply(ctx, planned, types.ListUnknown(types.StringType))
		require.False(t, d4.HasError())
		assert.Equal(t, planned, r4)
	})

	t.Run("empty-lists", func(t *testing.T) {
		emptyPlanned, _ := types.ListValue(types.StringType, []attr.Value{})
		emptyAPI, _ := types.ListValue(types.StringType, []attr.Value{})

		reconciled, diags := utils.ReconcileListForApply(ctx, emptyPlanned, emptyAPI)
		require.False(t, diags.HasError())
		assert.Equal(t, emptyPlanned, reconciled)
	})

	t.Run("ipv6-prefix-canonicalization-preserves-planned-spelling", func(t *testing.T) {
		planned, diags := types.ListValue(types.StringType, []attr.Value{
			types.StringValue("2001:db8:1:0:0:0:0:0/64"),
		})
		require.False(t, diags.HasError())

		api, diags := types.ListValue(types.StringType, []attr.Value{
			types.StringValue("2001:db8:1::/64"),
		})
		require.False(t, diags.HasError())

		reconciled, rDiags := utils.ReconcileListForApply(ctx, planned, api)
		require.False(t, rDiags.HasError())
		assert.True(t, planned.Equal(reconciled), "reconciled list should preserve non-canonical planned IPv6 prefix spelling")
		assert.Equal(t, planned.Elements(), reconciled.Elements())
	})

	t.Run("ipv6-prefix-canonicalization-with-addition-preserves-planned-spelling", func(t *testing.T) {
		planned, diags := types.ListValue(types.StringType, []attr.Value{
			types.StringValue("2001:db8:1:0:0:0:0:0/64"),
		})
		require.False(t, diags.HasError())

		api, diags := types.ListValue(types.StringType, []attr.Value{
			types.StringValue("2001:db8:1::/64"),
			types.StringValue("2001:db8:2::/64"),
		})
		require.False(t, diags.HasError())

		reconciled, rDiags := utils.ReconcileListForApply(ctx, planned, api)
		require.False(t, rDiags.HasError())

		expected, diags := types.ListValue(types.StringType, []attr.Value{
			types.StringValue("2001:db8:1:0:0:0:0:0/64"),
			types.StringValue("2001:db8:2::/64"),
		})
		require.False(t, diags.HasError())
		assert.Equal(t, expected.Elements(), reconciled.Elements())
	})
}

func TestReconcileListForRead(t *testing.T) {
	ctx := context.Background()

	t.Run("pure-permutation-preserves-prior-state-order", func(t *testing.T) {
		prior, diags := types.ListValue(types.StringType, []attr.Value{
			types.StringValue("96.86.173.186/32"),
			types.StringValue("104.129.180.10/24"),
		})
		require.False(t, diags.HasError())

		api, diags := types.ListValue(types.StringType, []attr.Value{
			types.StringValue("104.129.180.10/24"),
			types.StringValue("96.86.173.186/32"),
		})
		require.False(t, diags.HasError())

		reconciled, rDiags := utils.ReconcileListForRead(ctx, prior, api)
		require.False(t, rDiags.HasError())
		assert.True(t, prior.Equal(reconciled))
		assert.Equal(t, prior.Elements(), reconciled.Elements())
	})

	t.Run("drift-with-additions-appends-at-end", func(t *testing.T) {
		prior, _ := types.ListValue(types.StringType, []attr.Value{
			types.StringValue("B"),
			types.StringValue("A"),
		})
		api, _ := types.ListValue(types.StringType, []attr.Value{
			types.StringValue("A"),
			types.StringValue("B"),
			types.StringValue("C"),
		})

		reconciled, rDiags := utils.ReconcileListForRead(ctx, prior, api)
		require.False(t, rDiags.HasError())

		expected, _ := types.ListValue(types.StringType, []attr.Value{
			types.StringValue("B"),
			types.StringValue("A"),
			types.StringValue("C"),
		})
		assert.Equal(t, expected.Elements(), reconciled.Elements())
	})

	t.Run("null-and-unknown-handling", func(t *testing.T) {
		api, _ := types.ListValue(types.StringType, []attr.Value{types.StringValue("A")})

		r1, d1 := utils.ReconcileListForRead(ctx, types.ListNull(types.StringType), api)
		require.False(t, d1.HasError())
		assert.Equal(t, api, r1)

		r2, d2 := utils.ReconcileListForRead(ctx, types.ListUnknown(types.StringType), api)
		require.False(t, d2.HasError())
		assert.Equal(t, api, r2)
	})
}
