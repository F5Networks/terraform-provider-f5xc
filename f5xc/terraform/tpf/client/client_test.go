// Package client F5XC
// Copyright (c) 2025 F5 Inc. All rights reserved.

package client

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.volterra.us/stdlib/client/vesapi"
	"gopkg.volterra.us/stdlib/server"
	"software.sslmate.com/src/go-pkcs12"

	ves_io_schema_combined "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/combined"
)

const testTerraformVersion = "1.5.7"

// testSecret returns a random credential-shaped value, so tests never embed
// token, password or cookie literals.
func testSecret(t *testing.T) string {
	t.Helper()
	return rand.Text()
}

func testProviderUserAgent() string {
	return providerUserAgent(testTerraformVersion)
}

// ---------------------------------------------------------------------------
// Helpers / mocks
// ---------------------------------------------------------------------------

// mockAuthenticator is a test double for the Authenticator interface.
type mockAuthenticator struct {
	applyAuthErr  error
	client        *http.Client
	clientData    string
	appliedHeader string
}

func (m *mockAuthenticator) ApplyAuth(req *http.Request) error {
	if m.applyAuthErr != nil {
		return m.applyAuthErr
	}
	if m.appliedHeader != "" {
		req.Header.Set("Authorization", m.appliedHeader)
	}
	return nil
}

func (m *mockAuthenticator) GetClient() *http.Client {
	if m.client != nil {
		return m.client
	}
	return &http.Client{Timeout: 5 * time.Second}
}

func (m *mockAuthenticator) GetClientData() string {
	return m.clientData
}

// mockRoundTripper is a test double for http.RoundTripper.
type mockRoundTripper struct {
	response *http.Response
	err      error
	lastReq  *http.Request
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.lastReq = req
	if m.err != nil {
		return nil, m.err
	}
	if m.response != nil {
		return m.response, nil
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("")),
	}, nil
}

// createTempFile writes content to a temporary file and returns its path.
func createTempFile(t *testing.T, content []byte) (string, error) {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "test-*.p12")
	if err != nil {
		return "", fmt.Errorf("os.CreateTemp: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(content); err != nil {
		return "", fmt.Errorf("f.Write: %w", err)
	}
	return f.Name(), nil
}

// ---------------------------------------------------------------------------
// NewAuthClient
// ---------------------------------------------------------------------------

func TestNewAuthClient(t *testing.T) {
	auth := &mockAuthenticator{clientData: "tok"}
	ac := NewAuthClient("https://example.com", auth, testProviderUserAgent())

	if ac == nil {
		t.Fatal("expected non-nil AuthClient")
	}
	if ac.URL != "https://example.com" {
		t.Errorf("URL = %q, want %q", ac.URL, "https://example.com")
	}
	if ac.Authenticator != auth {
		t.Error("Authenticator not stored correctly")
	}
}

func TestNewAuthClient_EmptyURL(t *testing.T) {
	auth := &mockAuthenticator{}
	ac := NewAuthClient("", auth, testProviderUserAgent())
	if ac.URL != "" {
		t.Errorf("expected empty URL, got %q", ac.URL)
	}
}

func TestProviderUserAgent(t *testing.T) {
	ua := providerUserAgent(testTerraformVersion)
	if !strings.HasPrefix(ua, "terraform/1.5.7 terraform-provider-f5xc/") {
		t.Fatalf("User-Agent = %q, want terraform/<version> terraform-provider-f5xc/<version>", ua)
	}

	fallbackUA := providerUserAgent("")
	if !strings.HasPrefix(fallbackUA, "terraform/unknown terraform-provider-f5xc/") {
		t.Fatalf("fallback User-Agent = %q, want terraform/unknown terraform-provider-f5xc/<version>", fallbackUA)
	}
}

func TestProviderUserAgentWithVersion(t *testing.T) {
	t.Run("explicit-release-version", func(t *testing.T) {
		ua := providerUserAgentWithVersion("1.8.2", "3.1.4")
		if ua != "terraform/1.8.2 terraform-provider-f5xc/3.1.4" {
			t.Fatalf("User-Agent = %q, want %q", ua, "terraform/1.8.2 terraform-provider-f5xc/3.1.4")
		}
	})

	t.Run("explicit-version-with-v-prefix", func(t *testing.T) {
		ua := providerUserAgentWithVersion("1.8.2", "v3.1.4")
		if ua != "terraform/1.8.2 terraform-provider-f5xc/3.1.4" {
			t.Fatalf("User-Agent = %q, want %q", ua, "terraform/1.8.2 terraform-provider-f5xc/3.1.4")
		}
	})

	t.Run("fallback-on-empty-or-dev", func(t *testing.T) {
		for _, configured := range []string{"", "dev", "  "} {
			ua := providerUserAgentWithVersion(testTerraformVersion, configured)
			if !strings.HasPrefix(ua, "terraform/1.5.7 terraform-provider-f5xc/") {
				t.Fatalf("User-Agent(%q) = %q, want prefix terraform/1.5.7 terraform-provider-f5xc/", configured, ua)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// NewAPITokenAuth
// ---------------------------------------------------------------------------

func TestNewAPITokenAuth(t *testing.T) {
	timeout := 10 * time.Second
	token := testSecret(t)
	a := NewAPITokenAuth(token, timeout)

	if a == nil {
		t.Fatal("expected non-nil APITokenAuth")
	}
	if a.Token != token {
		t.Errorf("Token = %q, want %q", a.Token, token)
	}
	if a.Client == nil {
		t.Fatal("expected non-nil http.Client")
	}
	if a.Client.Timeout != timeout {
		t.Errorf("Client.Timeout = %v, want %v", a.Client.Timeout, timeout)
	}
}

func TestNewAPITokenAuth_EmptyToken(t *testing.T) {
	a := NewAPITokenAuth("", 5*time.Second)
	if a.Token != "" {
		t.Errorf("expected empty token, got %q", a.Token)
	}
}

// ---------------------------------------------------------------------------
// APITokenAuth – ApplyAuth
// ---------------------------------------------------------------------------

func TestAPITokenAuth_ApplyAuth(t *testing.T) {
	token := testSecret(t)
	a := &APITokenAuth{Token: token}
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	req.Header = make(http.Header)

	if err := a.ApplyAuth(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "APIToken " + token
	if got := req.Header.Get("Authorization"); got != want {
		t.Errorf("Authorization = %q, want %q", got, want)
	}
}

func TestAPITokenAuth_ApplyAuth_TrimsWhitespace(t *testing.T) {
	token := testSecret(t)
	a := &APITokenAuth{Token: "  " + token + "  "}
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	req.Header = make(http.Header)

	if err := a.ApplyAuth(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "APIToken " + token
	if got := req.Header.Get("Authorization"); got != want {
		t.Errorf("Authorization = %q, want %q", got, want)
	}
}

func TestAPITokenAuth_ApplyAuth_EmptyToken(t *testing.T) {
	a := &APITokenAuth{Token: ""}
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	req.Header = make(http.Header)

	if err := a.ApplyAuth(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Even with an empty token the header should be set (just "APIToken ")
	if got := req.Header.Get("Authorization"); !strings.HasPrefix(got, "APIToken") {
		t.Errorf("Authorization = %q, expected prefix 'APIToken'", got)
	}
}

// ---------------------------------------------------------------------------
// APITokenAuth – GetClient
// ---------------------------------------------------------------------------

func TestAPITokenAuth_GetClient(t *testing.T) {
	a := &APITokenAuth{Token: testSecret(t)}
	c := a.GetClient()
	if c == nil {
		t.Fatal("expected non-nil http.Client")
	}
	if c.Timeout != 10*time.Second {
		t.Errorf("Timeout = %v, want 10s", c.Timeout)
	}
}

// ---------------------------------------------------------------------------
// APITokenAuth – GetClientData
// ---------------------------------------------------------------------------

func TestAPITokenAuth_GetClientData(t *testing.T) {
	token := testSecret(t)
	a := &APITokenAuth{Token: token}
	if got := a.GetClientData(); got != token {
		t.Errorf("GetClientData() = %q, want %q", got, token)
	}
}

func TestAPITokenAuth_GetClientData_Empty(t *testing.T) {
	a := &APITokenAuth{Token: ""}
	if got := a.GetClientData(); got != "" {
		t.Errorf("GetClientData() = %q, want empty string", got)
	}
}

// ---------------------------------------------------------------------------
// P12CertAuth – ApplyAuth (no-op for mTLS)
// ---------------------------------------------------------------------------

func TestP12CertAuth_ApplyAuth_NoOp(t *testing.T) {
	a := &P12CertAuth{
		P12Path:  "/some/path.p12",
		Password: testSecret(t),
		Client:   &http.Client{},
	}
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	req.Header = make(http.Header)

	if err := a.ApplyAuth(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No Authorization header should be set for mTLS
	if got := req.Header.Get("Authorization"); got != "" {
		t.Errorf("expected no Authorization header, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// P12CertAuth – GetClient
// ---------------------------------------------------------------------------

func TestP12CertAuth_GetClient(t *testing.T) {
	inner := &http.Client{Timeout: 30 * time.Second}
	a := &P12CertAuth{Client: inner}
	if got := a.GetClient(); got != inner {
		t.Error("GetClient() did not return the stored client")
	}
}

// ---------------------------------------------------------------------------
// P12CertAuth – GetClientData
// ---------------------------------------------------------------------------

func TestP12CertAuth_GetClientData(t *testing.T) {
	a := &P12CertAuth{}
	if got := a.GetClientData(); got != "" {
		t.Errorf("GetClientData() = %q, want empty string", got)
	}
}

// ---------------------------------------------------------------------------
// AuthClient – Execute
// ---------------------------------------------------------------------------

func TestAuthClient_Execute_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept header = %q, want application/json", r.Header.Get("Accept"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type header = %q, want application/json", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	auth := &mockAuthenticator{client: srv.Client()}
	ac := NewAuthClient(srv.URL, auth, testProviderUserAgent())

	body, err := ac.Execute(context.Background(), http.MethodGet, srv.URL+"/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	if jsonErr := json.Unmarshal(body, &result); jsonErr != nil {
		t.Fatalf("failed to unmarshal response: %v", jsonErr)
	}
	if result["status"] != "ok" {
		t.Errorf("status = %q, want %q", result["status"], "ok")
	}
}

func TestAuthClient_Execute_SetsAuthorizationHeader(t *testing.T) {
	var receivedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	wantAuth := "APIToken " + testSecret(t)
	auth := &mockAuthenticator{
		client:        srv.Client(),
		appliedHeader: wantAuth,
	}
	ac := NewAuthClient(srv.URL, auth, testProviderUserAgent())

	_, err := ac.Execute(context.Background(), http.MethodGet, srv.URL+"/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedAuth != wantAuth {
		t.Errorf("Authorization = %q, want %q", receivedAuth, wantAuth)
	}
}

func TestAuthClient_Execute_SetsStandardHeaders(t *testing.T) {
	var capturedUserAgent, capturedAccept, capturedContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserAgent = r.Header.Get("User-Agent")
		capturedAccept = r.Header.Get("Accept")
		capturedContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	auth := &mockAuthenticator{client: srv.Client()}
	ac := NewAuthClient(srv.URL, auth, testProviderUserAgent())

	_, err := ac.Execute(context.Background(), http.MethodGet, srv.URL+"/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedUserAgent != testProviderUserAgent() {
		t.Errorf("User-Agent = %q, want %q", capturedUserAgent, testProviderUserAgent())
	}
	if capturedAccept != "application/json" {
		t.Errorf("Accept = %q, want application/json", capturedAccept)
	}
	if capturedContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", capturedContentType)
	}
}

func TestAuthClient_Execute_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	}))
	defer srv.Close()

	auth := &mockAuthenticator{client: srv.Client()}
	ac := NewAuthClient(srv.URL, auth, testProviderUserAgent())

	_, err := ac.Execute(context.Background(), http.MethodGet, srv.URL+"/missing", nil)
	if err == nil {
		t.Fatal("expected error for non-200 status, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error = %q, expected to contain '404'", err.Error())
	}
}

func TestAuthClient_Execute_500Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal server error"}`))
	}))
	defer srv.Close()

	auth := &mockAuthenticator{client: srv.Client()}
	ac := NewAuthClient(srv.URL, auth, testProviderUserAgent())

	_, err := ac.Execute(context.Background(), http.MethodGet, srv.URL+"/fail", nil)
	if err == nil {
		t.Fatal("expected error for 500 status, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error = %q, expected to contain '500'", err.Error())
	}
}

func TestAuthClient_Execute_ApplyAuthError(t *testing.T) {
	auth := &mockAuthenticator{
		applyAuthErr: fmt.Errorf("auth failed"),
		client:       &http.Client{},
	}
	ac := NewAuthClient("https://example.com", auth, testProviderUserAgent())

	_, err := ac.Execute(context.Background(), http.MethodGet, "https://example.com/test", nil)
	if err == nil {
		t.Fatal("expected error when ApplyAuth fails, got nil")
	}
	if !strings.Contains(err.Error(), "auth failed") {
		t.Errorf("error = %q, expected to contain 'auth failed'", err.Error())
	}
}

func TestAuthClient_Execute_InvalidURL(t *testing.T) {
	auth := &mockAuthenticator{client: &http.Client{}}
	ac := NewAuthClient("://bad-url", auth, testProviderUserAgent())

	_, err := ac.Execute(context.Background(), http.MethodGet, "://bad-url", nil)
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}

func TestAuthClient_Execute_WithBody(t *testing.T) {
	var receivedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		receivedBody = buf
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	auth := &mockAuthenticator{client: srv.Client()}
	ac := NewAuthClient(srv.URL, auth, testProviderUserAgent())

	payload := []byte(`{"key":"value"}`)
	_, err := ac.Execute(context.Background(), http.MethodPost, srv.URL+"/create", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(receivedBody) != string(payload) {
		t.Errorf("body = %q, want %q", string(receivedBody), string(payload))
	}
}

// ---------------------------------------------------------------------------
// serviceSlugFromContext
// ---------------------------------------------------------------------------

func TestServiceSlugFromContext_Present(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxServiceSlug{}, "config")
	slug := serviceSlugFromContext(ctx)
	if slug != "config" {
		t.Errorf("slug = %q, want %q", slug, "config")
	}
}

func TestServiceSlugFromContext_Missing(t *testing.T) {
	slug := serviceSlugFromContext(context.Background())
	if slug != "" {
		t.Errorf("slug = %q, want empty string", slug)
	}
}

func TestServiceSlugFromContext_WrongType(t *testing.T) {
	// Store an int instead of a string – should return ""
	ctx := context.WithValue(context.Background(), ctxServiceSlug{}, 42)
	slug := serviceSlugFromContext(ctx)
	if slug != "" {
		t.Errorf("slug = %q, want empty string for wrong type", slug)
	}
}

// ---------------------------------------------------------------------------
// contextFromServiceSlug
// ---------------------------------------------------------------------------

func TestContextFromServiceSlug(t *testing.T) {
	ctx := contextFromServiceSlug(context.Background(), "my-slug")
	if got := serviceSlugFromContext(ctx); got != "my-slug" {
		t.Errorf("slug = %q, want %q", got, "my-slug")
	}
}

func TestContextFromServiceSlug_EmptySlug(t *testing.T) {
	ctx := contextFromServiceSlug(context.Background(), "")
	// An empty slug is stored and retrieved as empty string.
	if got := serviceSlugFromContext(ctx); got != "" {
		t.Errorf("slug = %q, want empty string", got)
	}
}

// ---------------------------------------------------------------------------
// addSvcSlugToContextForCRUD
// ---------------------------------------------------------------------------

func TestAddSvcSlugToContextForCRUD_UnknownType(t *testing.T) {
	ctx, err := addSvcSlugToContextForCRUD(context.Background(), "unknown.object.type")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
}

func TestAddSvcSlugToContextForCRUD_EmptyType(t *testing.T) {
	ctx, err := addSvcSlugToContextForCRUD(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
}

// ---------------------------------------------------------------------------
// addSvcSlugToContextForCustom
// ---------------------------------------------------------------------------

func TestAddSvcSlugToContextForCustom_UnknownFQN(t *testing.T) {
	ctx, err := addSvcSlugToContextForCustom(context.Background(), "ves.io.unknown.service.Method")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
}

func TestAddSvcSlugToContextForCustom_SingleSegmentFQN(t *testing.T) {
	// A single-segment FQN (no dots) should not panic.
	ctx, err := addSvcSlugToContextForCustom(context.Background(), "Method")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
}

// ---------------------------------------------------------------------------
// ProviderConfig / ThrottlingConfig struct field tests
// ---------------------------------------------------------------------------

func TestProviderConfig_Fields(t *testing.T) {
	throttling := &ThrottlingConfig{
		MaxRetries:        5,
		InitialBackoff:    "250ms",
		MaxBackoff:        "8s",
		Multiplier:        2.0,
		JitterRatio:       0.2,
		RespectRetryAfter: true,
	}
	apiToken := testSecret(t)
	cfg := &ProviderConfig{
		Url:             "https://tenant.example.com",
		Test:            false,
		ProviderVersion: "2.1.0",
		Vesenv:          true,
		Timeout:         "30s",
		Throttling:      throttling,
		ApiCert:         "/path/to/cert",
		ApiKey:          "/path/to/key",
		ApiToken:        apiToken,
		ApiCACert:       "/path/to/ca",
		ApiP12File:      "/path/to/file.p12",
		ApiP12Password:  testSecret(t),
	}

	if cfg.Url != "https://tenant.example.com" {
		t.Errorf("Url = %q", cfg.Url)
	}
	if cfg.ProviderVersion != "2.1.0" {
		t.Errorf("ProviderVersion = %q", cfg.ProviderVersion)
	}
	if cfg.Timeout != "30s" {
		t.Errorf("Timeout = %q", cfg.Timeout)
	}
	if cfg.Throttling.MaxRetries != 5 {
		t.Errorf("Throttling.MaxRetries = %d", cfg.Throttling.MaxRetries)
	}
	if cfg.ApiToken != apiToken {
		t.Errorf("ApiToken = %q", cfg.ApiToken)
	}
	if !cfg.Vesenv {
		t.Error("Vesenv should be true")
	}
}

// ---------------------------------------------------------------------------
// Throttling policy tests
// ---------------------------------------------------------------------------

func TestResolveThrottlingPolicy_Defaults(t *testing.T) {
	policy, err := resolveThrottlingPolicy(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.Enabled {
		t.Fatal("policy.Enabled should default to false when cfg is nil")
	}
	if policy.MaxRetries <= 0 {
		t.Fatalf("policy.MaxRetries should be > 0, got %d", policy.MaxRetries)
	}
}

func TestResolveThrottlingPolicy_EnabledWhenConfigProvided(t *testing.T) {
	policy, err := resolveThrottlingPolicy(&ThrottlingConfig{MaxRetries: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !policy.Enabled {
		t.Fatal("policy.Enabled should be true when config is provided")
	}
}

func TestResolveThrottlingPolicy_ZeroMaxRetries(t *testing.T) {
	policy, err := resolveThrottlingPolicy(&ThrottlingConfig{MaxRetries: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.MaxRetries != 0 {
		t.Fatalf("policy.MaxRetries should be 0, got %d", policy.MaxRetries)
	}
}

// ---------------------------------------------------------------------------
// CRUD operation constants
// ---------------------------------------------------------------------------

func TestCRUDConstants(t *testing.T) {
	if Create != 0 {
		t.Errorf("Create = %d, want 0", Create)
	}
	if Delete != 1 {
		t.Errorf("Delete = %d, want 1", Delete)
	}
	if Replace != 2 {
		t.Errorf("Replace = %d, want 2", Replace)
	}
	if Get != 3 {
		t.Errorf("Get = %d, want 3", Get)
	}
}

// ---------------------------------------------------------------------------
// buildTLSConfigFromP12 – error paths
// ---------------------------------------------------------------------------

func TestBuildTLSConfigFromP12_FileNotFound(t *testing.T) {
	_, err := buildTLSConfigFromP12("/nonexistent/path/file.p12", testSecret(t))
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read p12 file") {
		t.Errorf("error = %q, expected to contain 'failed to read p12 file'", err.Error())
	}
}

func TestBuildTLSConfigFromP12_FileURIPrefix(t *testing.T) {
	// The function strips "file:///" prefix; the resulting path should still not exist.
	_, err := buildTLSConfigFromP12("file:///nonexistent/path/file.p12", testSecret(t))
	if err == nil {
		t.Fatal("expected error for non-existent file with file:/// prefix, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read p12 file") {
		t.Errorf("error = %q, expected to contain 'failed to read p12 file'", err.Error())
	}
}

func TestBuildTLSConfigFromP12_NullByte(t *testing.T) {
	_, err := buildTLSConfigFromP12("/path/with/\x00/null.p12", testSecret(t))
	if err == nil {
		t.Fatal("expected error for path with null byte, got nil")
	}
	if !strings.Contains(err.Error(), "contains null byte") {
		t.Errorf("error = %q, expected to contain 'contains null byte'", err.Error())
	}
}

func TestBuildTLSConfigFromP12_InvalidP12Data(t *testing.T) {
	tmpFile, err := createTempFile(t, []byte("not-a-valid-p12-file"))
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	_, err = buildTLSConfigFromP12(tmpFile, testSecret(t))
	if err == nil {
		t.Fatal("expected error for invalid P12 data, got nil")
	}
	if !strings.Contains(err.Error(), "failed to decode p12") {
		t.Errorf("error = %q, expected to contain 'failed to decode p12'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// NewP12CertAuth – error paths
// ---------------------------------------------------------------------------

func TestNewP12CertAuth_FileNotFound(t *testing.T) {
	_, err := NewP12CertAuth("/nonexistent/path/file.p12", testSecret(t), 10*time.Second)
	if err == nil {
		t.Fatal("expected error for non-existent P12 file, got nil")
	}
}

func TestNewP12CertAuth_InvalidP12(t *testing.T) {
	tmpFile, err := createTempFile(t, []byte("invalid-p12-data"))
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	_, err = NewP12CertAuth(tmpFile, testSecret(t), 10*time.Second)
	if err == nil {
		t.Fatal("expected error for invalid P12 data, got nil")
	}
}

// ---------------------------------------------------------------------------
// prismURIRemapperFn
// ---------------------------------------------------------------------------

func TestPrismURIRemapperFn_InvalidURL(t *testing.T) {
	policy, policyErr := resolveThrottlingPolicy(nil)
	if policyErr != nil {
		t.Fatalf("unexpected policy error: %v", policyErr)
	}
	_, err := prismURIRemapperFn("://invalid", testProviderUserAgent(), policy, newResourceThrottleRegistry())
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}

func TestPrismURIRemapperFn_ValidURL(t *testing.T) {
	policy, policyErr := resolveThrottlingPolicy(nil)
	if policyErr != nil {
		t.Fatalf("unexpected policy error: %v", policyErr)
	}
	fn, err := prismURIRemapperFn("https://tenant.example.com/api", testProviderUserAgent(), policy, newResourceThrottleRegistry())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fn == nil {
		t.Fatal("expected non-nil RoundTripperFunc")
	}
}

func TestPrismURIRemapperFn_NoSlugInContext(t *testing.T) {
	policy, policyErr := resolveThrottlingPolicy(nil)
	if policyErr != nil {
		t.Fatalf("unexpected policy error: %v", policyErr)
	}
	fn, err := prismURIRemapperFn("https://tenant.example.com/api", testProviderUserAgent(), policy, newResourceThrottleRegistry())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req, err := http.NewRequestWithContext(
		context.Background(), // no slug in context
		http.MethodGet,
		"https://tenant.example.com/api/public/namespaces/default/http_loadbalancers",
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	rt := &mockRoundTripper{}
	_, err = fn(req, rt)
	if err == nil {
		t.Fatal("expected error when no service slug is in context, got nil")
	}
	if !strings.Contains(err.Error(), "no service slug defined") {
		t.Errorf("error = %q, expected to contain 'no service slug defined'", err.Error())
	}
}

func TestPrismURIRemapperFn_WithSlugInContext(t *testing.T) {
	policy, policyErr := resolveThrottlingPolicy(nil)
	if policyErr != nil {
		t.Fatalf("unexpected policy error: %v", policyErr)
	}
	fn, err := prismURIRemapperFn("https://tenant.example.com/api", testProviderUserAgent(), policy, newResourceThrottleRegistry())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := contextFromServiceSlug(context.Background(), "config")
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"https://tenant.example.com/api/public/namespaces/default/http_loadbalancers",
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	rt := &mockRoundTripper{response: &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("")),
	}}
	resp, err := fn(req, rt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if rt.lastReq == nil {
		t.Fatal("expected round tripper to capture request")
	}
	// The URL should have been remapped: "/public" replaced by "/config"
	if strings.Contains(rt.lastReq.URL.Path, "/public") {
		t.Errorf("URL path still contains '/public': %s", rt.lastReq.URL.Path)
	}
	if !strings.Contains(rt.lastReq.URL.Path, "/config") {
		t.Errorf("URL path does not contain '/config': %s", rt.lastReq.URL.Path)
	}
	if got := rt.lastReq.Header.Get("User-Agent"); got != testProviderUserAgent() {
		t.Errorf("User-Agent = %q, want %q", got, testProviderUserAgent())
	}
}

// ---------------------------------------------------------------------------
// uriRemapperFn
// ---------------------------------------------------------------------------

func TestURIRemapperFn_SetsTenantHeader(t *testing.T) {
	policy, policyErr := resolveThrottlingPolicy(nil)
	if policyErr != nil {
		t.Fatalf("unexpected policy error: %v", policyErr)
	}
	fn := uriRemapperFn("my-tenant", testProviderUserAgent(), policy, newResourceThrottleRegistry())

	req, err := http.NewRequest(http.MethodGet, "https://example.com/api/test", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	rt := &mockRoundTripper{response: &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("")),
	}}
	_, err = fn(req, rt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.lastReq == nil {
		t.Fatal("expected round tripper to capture request")
	}
	if got := rt.lastReq.Header.Get("X-Volterra-Apigw-Tenant"); got != "my-tenant" {
		t.Errorf("X-Volterra-Apigw-Tenant = %q, want %q", got, "my-tenant")
	}
	if got := rt.lastReq.Header.Get("User-Agent"); got != testProviderUserAgent() {
		t.Errorf("User-Agent = %q, want %q", got, testProviderUserAgent())
	}
}

func TestURIRemapperFn_EmptyTenant(t *testing.T) {
	policy, policyErr := resolveThrottlingPolicy(nil)
	if policyErr != nil {
		t.Fatalf("unexpected policy error: %v", policyErr)
	}
	fn := uriRemapperFn("", testProviderUserAgent(), policy, newResourceThrottleRegistry())

	req, err := http.NewRequest(http.MethodGet, "https://example.com/api/test", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	rt := &mockRoundTripper{response: &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("")),
	}}
	_, err = fn(req, rt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.lastReq == nil {
		t.Fatal("expected round tripper to capture request")
	}
	// Empty tenant should still set the header (to empty string)
	if got := rt.lastReq.Header.Get("x-ves-tenant"); got != "" {
		t.Errorf("x-ves-tenant = %q, want empty string", got)
	}
	if got := rt.lastReq.Header.Get("User-Agent"); got != testProviderUserAgent() {
		t.Errorf("User-Agent = %q, want %q", got, testProviderUserAgent())
	}
}

// ---------------------------------------------------------------------------
// Authenticator interface compliance (compile-time checks)
// ---------------------------------------------------------------------------

func TestAPITokenAuth_ImplementsAuthenticator(t *testing.T) {
	var _ Authenticator = &APITokenAuth{}
}

func TestP12CertAuth_ImplementsAuthenticator(t *testing.T) {
	var _ Authenticator = &P12CertAuth{}
}

func TestResourceThrottleRegistry_SameResourceWaits(t *testing.T) {
	registry := newResourceThrottleRegistry()
	registry.publish("http_loadbalancer", 35*time.Millisecond)

	start := time.Now()
	err := registry.wait(context.Background(), "http_loadbalancer")
	if err != nil {
		t.Fatalf("unexpected wait error: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 25*time.Millisecond {
		t.Fatalf("expected wait >= 25ms, got %v", elapsed)
	}
}

func TestResourceThrottleRegistry_DifferentResourcesIsolated(t *testing.T) {
	registry := newResourceThrottleRegistry()
	registry.publish("http_loadbalancer", 200*time.Millisecond)

	start := time.Now()
	err := registry.wait(context.Background(), "origin_pool")
	if err != nil {
		t.Fatalf("unexpected wait error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 20*time.Millisecond {
		t.Fatalf("expected isolated resource wait to be near-zero, got %v", elapsed)
	}
}

func TestDoRequestWithThrottling_RetriesOn429(t *testing.T) {
	policy := defaultThrottlingPolicy()
	policy.Enabled = true
	policy.InitialBackoff = 1 * time.Millisecond
	policy.MaxBackoff = 2 * time.Millisecond
	policy.JitterRatio = 0
	policy.MaxRetries = 2

	registry := newResourceThrottleRegistry()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com/api/public/namespaces/default/http_loadbalancers", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	attempts := 0
	resp, err := doRequestWithThrottling(context.Background(), req, "http_loadbalancer", policy, registry, func(_ *http.Request) (*http.Response, error) {
		attempts++
		if attempts == 1 {
			return &http.Response{StatusCode: http.StatusTooManyRequests, Body: io.NopCloser(strings.NewReader("429")), Header: make(http.Header)}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("ok")), Header: make(http.Header)}, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 after retry, got %d", resp.StatusCode)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestDoRequestWithThrottling_ContextCancelledDuringCooldown(t *testing.T) {
	policy := defaultThrottlingPolicy()
	policy.Enabled = true
	policy.InitialBackoff = 1 * time.Millisecond
	policy.MaxBackoff = 1 * time.Millisecond
	policy.JitterRatio = 0

	registry := newResourceThrottleRegistry()
	registry.publish("http_loadbalancer", 2*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.com/api/public/namespaces/default/http_loadbalancers", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	_, err = doRequestWithThrottling(ctx, req, "http_loadbalancer", policy, registry, func(_ *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("ok"))}, nil
	})
	if err == nil {
		t.Fatal("expected cancellation error, got nil")
	}
}

func TestParseRetryAfterDelay_Seconds(t *testing.T) {
	resp := &http.Response{Header: make(http.Header)}
	resp.Header.Set("Retry-After", "3")

	delay, ok := parseRetryAfterDelay(resp)
	if !ok {
		t.Fatal("expected Retry-After to parse")
	}
	if delay != 3*time.Second {
		t.Fatalf("expected 3s delay, got %v", delay)
	}
}

func TestDoRequestWithThrottling_UsesRetryAfterHeader(t *testing.T) {
	policy := defaultThrottlingPolicy()
	policy.Enabled = true
	policy.InitialBackoff = 1 * time.Millisecond
	policy.MaxBackoff = 2 * time.Millisecond
	policy.JitterRatio = 0
	policy.MaxRetries = 1
	policy.RespectRetryAfter = true

	registry := newResourceThrottleRegistry()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com/api/public/namespaces/default/http_loadbalancers", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	attempts := 0
	start := time.Now()
	_, err = doRequestWithThrottling(context.Background(), req, "http_loadbalancer", policy, registry, func(_ *http.Request) (*http.Response, error) {
		attempts++
		if attempts == 1 {
			rsp := &http.Response{StatusCode: http.StatusTooManyRequests, Body: io.NopCloser(strings.NewReader("429")), Header: make(http.Header)}
			rsp.Header.Set("Retry-After", "1")
			return rsp, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("ok")), Header: make(http.Header)}, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if elapsed := time.Since(start); elapsed < 900*time.Millisecond {
		t.Fatalf("expected retry to respect Retry-After near 1s, got %v", elapsed)
	}
}

func TestComputeBackoffDelay_Bounds(t *testing.T) {
	policy := defaultThrottlingPolicy()
	policy.InitialBackoff = 100 * time.Millisecond
	policy.MaxBackoff = 300 * time.Millisecond
	policy.Multiplier = 2.0
	policy.JitterRatio = 0

	if d := computeBackoffDelay(1, policy); d != 100*time.Millisecond {
		t.Fatalf("attempt1 delay mismatch: %v", d)
	}
	if d := computeBackoffDelay(2, policy); d != 200*time.Millisecond {
		t.Fatalf("attempt2 delay mismatch: %v", d)
	}
	if d := computeBackoffDelay(3, policy); d != 300*time.Millisecond {
		t.Fatalf("attempt3 delay should cap at max_backoff, got %v", d)
	}
}

func TestLogAndRoundTrip_RedactsSensitiveHeaders(t *testing.T) {
	var loggedOutput strings.Builder
	log.SetOutput(&loggedOutput)
	defer log.SetOutput(os.Stderr)

	mockRT := &mockRoundTripper{}
	req, err := http.NewRequest(http.MethodGet, "https://tenant.example.com/api/test", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	authToken, volterraToken, sessionCookie := testSecret(t), testSecret(t), testSecret(t)
	req.Header.Set("Authorization", "APIToken "+authToken)
	req.Header.Set("X-Volterra-Token", volterraToken)
	req.Header.Set("Cookie", "session="+sessionCookie)

	_, err = logAndRoundTrip(mockRT, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 1. Verify original request was NOT mutated
	if req.Header.Get("Authorization") != "APIToken "+authToken {
		t.Errorf("original Authorization was mutated: %s", req.Header.Get("Authorization"))
	}
	if req.Header.Get("X-Volterra-Token") != volterraToken {
		t.Errorf("original X-Volterra-Token was mutated: %s", req.Header.Get("X-Volterra-Token"))
	}
	if req.Header.Get("Cookie") != "session="+sessionCookie {
		t.Errorf("original Cookie was mutated: %s", req.Header.Get("Cookie"))
	}

	// 2. Verify log output masked the sensitive headers
	out := loggedOutput.String()
	if strings.Contains(out, authToken) {
		t.Errorf("log output contains plaintext token: %s", out)
	}
	if strings.Contains(out, volterraToken) {
		t.Errorf("log output contains plaintext volterra token: %s", out)
	}
	if strings.Contains(out, sessionCookie) {
		t.Errorf("log output contains plaintext cookie: %s", out)
	}
	if !strings.Contains(out, "APIToken [REDACTED]") {
		t.Errorf("log output does not contain redacted token placeholder: %s", out)
	}
}

func TestAPITokenAuth_EnforcesTLS12(t *testing.T) {
	auth := NewAPITokenAuth(testSecret(t), 5*time.Second)
	tr, ok := auth.Client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", auth.Client.Transport)
	}
	if tr.TLSClientConfig == nil || tr.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected MinVersion TLS 1.2, got %v", tr.TLSClientConfig)
	}

	authLazy := &APITokenAuth{Token: testSecret(t)}
	lazyClient := authLazy.GetClient()
	lazyTr, ok := lazyClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", lazyClient.Transport)
	}
	if lazyTr.TLSClientConfig == nil || lazyTr.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected MinVersion TLS 1.2 in GetClient(), got %v", lazyTr.TLSClientConfig)
	}
}

func TestLogAndRoundTrip_RedactsSetCookieInResponse(t *testing.T) {
	var loggedOutput strings.Builder
	log.SetOutput(&loggedOutput)
	defer log.SetOutput(os.Stderr)

	setCookie := testSecret(t)
	respHeader := make(http.Header)
	respHeader.Set("Set-Cookie", "session="+setCookie+"; Path=/")
	respHeader.Set("Content-Type", "application/json")

	mockRT := &mockRoundTripper{
		response: &http.Response{
			StatusCode:    http.StatusOK,
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        respHeader,
			Body:          io.NopCloser(strings.NewReader(`{"status":"ok"}`)),
			ContentLength: int64(len(`{"status":"ok"}`)),
		},
	}
	req, err := http.NewRequest(http.MethodGet, "https://tenant.example.com/api/test", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := logAndRoundTrip(mockRT, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify original response still contains the Set-Cookie for the caller
	if resp.Header.Get("Set-Cookie") != "session="+setCookie+"; Path=/" {
		t.Errorf("original Set-Cookie was modified: %s", resp.Header.Get("Set-Cookie"))
	}

	// Verify log output masked the cookie
	out := loggedOutput.String()
	if strings.Contains(out, setCookie) {
		t.Errorf("log output contains plaintext Set-Cookie secret: %s", out)
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Errorf("log output does not contain [REDACTED]: %s", out)
	}

	// Verify response body is still readable
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if string(bodyBytes) != `{"status":"ok"}` {
		t.Errorf("response body corrupted: got %s, want %s", string(bodyBytes), `{"status":"ok"}`)
	}
}

func TestLogAndRoundTrip_PreservesRequestBody(t *testing.T) {
	mockRT := &mockRoundTripper{
		response: &http.Response{StatusCode: http.StatusOK},
	}
	bodyContent := `{"test":"payload-12345"}`
	req, err := http.NewRequest(http.MethodPost, "https://tenant.example.com/api/test", strings.NewReader(bodyContent))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	_, err = logAndRoundTrip(mockRT, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mockRT.lastReq == nil || mockRT.lastReq.Body == nil {
		t.Fatal("expected mockRT.lastReq.Body to be non-nil")
	}

	receivedBytes, err := io.ReadAll(mockRT.lastReq.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	if string(receivedBytes) != bodyContent {
		t.Fatalf("request body was drained or modified: got %q, want %q", string(receivedBytes), bodyContent)
	}
}

// ---------------------------------------------------------------------------
// PKCS#12 fixtures
// ---------------------------------------------------------------------------

// newTestCert returns a fresh ECDSA key and a certificate for it. When parent
// is nil, the certificate is self-signed and can act as a CA.
func newTestCert(t *testing.T, cn string, parent *x509.Certificate, parentKey *ecdsa.PrivateKey) (*ecdsa.PrivateKey, *x509.Certificate) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  parent == nil,
		BasicConstraintsValid: true,
	}
	signer, signerKey := tmpl, key
	if parent != nil {
		signer, signerKey = parent, parentKey
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, signer, &key.PublicKey, signerKey)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return key, cert
}

// writeTestP12 writes a PKCS#12 bundle to a temp file and returns its path and
// a password generated for this run. When mismatchKey is true, the bundle
// holds a private key that does not match the end-entity certificate.
func writeTestP12(t *testing.T, mismatchKey bool) (string, string) {
	t.Helper()
	caKey, caCert := newTestCert(t, "test-ca", nil, nil)
	leafKey, leafCert := newTestCert(t, "test-client", caCert, caKey)
	if mismatchKey {
		otherKey, _ := newTestCert(t, "other", nil, nil)
		leafKey = otherKey
	}

	password := rand.Text()
	data, err := pkcs12.Modern.Encode(leafKey, leafCert, []*x509.Certificate{caCert}, password)
	require.NoError(t, err)

	path, err := createTempFile(t, data)
	require.NoError(t, err)
	return path, password
}

// TestBuildTLSConfigFromP12_Success verifies that a valid bundle produces a
// TLS 1.2+ config whose client certificate holds the leaf and the CA chain.
func TestBuildTLSConfigFromP12_Success(t *testing.T) {
	path, password := writeTestP12(t, false)

	for _, p := range []string{path, "file:///" + path} {
		cfg, err := buildTLSConfigFromP12(p, password)
		require.NoError(t, err)
		require.Len(t, cfg.Certificates, 1)
		assert.Len(t, cfg.Certificates[0].Certificate, 2, "leaf and CA certificate")
		assert.Equal(t, uint16(0x0303), cfg.MinVersion)
	}
}

// TestBuildTLSConfigFromP12_KeyMismatch verifies that a bundle whose key does
// not match its certificate is rejected.
func TestBuildTLSConfigFromP12_KeyMismatch(t *testing.T) {
	path, password := writeTestP12(t, true)

	cfg, err := buildTLSConfigFromP12(path, password)

	require.ErrorContains(t, err, "failed to create TLS certificate from p12")
	assert.Nil(t, cfg)
}

// TestNewP12CertAuth_Success verifies that a valid bundle yields an
// authenticator with a configured client and no token data.
func TestNewP12CertAuth_Success(t *testing.T) {
	path, password := writeTestP12(t, false)

	auth, err := NewP12CertAuth(path, password, 7*time.Second)

	require.NoError(t, err)
	assert.Equal(t, path, auth.P12Path)
	require.NotNil(t, auth.GetClient())
	assert.Equal(t, 7*time.Second, auth.GetClient().Timeout)
	assert.Empty(t, auth.GetClientData())
}

// ---------------------------------------------------------------------------
// providerBinaryInfo
// ---------------------------------------------------------------------------

const binaryInfoHelperEnv = "F5XC_CLIENT_BINARY_INFO_HELPER"

// TestProviderBinaryInfoHelper prints providerBinaryInfo for the running
// binary. It runs only as a child process of TestProviderBinaryInfo.
func TestProviderBinaryInfoHelper(t *testing.T) {
	if os.Getenv(binaryInfoHelperEnv) != "1" {
		t.Skip("helper process only")
	}
	name, version := providerBinaryInfo()
	fmt.Printf("BINARY_INFO=%s|%s\n", name, version)
}

// TestProviderBinaryInfo verifies how the provider name and version are taken
// from the binary file name first and from its directory path second.
func TestProviderBinaryInfo(t *testing.T) {
	exe, err := os.Executable()
	require.NoError(t, err)

	tests := []struct {
		name        string
		relPath     string
		wantName    string
		wantVersion string
	}{
		{
			name:        "name and version from file name",
			relPath:     "terraform-provider-f5xc_v1.2.3.exe",
			wantName:    "terraform-provider-f5xc",
			wantVersion: "1.2.3",
		},
		{
			name:        "version from directory path",
			relPath:     filepath.Join("9.8.7", "terraform-provider-acme"),
			wantName:    "terraform-provider-acme",
			wantVersion: "9.8.7",
		},
		{
			name:        "empty base name falls back to defaults",
			relPath:     ".test",
			wantName:    defaultProviderName,
			wantVersion: defaultProviderVersion,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			target := filepath.Join(t.TempDir(), tc.relPath)
			require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o750))
			copyExecutable(t, exe, target)

			args := []string{"-test.run=^TestProviderBinaryInfoHelper$"}
			// Child counters land in the parent's cover dir, so the parent
			// profile includes the branches that the child runs.
			if f := flag.Lookup("test.gocoverdir"); f != nil && f.Value.String() != "" {
				args = append(args, "-test.gocoverdir="+f.Value.String())
			}
			// #nosec G204 -- runs a copy of this test binary
			cmd := exec.Command(target, args...)
			cmd.Env = append(os.Environ(), binaryInfoHelperEnv+"=1")
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, string(out))

			assert.Contains(t, string(out), fmt.Sprintf("BINARY_INFO=%s|%s\n", tc.wantName, tc.wantVersion))
		})
	}
}

// copyExecutable copies the file src to dst. It does not hard-link because
// macOS caches code-signing state per inode and can kill a second binary
// that shares the inode of a running process.
func copyExecutable(t *testing.T, src, dst string) {
	t.Helper()
	// #nosec G304 -- src is this test binary
	data, err := os.ReadFile(src)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(dst, data, 0o700)) // #nosec G306 -- executable test copy
}

// ---------------------------------------------------------------------------
// APITokenAuth / AuthClient.Execute
// ---------------------------------------------------------------------------

// TestAPITokenAuth_GetClient_Lazy verifies that a token authenticator without
// a client builds one once and reuses it.
func TestAPITokenAuth_GetClient_Lazy(t *testing.T) {
	auth := &APITokenAuth{Token: testSecret(t)}

	first := auth.GetClient()

	require.NotNil(t, first)
	assert.Equal(t, 10*time.Second, first.Timeout)
	assert.Same(t, first, auth.GetClient())
}

// headerAuth sets the given headers on each request.
type headerAuth struct {
	mockAuthenticator
	headers map[string]string
}

func (h *headerAuth) ApplyAuth(req *http.Request) error {
	for k, v := range h.headers {
		req.Header.Set(k, v)
	}
	return nil
}

// TestAuthClient_Execute_DefaultUserAgentAndSensitiveHeaders verifies that an
// empty UserAgent falls back to the provider default, and that token and
// cookie headers still reach the server.
func TestAuthClient_Execute_DefaultUserAgentAndSensitiveHeaders(t *testing.T) {
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()

	token, cookie := testSecret(t), "c="+testSecret(t)
	auth := &headerAuth{headers: map[string]string{"X-Volterra-Token": token, "Cookie": cookie}}
	ac := NewAuthClient(srv.URL, auth, "")

	body, err := ac.Execute(context.Background(), http.MethodGet, srv.URL, nil)

	require.NoError(t, err)
	assert.Equal(t, "{}", string(body))
	assert.Equal(t, providerUserAgent(""), got.Get("User-Agent"))
	assert.Equal(t, token, got.Get("X-Volterra-Token"))
	assert.Equal(t, cookie, got.Get("Cookie"))
}

// TestAuthClient_Execute_TransportError verifies that a transport failure is
// returned as an execution error.
func TestAuthClient_Execute_TransportError(t *testing.T) {
	rt := &mockRoundTripper{err: errors.New("dial failed")}
	ac := NewAuthClient("https://x.test", &mockAuthenticator{client: &http.Client{Transport: rt}}, "ua")

	body, err := ac.Execute(context.Background(), http.MethodGet, "https://x.test/api", nil)

	require.ErrorContains(t, err, "error executing request")
	assert.Nil(t, body)
}

// errBody is a response body whose Read and Close can fail.
type errBody struct {
	io.Reader
	closeErr error
}

func (b errBody) Close() error { return b.closeErr }

// TestAuthClient_Execute_BodyErrors verifies that a body read failure is
// returned and that a body close failure does not hide a good response.
func TestAuthClient_Execute_BodyErrors(t *testing.T) {
	tests := []struct {
		name    string
		body    io.ReadCloser
		wantErr string
	}{
		{name: "read error", body: errBody{Reader: failingReader{}}, wantErr: "error reading response body"},
		{name: "close error", body: errBody{Reader: strings.NewReader("ok"), closeErr: errors.New("close failed")}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rt := &mockRoundTripper{response: &http.Response{StatusCode: http.StatusOK, Body: tc.body}}
			ac := NewAuthClient("https://x.test", &mockAuthenticator{client: &http.Client{Transport: rt}}, "ua")

			body, err := ac.Execute(context.Background(), http.MethodGet, "https://x.test/api", nil)

			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "ok", string(body))
		})
	}
}

// TestAuthClient_Execute_NonOKEmptyBody verifies that a non-OK status with an
// empty body returns an error that names only the status.
func TestAuthClient_Execute_NonOKEmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	ac := NewAuthClient(srv.URL, &mockAuthenticator{}, "ua")

	_, err := ac.Execute(context.Background(), http.MethodGet, srv.URL, nil)

	require.EqualError(t, err, "API call returned status 404 Not Found")
}

// ---------------------------------------------------------------------------
// APIClient CRUD wrappers
// ---------------------------------------------------------------------------

// TestAPIClient_Wrappers_PropagateErrors verifies that each wrapper passes its
// call options to the underlying vesapi client and returns its error.
func TestAPIClient_Wrappers_PropagateErrors(t *testing.T) {
	ctx := context.Background()
	optErr := errors.New("bad option")
	failOpt := vesapi.CallOpt(func(*vesapi.CallOpts) error { return optErr })
	// The server fails the test if it is called: each wrapper must stop on
	// the option error before it sends a request.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	// The metadata registry must be set: without it, the vesapi object calls
	// return an error before they evaluate the call options.
	vc, err := vesapi.NewAPIClient(srv.URL+"/api", ves_io_schema_combined.MDR, vesapi.WithCfgApiToken(testSecret(t)))
	require.NoError(t, err)
	c := &APIClient{APIClient: vc}

	calls := map[string]func() error{
		"CreateObject": func() error {
			_, err := c.CreateObject(ctx, "ves.io.schema.namespace.Object", nil, failOpt)
			return err
		},
		"GetObject": func() error {
			_, err := c.GetObject(ctx, "ves.io.schema.namespace.Object", "ns", "n", failOpt)
			return err
		},
		"ReplaceObject": func() error {
			return c.ReplaceObject(ctx, "ves.io.schema.namespace.Object", nil, failOpt)
		},
		"DeleteObject": func() error {
			return c.DeleteObject(ctx, "ves.io.schema.namespace.Object", "ns", "n", failOpt)
		},
		"ListObjects": func() error {
			_, err := c.ListObjects(ctx, "ves.io.schema.namespace.Object", "ns", failOpt)
			return err
		},
		"CustomAPI": func() error {
			_, err := c.CustomAPI(ctx, http.MethodGet, "/x", "ves.io.schema.namespace.CustomAPI.Get", "", failOpt)
			return err
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			require.ErrorIs(t, call(), optErr)
		})
	}
}

// ---------------------------------------------------------------------------
// logAndRoundTrip / prismURIRemapperFn
// ---------------------------------------------------------------------------

// TestLogAndRoundTrip_BodyWithoutGetBody verifies that a body without GetBody
// is buffered once, so the request can still be replayed through GetBody.
func TestLogAndRoundTrip_BodyWithoutGetBody(t *testing.T) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)

	req, err := http.NewRequest(http.MethodPost, "https://x.test/api", io.NopCloser(strings.NewReader("payload")))
	require.NoError(t, err)
	require.Nil(t, req.GetBody)

	_, err = logAndRoundTrip(&mockRoundTripper{}, req)
	require.NoError(t, err)

	rc, err := req.GetBody()
	require.NoError(t, err)
	replayed, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "payload", string(replayed))
}

// TestLogAndRoundTrip_UnreadableBody verifies that a body read failure sends
// an empty body instead of failing the request.
func TestLogAndRoundTrip_UnreadableBody(t *testing.T) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)

	req, err := http.NewRequest(http.MethodPost, "https://x.test/api", io.NopCloser(failingReader{}))
	require.NoError(t, err)
	rt := &mockRoundTripper{}

	_, err = logAndRoundTrip(rt, req)

	require.NoError(t, err)
	sent, err := io.ReadAll(rt.lastReq.Body)
	require.NoError(t, err)
	assert.Empty(t, sent)
}

// TestLogAndRoundTrip_TransportError verifies that a transport error is
// logged and returned unchanged.
func TestLogAndRoundTrip_TransportError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	req, err := http.NewRequest(http.MethodGet, "https://x.test/api", nil)
	require.NoError(t, err)
	wantErr := errors.New("connection reset")

	rsp, err := logAndRoundTrip(&mockRoundTripper{err: wantErr}, req)

	assert.Nil(t, rsp)
	require.ErrorIs(t, err, wantErr)
	assert.Contains(t, buf.String(), "Response Error")
}

// TestPrismURIRemapperFn_InvalidRemappedURL verifies that a slug that makes
// the remapped URL unparsable returns an error and sends nothing.
func TestPrismURIRemapperFn_InvalidRemappedURL(t *testing.T) {
	policy, err := resolveThrottlingPolicy(nil)
	require.NoError(t, err)
	fn, err := prismURIRemapperFn("https://tenant.example.com/api", "ua", policy, newResourceThrottleRegistry())
	require.NoError(t, err)

	ctx := contextFromServiceSlug(context.Background(), "%zz")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://tenant.example.com/api/public/namespaces/default/x", nil)
	require.NoError(t, err)
	rt := &mockRoundTripper{}

	_, err = fn(req, rt)

	require.ErrorContains(t, err, "Parsing remapped url")
	assert.Nil(t, rt.lastReq)
}

// ---------------------------------------------------------------------------
// getClientOpts / NewVesAPIClient
// ---------------------------------------------------------------------------

// TestGetClientOpts verifies which options each provider setting adds, and
// that invalid URL, timeout, and P12 settings fail.
func TestGetClientOpts(t *testing.T) {
	p12Path, p12Password := writeTestP12(t, false)
	policy, err := resolveThrottlingPolicy(nil)
	require.NoError(t, err)

	tests := []struct {
		name    string
		cfg     ProviderConfig
		wantLen int
		wantErr string
	}{
		{name: "invalid url", cfg: ProviderConfig{Url: "://bad", Timeout: "1s"}, wantErr: "Parsing apigw url"},
		{name: "invalid timeout", cfg: ProviderConfig{Url: "https://x.test/api", Timeout: "soon"}, wantErr: "invalid duration"},
		{name: "minimal", cfg: ProviderConfig{Url: "https://x.test/api", Timeout: "1s"}, wantLen: 2},
		{name: "vesenv", cfg: ProviderConfig{Url: "https://x.test/api", Timeout: "1s", Vesenv: true, tenant: "acme"}, wantLen: 2},
		{
			name: "all credentials",
			cfg: ProviderConfig{
				Url: "https://x.test/api", Timeout: "1s",
				ApiCert: "cert.pem", ApiKey: "key.pem", ApiCACert: "ca.pem", ApiToken: testSecret(t),
				ApiP12File: p12Path, ApiP12Password: p12Password,
			},
			wantLen: 7,
		},
		{
			name:    "invalid p12",
			cfg:     ProviderConfig{Url: "https://x.test/api", Timeout: "1s", ApiP12File: filepath.Join(t.TempDir(), "missing.p12"), ApiP12Password: testSecret(t)},
			wantErr: "Building TLS configuration",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts, err := getClientOpts(&tc.cfg, policy, newResourceThrottleRegistry())
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				assert.Nil(t, opts)
				return
			}
			require.NoError(t, err)
			assert.Len(t, opts, tc.wantLen)
		})
	}
}

// TestUriRemapperFn_TenantHeaderMatchesServer verifies that the Vesenv
// remapper sends the tenant header that the server reads into its context.
func TestUriRemapperFn_TenantHeaderMatchesServer(t *testing.T) {
	policy, err := resolveThrottlingPolicy(nil)
	require.NoError(t, err)
	fn := uriRemapperFn("acme", "ua", policy, newResourceThrottleRegistry())
	req, err := http.NewRequest(http.MethodGet, "https://x.test/api", nil)
	require.NoError(t, err)
	rt := &mockRoundTripper{}

	_, err = fn(req, rt)

	require.NoError(t, err)
	ctx := server.ContextFromRestHdrs(context.Background(), rt.lastReq.Header)
	assert.Equal(t, "acme", server.TenantFromContext(ctx))
}

// TestNewVesAPIClient verifies client construction for token and P12
// authentication, and the errors for invalid throttling, timeout, and test
// TLS settings.
func TestNewVesAPIClient(t *testing.T) {
	p12Path, p12Password := writeTestP12(t, false)
	prevCN := vesapi.ServerCN
	t.Cleanup(func() { vesapi.ServerCN = prevCN })

	base := func() ProviderConfig {
		return ProviderConfig{
			Url:           "https://tenant.example.com/api",
			Timeout:       "5s",
			TimeoutCreate: "1m",
			TimeoutRead:   "2m",
			TimeoutUpdate: "3m",
			TimeoutDelete: "4m",
		}
	}

	t.Run("invalid throttling", func(t *testing.T) {
		cfg := base()
		cfg.Throttling = &ThrottlingConfig{MaxRetries: -1}
		_, err := NewVesAPIClient(&cfg)
		require.ErrorContains(t, err, "Building throttling policy")
	})

	t.Run("invalid timeout", func(t *testing.T) {
		cfg := base()
		cfg.Timeout = "soon"
		_, err := NewVesAPIClient(&cfg)
		require.ErrorContains(t, err, "Building client options")
	})

	t.Run("test mode without fixtures", func(t *testing.T) {
		cfg := base()
		cfg.Test = true
		_, err := NewVesAPIClient(&cfg)
		require.ErrorContains(t, err, "Creating new F5XC APIClient")
	})

	t.Run("no credentials", func(t *testing.T) {
		cfg := base()
		_, err := NewVesAPIClient(&cfg)
		require.ErrorContains(t, err, "Creating new F5XC APIClient")
	})

	t.Run("api token", func(t *testing.T) {
		cfg := base()
		cfg.ApiToken = testSecret(t)
		cfg.Throttling = &ThrottlingConfig{MaxRetries: 1}
		c, err := NewVesAPIClient(&cfg)
		require.NoError(t, err)
		assert.NotNil(t, c.APIClient)
		assert.Equal(t, &ClientTimeouts{TimeoutCreate: "1m", TimeoutRead: "2m", TimeoutUpdate: "3m", TimeoutDelete: "4m"}, c.Config)
		require.NotNil(t, c.RestClient)
		assert.IsType(t, &APITokenAuth{}, c.RestClient.Authenticator)
		assert.True(t, c.RestClient.ThrottlePolicy.Enabled)
		assert.NotNil(t, c.RestClient.ThrottleRegistry)
	})

	t.Run("p12", func(t *testing.T) {
		cfg := base()
		cfg.ApiP12File = p12Path
		cfg.ApiP12Password = p12Password
		c, err := NewVesAPIClient(&cfg)
		require.NoError(t, err)
		require.NotNil(t, c.RestClient)
		assert.IsType(t, &P12CertAuth{}, c.RestClient.Authenticator)
	})
}

// ---------------------------------------------------------------------------
// throttling edge cases
// ---------------------------------------------------------------------------

// TestComputeBackoffDelay_NegativeClampsToZero verifies that a negative
// computed delay is clamped to zero.
func TestComputeBackoffDelay_NegativeClampsToZero(t *testing.T) {
	policy := throttlingPolicy{InitialBackoff: -time.Second, MaxBackoff: time.Second, Multiplier: 1}

	assert.Zero(t, computeBackoffDelay(1, policy))
}

// TestDoRequestWithThrottling_NegativeRetriesSendsOnce verifies that a
// negative retry count still sends the request exactly once.
func TestDoRequestWithThrottling_NegativeRetriesSendsOnce(t *testing.T) {
	sends := 0
	resp, err := doRequestWithThrottling(context.Background(), newTestRequest(t, http.MethodGet, nil), "sites",
		fastRetryPolicy(-5), newResourceThrottleRegistry(), func(*http.Request) (*http.Response, error) {
			sends++
			return statusResponse(http.StatusTooManyRequests), nil
		})

	require.NoError(t, err)
	defer closeResponseBody(resp)
	assert.Equal(t, 1, sends)
}

// TestDoRequestWithThrottling_CloneError verifies that a body that cannot be
// rebuilt for the first attempt fails before any send.
func TestDoRequestWithThrottling_CloneError(t *testing.T) {
	req := newTestRequest(t, http.MethodPut, strings.NewReader("x"))
	req.GetBody = func() (io.ReadCloser, error) { return nil, errors.New("body gone") }
	sends := 0

	resp, err := doRequestWithThrottling(context.Background(), req, "sites",
		fastRetryPolicy(1), newResourceThrottleRegistry(), func(*http.Request) (*http.Response, error) {
			sends++
			return statusResponse(http.StatusOK), nil
		})

	require.EqualError(t, err, "body gone")
	assert.Nil(t, resp)
	assert.Zero(t, sends)
}
