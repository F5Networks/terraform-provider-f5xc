//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package utils

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- list reconciliation ---

// ReconcileListForRead keeps prior state when the API list is null.
func TestReconcileListForRead_NullAPI(t *testing.T) {
	prior := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("a")})
	got, diags := ReconcileListForRead(context.Background(), prior, types.ListNull(types.StringType))
	require.False(t, diags.HasError())
	assert.True(t, got.Equal(prior))
}

// A prefix never equals a bare address, and equivalent addresses match.
func TestIPStringsOrPrefixesEqual_Addresses(t *testing.T) {
	assert.False(t, ipStringsOrPrefixesEqual("10.0.0.0/8", "10.0.0.1"))
	assert.True(t, ipStringsOrPrefixesEqual("2001:db8::1", "2001:DB8:0:0::1"))
	assert.False(t, ipStringsOrPrefixesEqual("10.0.0.1", "not-an-ip"))
}

// Null base elements are skipped and do not appear in the reconciled list.
func TestReconcileOrderGeneric_SkipsNullBaseElements(t *testing.T) {
	prior := types.ListValueMust(types.StringType, []attr.Value{types.StringNull(), types.StringValue("a")})
	api := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("b")})
	got, diags := ReconcileListForRead(context.Background(), prior, api)
	require.False(t, diags.HasError())
	assert.True(t, got.Equal(api))
	assert.False(t, strings.Contains(got.String(), "null"))
}
