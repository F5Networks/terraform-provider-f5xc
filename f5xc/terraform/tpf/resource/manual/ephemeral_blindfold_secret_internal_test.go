//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package manual

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// blindfoldFixture serves the public-key and policy-document endpoints with
// a key generated at test time.
type blindfoldFixture struct {
	keyStatus    int
	keyBody      string
	policyStatus int
	policyBody   string
}

func (f *blindfoldFixture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	status, body := f.policyStatus, f.policyBody
	if strings.HasSuffix(r.URL.Path, "/get_public_key") {
		status, body = f.keyStatus, f.keyBody
	}
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

// publicKeyBody returns a get_public_key response for a modulus of bits.
func publicKeyBody(t *testing.T, bits int) string {
	t.Helper()
	var n *big.Int
	if bits >= 1024 {
		priv, err := rsa.GenerateKey(rand.Reader, bits)
		require.NoError(t, err)
		n = priv.N
	} else {
		n = new(big.Int).Lsh(big.NewInt(1), uint(bits-1))
	}
	mod := base64.StdEncoding.EncodeToString(n.Bytes())
	exp := base64.StdEncoding.EncodeToString(big.NewInt(65537).Bytes())
	return `{"data":{"tenant":"acme","key_version":7,"modulus_base64":"` + mod + `","public_exponent_base64":"` + exp + `"}}`
}

const policyBody = `{"data":{"tenant":"acme","namespace":"shared","name":"p1","policy_id":"101"}}`

func openBlindfold(t *testing.T, fixture *blindfoldFixture, keyVersion *int64) ephemeral.OpenResponse {
	t.Helper()
	ctx := context.Background()
	srv := httptest.NewServer(fixture)
	t.Cleanup(srv.Close)

	res := &EphemeralF5XCBlindfoldSecret{client: newMockAPIClient(srv)}
	var schemaResp ephemeral.SchemaResponse
	res.Schema(ctx, ephemeral.SchemaRequest{}, &schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())

	var kv any
	if keyVersion != nil {
		kv = *keyVersion
	}
	tfType := schemaResp.Schema.Type().TerraformType(ctx)
	cfg := tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(tfType, map[string]tftypes.Value{
		"namespace":   tftypes.NewValue(tftypes.String, "shared"),
		"policy_name": tftypes.NewValue(tftypes.String, "p1"),
		"content":     tftypes.NewValue(tftypes.String, "plaintext-value"),
		"key_version": tftypes.NewValue(tftypes.Number, kv),
		"location":    tftypes.NewValue(tftypes.String, nil),
		"policy_id":   tftypes.NewValue(tftypes.Number, nil),
		"tenant":      tftypes.NewValue(tftypes.String, nil),
	})}

	resp := ephemeral.OpenResponse{Result: tfsdk.EphemeralResultData{Schema: schemaResp.Schema, Raw: tftypes.NewValue(tfType, nil)}}
	res.Open(ctx, ephemeral.OpenRequest{Config: cfg}, &resp)
	return resp
}

// TestEphemeralBlindfoldSecret_Open verifies that Open fills location,
// policy_id, tenant, and key_version from the API, keeps a configured
// key_version, and reports each failing step with its own diagnostic.
func TestEphemeralBlindfoldSecret_Open(t *testing.T) {
	key := publicKeyBody(t, 2048)

	t.Run("success uses server key version", func(t *testing.T) {
		resp := openBlindfold(t, &blindfoldFixture{keyBody: key, policyBody: policyBody}, nil)
		require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

		var got blindfoldSecretModel
		require.False(t, resp.Result.Get(context.Background(), &got).HasError())
		assert.True(t, strings.HasPrefix(got.Location.ValueString(), "string:///"), got.Location.ValueString())
		assert.Equal(t, int64(101), got.PolicyID.ValueInt64())
		assert.Equal(t, "acme", got.Tenant.ValueString())
		assert.Equal(t, int64(7), got.KeyVersion.ValueInt64())
	})

	t.Run("success keeps configured key version", func(t *testing.T) {
		kv := int64(3)
		resp := openBlindfold(t, &blindfoldFixture{keyBody: key, policyBody: policyBody}, &kv)
		require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

		var got blindfoldSecretModel
		require.False(t, resp.Result.Get(context.Background(), &got).HasError())
		assert.Equal(t, int64(3), got.KeyVersion.ValueInt64())
	})

	t.Run("public key failure", func(t *testing.T) {
		resp := openBlindfold(t, &blindfoldFixture{keyStatus: http.StatusInternalServerError}, nil)
		assert.Equal(t, "Blindfold: fetch public key failed", resp.Diagnostics.Errors()[0].Summary())
	})

	t.Run("policy document failure", func(t *testing.T) {
		resp := openBlindfold(t, &blindfoldFixture{keyBody: key, policyStatus: http.StatusForbidden}, nil)
		assert.Equal(t, "Blindfold: fetch policy document failed", resp.Diagnostics.Errors()[0].Summary())
	})

	t.Run("encryption failure", func(t *testing.T) {
		resp := openBlindfold(t, &blindfoldFixture{keyBody: publicKeyBody(t, 64), policyBody: policyBody}, nil)
		assert.Equal(t, "Blindfold: envelope encryption failed", resp.Diagnostics.Errors()[0].Summary())
	})
}

// TestEphemeralBlindfoldSecret_Open_InvalidConfig verifies that an
// undecodable config stops Open with an error diagnostic.
func TestEphemeralBlindfoldSecret_Open_InvalidConfig(t *testing.T) {
	ctx := context.Background()
	res := &EphemeralF5XCBlindfoldSecret{}
	var schemaResp ephemeral.SchemaResponse
	res.Schema(ctx, ephemeral.SchemaRequest{}, &schemaResp)

	var resp ephemeral.OpenResponse
	res.Open(ctx, ephemeral.OpenRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: invalidRaw}}, &resp)
	assert.True(t, resp.Diagnostics.HasError())
}
