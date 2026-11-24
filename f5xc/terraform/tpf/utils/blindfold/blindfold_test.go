//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package blindfold

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"testing"
)

// fakeExec is an in-memory restExecutor that returns canned responses per URL.
type fakeExec struct {
	responses map[string][]byte
	err       error
}

func (f *fakeExec) Execute(_ context.Context, method, url string, _ []byte) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	if method != http.MethodGet {
		return nil, fmt.Errorf("unexpected method %s", method)
	}
	body, ok := f.responses[url]
	if !ok {
		return nil, fmt.Errorf("no canned response for %s", url)
	}
	return body, nil
}

func newTestKey(t *testing.T) (*rsa.PrivateKey, *KeyData) {
	t.Helper()
	// For policyID=101 used in TestBlindfold_RoundTrip, tau = 2^63 + 203 is divisible
	// by 3 and 13. Ensure gcd(e*tau, phi(N)) == 1 so the blinded RSA exponent is invertible
	// during round-trip decryption in tests.
	tau := new(big.Int).Lsh(big.NewInt(1), blindfoldK-1)
	tau.Add(tau, new(big.Int).Mul(big.NewInt(2), new(big.Int).SetUint64(101)))
	tau.Add(tau, big.NewInt(1))
	one := big.NewInt(1)

	var priv *rsa.PrivateKey
	for {
		candidate, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("generate rsa key: %v", err)
		}
		if len(candidate.Primes) == 2 {
			phi := new(big.Int).Mul(
				new(big.Int).Sub(candidate.Primes[0], one),
				new(big.Int).Sub(candidate.Primes[1], one),
			)
			etau := new(big.Int).Mul(big.NewInt(int64(candidate.PublicKey.E)), tau)
			if new(big.Int).ModInverse(etau, phi) != nil {
				priv = candidate
				break
			}
		}
	}
	expBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(expBytes, uint32(priv.PublicKey.E))
	// Trim leading zero bytes to match F5XC formatting (variable-length exponent).
	for len(expBytes) > 1 && expBytes[0] == 0 {
		expBytes = expBytes[1:]
	}
	return priv, &KeyData{
		Tenant:               "acme",
		KeyVersion:           7,
		ModulusBase64:        base64.StdEncoding.EncodeToString(priv.PublicKey.N.Bytes()),
		PublicExponentBase64: base64.StdEncoding.EncodeToString(expBytes),
	}
}

func TestGetPublicKey_Success(t *testing.T) {
	base := "https://tenant.example/api"
	_, key := newTestKey(t)
	body, _ := json.Marshal(getPublicKeyResponse{Data: key})

	c := &Client{
		Exec: &fakeExec{responses: map[string][]byte{
			base + "/secret_management/get_public_key": body,
		}},
		BaseURL: base,
	}

	got, err := c.GetPublicKey(context.Background())
	if err != nil {
		t.Fatalf("GetPublicKey: %v", err)
	}
	if got.ModulusBase64 != key.ModulusBase64 || got.KeyVersion != 7 {
		t.Fatalf("unexpected KeyData: %+v", got)
	}
}

func TestGetPublicKey_Failures(t *testing.T) {
	base := "https://tenant.example/api"
	url := base + "/secret_management/get_public_key"

	t.Run("http_error", func(t *testing.T) {
		c := &Client{
			Exec:    &fakeExec{err: fmt.Errorf("boom")},
			BaseURL: base,
		}
		if _, err := c.GetPublicKey(context.Background()); err == nil {
			t.Fatal("expected error for http failure, got nil")
		}
	})
	t.Run("bad_json", func(t *testing.T) {
		c := &Client{
			Exec:    &fakeExec{responses: map[string][]byte{url: []byte("not-json")}},
			BaseURL: base,
		}
		if _, err := c.GetPublicKey(context.Background()); err == nil {
			t.Fatal("expected error for malformed json, got nil")
		}
	})
	t.Run("empty_key", func(t *testing.T) {
		body, _ := json.Marshal(getPublicKeyResponse{})
		c := &Client{
			Exec:    &fakeExec{responses: map[string][]byte{url: body}},
			BaseURL: base,
		}
		if _, err := c.GetPublicKey(context.Background()); err == nil {
			t.Fatal("expected error for empty key data, got nil")
		}
	})
}

func TestGetPolicyDocument_Success(t *testing.T) {
	base := "https://tenant.example/api"
	policy := &PolicyData{Tenant: "acme", Namespace: "shared", Name: "prod", PolicyID: 42}
	body, _ := json.Marshal(getPolicyDocumentResponse{Data: policy})

	c := &Client{
		Exec: &fakeExec{responses: map[string][]byte{
			base + "/secret_management/namespaces/shared/secret_policys/prod/get_policy_document": body,
		}},
		BaseURL: base,
	}

	got, err := c.GetPolicyDocument(context.Background(), "shared", "prod")
	if err != nil {
		t.Fatalf("GetPolicyDocument: %v", err)
	}
	if got.PolicyID != 42 || got.Namespace != "shared" {
		t.Fatalf("unexpected PolicyData: %+v", got)
	}
}

func TestGetPolicyDocument_Failures(t *testing.T) {
	base := "https://tenant.example/api"
	url := base + "/secret_management/namespaces/shared/secret_policys/prod/get_policy_document"

	t.Run("http_error", func(t *testing.T) {
		c := &Client{
			Exec:    &fakeExec{err: fmt.Errorf("boom")},
			BaseURL: base,
		}
		if _, err := c.GetPolicyDocument(context.Background(), "shared", "prod"); err == nil {
			t.Fatal("expected error for http failure, got nil")
		}
	})
	t.Run("bad_json", func(t *testing.T) {
		c := &Client{
			Exec:    &fakeExec{responses: map[string][]byte{url: []byte("not-json")}},
			BaseURL: base,
		}
		if _, err := c.GetPolicyDocument(context.Background(), "shared", "prod"); err == nil {
			t.Fatal("expected error for malformed json, got nil")
		}
	})
	t.Run("empty_policy", func(t *testing.T) {
		body, _ := json.Marshal(getPolicyDocumentResponse{})
		c := &Client{
			Exec:    &fakeExec{responses: map[string][]byte{url: body}},
			BaseURL: base,
		}
		if _, err := c.GetPolicyDocument(context.Background(), "shared", "prod"); err == nil {
			t.Fatal("expected error for empty policy data, got nil")
		}
	})
}

func TestGetPolicyDocument_Empty(t *testing.T) {
	c := &Client{Exec: &fakeExec{}, BaseURL: "https://tenant.example/api"}
	if _, err := c.GetPolicyDocument(context.Background(), "", "x"); err == nil {
		t.Fatal("expected error for empty namespace")
	}
	if _, err := c.GetPolicyDocument(context.Background(), "x", ""); err == nil {
		t.Fatal("expected error for empty name")
	}
}

// TestBlindfold_RoundTrip verifies that a payload encrypted with a generated
// public key can be recovered end-to-end using the corresponding private key.
// It parses the exact envelope layout produced by Blindfold and inverts the
// blinded-RSA wrapping C = M^(e*tau) mod N to recover the AES-256-GCM DEK.
func TestBlindfold_RoundTrip(t *testing.T) {
	priv, key := newTestKey(t)
	policy := &PolicyData{Tenant: "acme", Namespace: "shared", Name: "prod", PolicyID: 101}
	plaintext := []byte("super-secret-tls-private-key-material")

	url, err := Blindfold(plaintext, key, policy)
	if err != nil {
		t.Fatalf("Blindfold: %v", err)
	}
	if !strings.HasPrefix(url, "string:///") {
		t.Fatalf("expected string:/// prefix, got %q", url[:20])
	}
	buf, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(url, "string:///"))
	if err != nil {
		t.Fatalf("decode envelope base64: %v", err)
	}

	env, err := parseBlindfoldEnvelope(buf)
	if err != nil {
		t.Fatalf("parse envelope: %v", err)
	}
	if env.version != 0 {
		t.Fatalf("unexpected version %d", env.version)
	}
	if env.tenant != "acme" {
		t.Fatalf("unexpected tenant %q", env.tenant)
	}
	if env.keyVersion != 7 {
		t.Fatalf("unexpected key_version %d", env.keyVersion)
	}
	if env.policyID != 101 {
		t.Fatalf("unexpected policy_id %d", env.policyID)
	}
	if env.algo != blindfoldAlgoAES256 {
		t.Fatalf("unexpected algo %d", env.algo)
	}

	modLen := len(env.mod)
	if modLen < blindfoldMagicSize+blindfoldKeyLenAES256+2 {
		t.Fatalf("modulus too small: %d", modLen)
	}

	// Invert C = M^(e*tau) mod N to recover M.
	N := priv.PublicKey.N
	e := big.NewInt(int64(priv.PublicKey.E))
	tau := new(big.Int).Lsh(big.NewInt(1), blindfoldK-1)
	tau.Add(tau, new(big.Int).Mul(big.NewInt(2), new(big.Int).SetUint64(env.policyID)))
	tau.Add(tau, big.NewInt(1))
	etau := new(big.Int).Mul(e, tau)

	if len(priv.Primes) != 2 {
		t.Fatalf("expected 2 primes in test key, got %d", len(priv.Primes))
	}
	one := big.NewInt(1)
	phi := new(big.Int).Mul(new(big.Int).Sub(priv.Primes[0], one), new(big.Int).Sub(priv.Primes[1], one))
	invEtau := new(big.Int).ModInverse(etau, phi)
	if invEtau == nil {
		t.Fatal("e*tau not invertible mod phi(N)")
	}

	C := new(big.Int).SetBytes(env.encKey)
	M := new(big.Int).Exp(C, invEtau, N)

	// big.Int.Bytes strips leading zeros; the padded key blob is modLen-2 bytes.
	mBytes := M.Bytes()
	if pad := (modLen - 2) - len(mBytes); pad > 0 {
		mBytes = append(make([]byte, pad), mBytes...)
	}
	if len(mBytes) < blindfoldMagicSize+blindfoldKeyLenAES256 {
		t.Fatalf("recovered M too short: %d", len(mBytes))
	}

	magic := binary.BigEndian.Uint32(mBytes[:blindfoldMagicSize])
	if magic != blindfoldMagicNumber {
		t.Fatalf("magic mismatch: got %#x want %#x", magic, blindfoldMagicNumber)
	}
	nonce := mBytes[blindfoldMagicSize : blindfoldMagicSize+12]
	aesKey := mBytes[blindfoldMagicSize+12 : blindfoldMagicSize+12+32]

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		t.Fatalf("aes cipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("gcm: %v", err)
	}
	got, err := gcm.Open(nil, nonce, env.encMsg, nil)
	if err != nil {
		t.Fatalf("gcm open: %v", err)
	}
	if string(got) != string(plaintext) {
		t.Fatalf("plaintext mismatch: got %q want %q", got, plaintext)
	}
}

type blindfoldEnvelope struct {
	version    uint16
	tenant     string
	keyVersion uint32
	policyID   uint64
	algo       uint8
	exp        []byte
	mod        []byte
	encKey     []byte
	encMsg     []byte
}

func parseBlindfoldEnvelope(buf []byte) (*blindfoldEnvelope, error) {
	var env blindfoldEnvelope
	r := bytesReader{buf: buf}
	var err error
	if env.version, err = r.u16(); err != nil {
		return nil, fmt.Errorf("version: %w", err)
	}
	tenantLen, err := r.u16()
	if err != nil {
		return nil, fmt.Errorf("tenant_len: %w", err)
	}
	tenant, err := r.bytes(int(tenantLen))
	if err != nil {
		return nil, fmt.Errorf("tenant: %w", err)
	}
	env.tenant = string(tenant)
	if env.keyVersion, err = r.u32(); err != nil {
		return nil, fmt.Errorf("key_version: %w", err)
	}
	if env.policyID, err = r.u64(); err != nil {
		return nil, fmt.Errorf("policy_id: %w", err)
	}
	if env.algo, err = r.u8(); err != nil {
		return nil, fmt.Errorf("algo: %w", err)
	}
	if env.exp, err = r.lenPrefixed32(); err != nil {
		return nil, fmt.Errorf("exp: %w", err)
	}
	if env.mod, err = r.lenPrefixed32(); err != nil {
		return nil, fmt.Errorf("mod: %w", err)
	}
	if env.encKey, err = r.lenPrefixed32(); err != nil {
		return nil, fmt.Errorf("enc_key: %w", err)
	}
	env.encMsg = r.rest()
	return &env, nil
}

type bytesReader struct {
	buf []byte
	off int
}

func (r *bytesReader) need(n int) error {
	if r.off+n > len(r.buf) {
		return fmt.Errorf("short read: want %d have %d", n, len(r.buf)-r.off)
	}
	return nil
}
func (r *bytesReader) u8() (uint8, error) {
	if err := r.need(1); err != nil {
		return 0, err
	}
	v := r.buf[r.off]
	r.off++
	return v, nil
}
func (r *bytesReader) u16() (uint16, error) {
	if err := r.need(2); err != nil {
		return 0, err
	}
	v := binary.BigEndian.Uint16(r.buf[r.off:])
	r.off += 2
	return v, nil
}
func (r *bytesReader) u32() (uint32, error) {
	if err := r.need(4); err != nil {
		return 0, err
	}
	v := binary.BigEndian.Uint32(r.buf[r.off:])
	r.off += 4
	return v, nil
}
func (r *bytesReader) u64() (uint64, error) {
	if err := r.need(8); err != nil {
		return 0, err
	}
	v := binary.BigEndian.Uint64(r.buf[r.off:])
	r.off += 8
	return v, nil
}
func (r *bytesReader) bytes(n int) ([]byte, error) {
	if err := r.need(n); err != nil {
		return nil, err
	}
	v := r.buf[r.off : r.off+n]
	r.off += n
	return v, nil
}
func (r *bytesReader) lenPrefixed32() ([]byte, error) {
	n, err := r.u32()
	if err != nil {
		return nil, err
	}
	return r.bytes(int(n))
}
func (r *bytesReader) rest() []byte { return r.buf[r.off:] }

func TestBlindfold_EmptyPlaintext(t *testing.T) {
	_, key := newTestKey(t)
	policy := &PolicyData{PolicyID: 1}
	if _, err := Blindfold(nil, key, policy); err == nil {
		t.Fatal("expected error for empty plaintext")
	}
}

func TestBlindfold_NilPolicy(t *testing.T) {
	_, key := newTestKey(t)
	if _, err := Blindfold([]byte("x"), key, nil); err == nil {
		t.Fatal("expected error for nil policy")
	}
}

func TestBlindfold_Failures(t *testing.T) {
	_, key := newTestKey(t)
	policy := &PolicyData{PolicyID: 1}

	t.Run("bad_modulus", func(t *testing.T) {
		badKey := *key
		badKey.ModulusBase64 = "not-base64-!"
		if _, err := Blindfold([]byte("x"), &badKey, policy); err == nil {
			t.Fatal("expected error for bad modulus")
		}
	})
	t.Run("bad_exponent", func(t *testing.T) {
		badKey := *key
		badKey.PublicExponentBase64 = "not-base64-!"
		if _, err := Blindfold([]byte("x"), &badKey, policy); err == nil {
			t.Fatal("expected error for bad exponent")
		}
	})
	t.Run("small_modulus", func(t *testing.T) {
		badKey := *key
		badKey.ModulusBase64 = base64.StdEncoding.EncodeToString([]byte("small"))
		if _, err := Blindfold([]byte("x"), &badKey, policy); err == nil {
			t.Fatal("expected error for small modulus")
		}
	})
}

func TestPolicyIDU64_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		in      []byte
		wantErr bool
	}{
		{"number", []byte("123"), false},
		{"string", []byte(`"456"`), false},
		{"null", []byte("null"), false},
		{"empty", []byte(""), false},
		{"bad_string", []byte(`"xyz"`), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p policyIDU64
			err := p.UnmarshalJSON(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// GetPublicKey accepts a response that carries the key fields at the top
// level, without the "data" wrapper.
func TestGetPublicKey_FlatResponse(t *testing.T) {
	base := "https://tenant.example/api"
	_, key := newTestKey(t)
	body, err := json.Marshal(key)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	c := &Client{
		Exec: &fakeExec{responses: map[string][]byte{
			base + "/secret_management/get_public_key": body,
		}},
		BaseURL: base + "/",
	}

	got, err := c.GetPublicKey(context.Background())
	if err != nil {
		t.Fatalf("GetPublicKey: %v", err)
	}
	if got.ModulusBase64 != key.ModulusBase64 || got.PublicExponentBase64 != key.PublicExponentBase64 {
		t.Fatalf("unexpected KeyData: %+v", got)
	}
}

// An empty JSON string leaves the policy ID unchanged and is not an error.
func TestPolicyIDU64_UnmarshalJSON_EmptyString(t *testing.T) {
	p := policyIDU64(9)
	if err := p.UnmarshalJSON([]byte(`""`)); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if p != 9 {
		t.Fatalf("policy ID changed to %d", p)
	}
}

// decodeBase64Flexible decodes every standard and URL-safe variant, with and
// without padding, and rejects empty or invalid input.
func TestDecodeBase64Flexible(t *testing.T) {
	// 0xfb 0xff produces '+' and '/' in standard encoding and '-' and '_' in
	// URL encoding, and needs padding.
	raw := []byte{0xfb, 0xff, 0xfe, 0x01}
	inputs := map[string]string{
		"std":        base64.StdEncoding.EncodeToString(raw),
		"raw_std":    base64.RawStdEncoding.EncodeToString(raw),
		"url":        base64.URLEncoding.EncodeToString(raw),
		"raw_url":    base64.RawURLEncoding.EncodeToString(raw),
		"whitespace": "  " + base64.StdEncoding.EncodeToString(raw) + "\n",
	}
	for name, in := range inputs {
		t.Run(name, func(t *testing.T) {
			got, err := decodeBase64Flexible(in)
			if err != nil {
				t.Fatalf("decode %q: %v", in, err)
			}
			if !bytes.Equal(got, raw) {
				t.Fatalf("decode %q = %x, want %x", in, got, raw)
			}
		})
	}

	for name, in := range map[string]string{"empty": "   ", "invalid": "@@@"} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeBase64Flexible(in); err == nil {
				t.Fatalf("expected error for %q", in)
			}
		})
	}
}

// Blindfold rejects a nil key.
func TestBlindfold_NilKey(t *testing.T) {
	if _, err := Blindfold([]byte("x"), nil, &PolicyData{PolicyID: 1}); err == nil {
		t.Fatal("expected error for nil key")
	}
}

// The bundle uses the policy tenant when the key has no tenant.
func TestBlindfold_TenantFallsBackToPolicy(t *testing.T) {
	_, key := newTestKey(t)
	noTenant := *key
	noTenant.Tenant = ""
	policy := &PolicyData{Tenant: "policy-tenant", PolicyID: 101}

	out, err := Blindfold([]byte("payload"), &noTenant, policy)
	if err != nil {
		t.Fatalf("Blindfold: %v", err)
	}
	buf, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(out, "string:///"))
	if err != nil {
		t.Fatalf("decode bundle: %v", err)
	}
	env, err := parseBlindfoldEnvelope(buf)
	if err != nil {
		t.Fatalf("parse envelope: %v", err)
	}
	if env.tenant != "policy-tenant" {
		t.Fatalf("bundle tenant = %q, want %q", env.tenant, "policy-tenant")
	}
}

// head truncates strings longer than n and returns shorter strings unchanged.
func TestHead(t *testing.T) {
	if got := head("abcdef", 3); got != "abc" {
		t.Fatalf("head long = %q", got)
	}
	if got := head("ab", 3); got != "ab" {
		t.Fatalf("head short = %q", got)
	}
}
