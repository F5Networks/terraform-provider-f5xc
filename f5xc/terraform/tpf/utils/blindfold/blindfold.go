//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

// Package blindfold implements the F5XC Blindfold client-side secret bundle
// format. It matches gopkg.volterra.us/stdlib/blindfoldutil.EncryptSecret:
// AES-256-GCM data encryption, then a blinded RSA power C = M^(e*tau) mod N
// where tau = 2^(k-1) + 2*policy_id + 1, wrapped in a fixed byte layout.
package blindfold

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// PublicKeyFetcher and PolicyFetcher operate on the AuthClient exposed by the
// provider. The interface abstracts the concrete client so the engine can be
// unit tested with a fake.
type restExecutor interface {
	Execute(ctx context.Context, method, url string, body []byte) ([]byte, error)
}

// Client bundles the REST executor and base URL used to reach the F5XC public
// APIs.
type Client struct {
	Exec    restExecutor
	BaseURL string
}

// KeyData mirrors ves.io.schema.secret_management.KeyData.
type KeyData struct {
	Tenant               string `json:"tenant"`
	KeyVersion           uint32 `json:"key_version"`
	ModulusBase64        string `json:"modulus_base64"`
	PublicExponentBase64 string `json:"public_exponent_base64"`
}

// PolicyData mirrors ves.io.schema.secret_management.PolicyData.
type PolicyData struct {
	Tenant    string      `json:"tenant"`
	Namespace string      `json:"namespace"`
	Name      string      `json:"name"`
	PolicyID  policyIDU64 `json:"policy_id"`
}

// policyIDU64 accepts both JSON number and JSON string encodings of a uint64.
// The F5XC gRPC-gateway serializes proto uint64 fields as JSON strings.
type policyIDU64 uint64

func (p *policyIDU64) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	s := strings.Trim(string(b), "\"")
	if s == "" {
		return nil
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return fmt.Errorf("policy_id: %w", err)
	}
	*p = policyIDU64(v)
	return nil
}

type getPublicKeyResponse struct {
	Data *KeyData `json:"data"`
}

type getPolicyDocumentResponse struct {
	Data *PolicyData `json:"data"`
}

// GetPublicKey fetches the tenant public RSA key from
// GET /secret_management/get_public_key.
func (c *Client) GetPublicKey(ctx context.Context) (*KeyData, error) {
	base := strings.TrimSuffix(c.BaseURL, "/")
	url := base + "/secret_management/get_public_key"

	tflog.Info(ctx, "Blindfold: fetching tenant public key", map[string]any{"url": url})

	body, err := c.Exec.Execute(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("blindfold: get_public_key request failed: %w", err)
	}

	var resp getPublicKeyResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("blindfold: decoding get_public_key response (body=%q): %w", string(body), err)
	}
	// Fallback: some grpc-gateway responses omit the "data" wrapper.
	if resp.Data == nil {
		var flat KeyData
		if err := json.Unmarshal(body, &flat); err == nil && (flat.ModulusBase64 != "" || flat.PublicExponentBase64 != "") {
			resp.Data = &flat
		}
	}
	if resp.Data == nil || resp.Data.ModulusBase64 == "" || resp.Data.PublicExponentBase64 == "" {
		return nil, fmt.Errorf("blindfold: get_public_key returned empty key material (body=%q)", string(body))
	}
	tflog.Debug(ctx, "Blindfold: received tenant public key", map[string]any{
		"modulus_len":  len(resp.Data.ModulusBase64),
		"exponent_b64": resp.Data.PublicExponentBase64,
		"key_version":  resp.Data.KeyVersion,
	})
	return resp.Data, nil
}

// GetPolicyDocument fetches the referenced secret policy from
// GET /secret_management/namespaces/{namespace}/secret_policys/{name}/get_policy_document.
func (c *Client) GetPolicyDocument(ctx context.Context, namespace, name string) (*PolicyData, error) {
	if namespace == "" {
		return nil, fmt.Errorf("blindfold: policy namespace is required")
	}
	if name == "" {
		return nil, fmt.Errorf("blindfold: policy name is required")
	}
	base := strings.TrimSuffix(c.BaseURL, "/")
	url := fmt.Sprintf("%s/secret_management/namespaces/%s/secret_policys/%s/get_policy_document", base, namespace, name)

	tflog.Info(ctx, "Blindfold: fetching secret policy document", map[string]any{"url": url})

	body, err := c.Exec.Execute(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("blindfold: get_policy_document request failed: %w", err)
	}

	var resp getPolicyDocumentResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("blindfold: decoding get_policy_document response: %w", err)
	}
	if resp.Data == nil || resp.Data.PolicyID == 0 {
		return nil, fmt.Errorf("blindfold: get_policy_document returned empty policy data")
	}
	return resp.Data, nil
}

// decodeBase64Flexible accepts both standard and URL-safe base64, with or
// without padding.
func decodeBase64Flexible(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty")
	}
	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}
	var lastErr error
	for _, enc := range encodings {
		b, err := enc.DecodeString(s)
		if err == nil {
			return b, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// Blindfold constants aligned with gopkg.volterra.us/stdlib/blindfoldutil.
const (
	blindfoldK            = 32
	blindfoldMagicNumber  = uint32(0xdeadbeef)
	blindfoldMagicSize    = 4
	blindfoldAlgoAES256   = uint8(2)
	blindfoldKeyLenAES256 = 44 // 12-byte AES-GCM nonce + 32-byte AES-256 key
)

// Blindfold produces a F5XC secret bundle (base64 payload for string:///) that
// wingman can decrypt. Layout matches blindfoldutil.EncryptSecret exactly:
//
//	[u16 version][u16 tenant_len][tenant][u32 key_version][u64 policy_id][u8 algo]
//	[u32 exp_len][exp][u32 mod_len][mod][u32 enc_key_len][enc_key][enc_msg]
//
// enc_key = M^(e*tau) mod N where
// M = magic(4) || nonce(12) || aes_key(32) || random padding to modLen-2
// tau = 2^(k-1) + 2*policy_id + 1  (k=32)
func Blindfold(plaintext []byte, key *KeyData, policy *PolicyData) (string, error) {
	if len(plaintext) == 0 {
		return "", fmt.Errorf("blindfold: plaintext is empty")
	}
	if key == nil {
		return "", fmt.Errorf("blindfold: key material is nil")
	}
	if policy == nil {
		return "", fmt.Errorf("blindfold: policy data is nil")
	}

	modBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(key.ModulusBase64))
	if err != nil || len(modBytes) == 0 {
		return "", fmt.Errorf("blindfold: decode modulus (b64_head=%q): %w", head(key.ModulusBase64, 12), err)
	}
	expBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(key.PublicExponentBase64))
	if err != nil || len(expBytes) == 0 {
		return "", fmt.Errorf("blindfold: decode exponent (b64=%q): %w", key.PublicExponentBase64, err)
	}
	modLen := len(modBytes)
	if modLen < blindfoldMagicSize+blindfoldKeyLenAES256+2 {
		return "", fmt.Errorf("blindfold: modulus too small (%d bytes)", modLen)
	}

	// 1. AES-256-GCM encrypt plaintext.
	aesKey := make([]byte, 32)
	if _, err := rand.Read(aesKey); err != nil {
		return "", fmt.Errorf("blindfold: aes key gen: %w", err)
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", fmt.Errorf("blindfold: aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("blindfold: gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("blindfold: nonce gen: %w", err)
	}
	encMsg := gcm.Seal(nil, nonce, plaintext, nil)

	// 2. Build padded key blob = magic(4) || nonce(12) || aes_key(32) || random.
	paddedKey := make([]byte, modLen-2)
	binary.BigEndian.PutUint32(paddedKey[:blindfoldMagicSize], blindfoldMagicNumber)
	copy(paddedKey[blindfoldMagicSize:blindfoldMagicSize+12], nonce)
	copy(paddedKey[blindfoldMagicSize+12:blindfoldMagicSize+12+32], aesKey)
	if _, err := rand.Read(paddedKey[blindfoldMagicSize+blindfoldKeyLenAES256:]); err != nil {
		return "", fmt.Errorf("blindfold: padding: %w", err)
	}

	// 3. C = M^(e*tau) mod N,  tau = 2^(k-1) + 2*policy_id + 1.
	N := new(big.Int).SetBytes(modBytes)
	e := new(big.Int).SetBytes(expBytes)
	M := new(big.Int).SetBytes(paddedKey)
	tau := new(big.Int).Lsh(big.NewInt(1), blindfoldK-1)
	tau.Add(tau, new(big.Int).Mul(big.NewInt(2), new(big.Int).SetUint64(uint64(policy.PolicyID))))
	tau.Add(tau, big.NewInt(1))
	etau := new(big.Int).Mul(e, tau)
	C := new(big.Int).Exp(M, etau, N)
	encKey := C.Bytes()

	// 4. Assemble bundle.
	// The bundle Tenant field must match the tenant that owns the RSA public
	// key used for encryption. wingman looks up its private key using this
	// tenant on decrypt. Using policy.Tenant (which is often "ves-io" for
	// shared policies like ves-io-allow-config) makes wingman select the wrong
	// private key and reject the secret as "Invalid blindfold secret".
	bundleTenant := key.Tenant
	if bundleTenant == "" {
		bundleTenant = policy.Tenant
	}
	buf := make([]byte, 0, 32+len(bundleTenant)+len(expBytes)+len(modBytes)+len(encKey)+len(encMsg))
	var b2 [2]byte
	var b4 [4]byte
	var b8 [8]byte
	binary.BigEndian.PutUint16(b2[:], 0) // version
	buf = append(buf, b2[:]...)
	binary.BigEndian.PutUint16(b2[:], uint16(len(bundleTenant)))
	buf = append(buf, b2[:]...)
	buf = append(buf, []byte(bundleTenant)...)
	binary.BigEndian.PutUint32(b4[:], key.KeyVersion)
	buf = append(buf, b4[:]...)
	binary.BigEndian.PutUint64(b8[:], uint64(policy.PolicyID))
	buf = append(buf, b8[:]...)
	buf = append(buf, blindfoldAlgoAES256)
	binary.BigEndian.PutUint32(b4[:], uint32(len(expBytes)))
	buf = append(buf, b4[:]...)
	buf = append(buf, expBytes...)
	binary.BigEndian.PutUint32(b4[:], uint32(len(modBytes)))
	buf = append(buf, b4[:]...)
	buf = append(buf, modBytes...)
	binary.BigEndian.PutUint32(b4[:], uint32(len(encKey)))
	buf = append(buf, b4[:]...)
	buf = append(buf, encKey...)
	buf = append(buf, encMsg...)

	// Scrub AES key.
	for i := range aesKey {
		aesKey[i] = 0
	}

	out := base64.StdEncoding.EncodeToString(buf)
	tflog.Debug(context.Background(), "Blindfold bundle assembled", map[string]any{
		"bundle_len":  len(buf),
		"tenant":      bundleTenant,
		"key_version": key.KeyVersion,
		"policy_id":   uint64(policy.PolicyID),
		"algo":        blindfoldAlgoAES256,
		"exp_len":     len(expBytes),
		"mod_len":     len(modBytes),
		"enc_key_len": len(encKey),
		"enc_msg_len": len(encMsg),
	})
	return "string:///" + out, nil
}

func head(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
