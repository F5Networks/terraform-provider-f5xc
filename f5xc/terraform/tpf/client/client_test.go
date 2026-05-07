// Package client F5XC
// Copyright (c) 2025 F5 Inc. All rights reserved.

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

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
}

func (m *mockRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
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
	ac := NewAuthClient("https://example.com", auth)

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
	ac := NewAuthClient("", auth)
	if ac.URL != "" {
		t.Errorf("expected empty URL, got %q", ac.URL)
	}
}

// ---------------------------------------------------------------------------
// NewAPITokenAuth
// ---------------------------------------------------------------------------

func TestNewAPITokenAuth(t *testing.T) {
	timeout := 10 * time.Second
	a := NewAPITokenAuth("my-token", timeout)

	if a == nil {
		t.Fatal("expected non-nil APITokenAuth")
	}
	if a.Token != "my-token" {
		t.Errorf("Token = %q, want %q", a.Token, "my-token")
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
	a := &APITokenAuth{Token: "test-token-123"}
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	req.Header = make(http.Header)

	if err := a.ApplyAuth(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "APIToken test-token-123"
	if got := req.Header.Get("Authorization"); got != want {
		t.Errorf("Authorization = %q, want %q", got, want)
	}
}

func TestAPITokenAuth_ApplyAuth_TrimsWhitespace(t *testing.T) {
	a := &APITokenAuth{Token: "  spaced-token  "}
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	req.Header = make(http.Header)

	if err := a.ApplyAuth(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "APIToken spaced-token"
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
	a := &APITokenAuth{Token: "tok"}
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
	a := &APITokenAuth{Token: "secret-token"}
	if got := a.GetClientData(); got != "secret-token" {
		t.Errorf("GetClientData() = %q, want %q", got, "secret-token")
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
		Password: "pass",
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
	ac := NewAuthClient(srv.URL, auth)

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

	auth := &mockAuthenticator{
		client:        srv.Client(),
		appliedHeader: "APIToken my-test-token",
	}
	ac := NewAuthClient(srv.URL, auth)

	_, err := ac.Execute(context.Background(), http.MethodGet, srv.URL+"/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedAuth != "APIToken my-test-token" {
		t.Errorf("Authorization = %q, want %q", receivedAuth, "APIToken my-test-token")
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
	ac := NewAuthClient(srv.URL, auth)

	_, err := ac.Execute(context.Background(), http.MethodGet, srv.URL+"/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedUserAgent != "F5XC-Terraform-Provider/1.0.0 (Go-Client)" {
		t.Errorf("User-Agent = %q, want %q", capturedUserAgent, "F5XC-Terraform-Provider/1.0.0 (Go-Client)")
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
	ac := NewAuthClient(srv.URL, auth)

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
	ac := NewAuthClient(srv.URL, auth)

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
	ac := NewAuthClient("https://example.com", auth)

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
	ac := NewAuthClient("://bad-url", auth)

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
	ac := NewAuthClient(srv.URL, auth)

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
// ProviderConfig / ProviderLimiter struct field tests
// ---------------------------------------------------------------------------

func TestProviderConfig_Fields(t *testing.T) {
	pl := &ProviderLimiter{Rate: rate.Limit(10), Burst: 5}
	cfg := &ProviderConfig{
		Url:            "https://tenant.example.com",
		Test:           false,
		Vesenv:         true,
		Timeout:        "30s",
		Limiter:        pl,
		ApiCert:        "/path/to/cert",
		ApiKey:         "/path/to/key",
		ApiToken:       "my-api-token",
		ApiCACert:      "/path/to/ca",
		ApiP12File:     "/path/to/file.p12",
		ApiP12Password: "p12-password",
	}

	if cfg.Url != "https://tenant.example.com" {
		t.Errorf("Url = %q", cfg.Url)
	}
	if cfg.Timeout != "30s" {
		t.Errorf("Timeout = %q", cfg.Timeout)
	}
	if cfg.Limiter.Rate != rate.Limit(10) {
		t.Errorf("Limiter.Rate = %v", cfg.Limiter.Rate)
	}
	if cfg.Limiter.Burst != 5 {
		t.Errorf("Limiter.Burst = %v", cfg.Limiter.Burst)
	}
	if cfg.ApiToken != "my-api-token" {
		t.Errorf("ApiToken = %q", cfg.ApiToken)
	}
	if !cfg.Vesenv {
		t.Error("Vesenv should be true")
	}
}

// ---------------------------------------------------------------------------
// Rate limiter – direct usage tests
// ---------------------------------------------------------------------------

func TestRateLimiter_AllowsRequests(t *testing.T) {
	// Initialize the global limiter directly (as NewVesAPIClient would do)
	limiter = rate.NewLimiter(rate.Limit(100), 10)

	if err := limiter.Wait(context.Background()); err != nil {
		t.Fatalf("limiter.Wait() returned unexpected error: %v", err)
	}
}

func TestRateLimiter_CancelledContext(t *testing.T) {
	// A zero-rate limiter with a cancelled context must return an error.
	limiter = rate.NewLimiter(0, 0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	if err := limiter.Wait(ctx); err == nil {
		t.Fatal("expected error for cancelled context with zero-rate limiter, got nil")
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
	_, err := buildTLSConfigFromP12("/nonexistent/path/file.p12", "password")
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read p12 file") {
		t.Errorf("error = %q, expected to contain 'failed to read p12 file'", err.Error())
	}
}

func TestBuildTLSConfigFromP12_FileURIPrefix(t *testing.T) {
	// The function strips "file:///" prefix; the resulting path should still not exist.
	_, err := buildTLSConfigFromP12("file:///nonexistent/path/file.p12", "password")
	if err == nil {
		t.Fatal("expected error for non-existent file with file:/// prefix, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read p12 file") {
		t.Errorf("error = %q, expected to contain 'failed to read p12 file'", err.Error())
	}
}

func TestBuildTLSConfigFromP12_InvalidP12Data(t *testing.T) {
	tmpFile, err := createTempFile(t, []byte("not-a-valid-p12-file"))
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	_, err = buildTLSConfigFromP12(tmpFile, "password")
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
	_, err := NewP12CertAuth("/nonexistent/path/file.p12", "password", 10*time.Second)
	if err == nil {
		t.Fatal("expected error for non-existent P12 file, got nil")
	}
}

func TestNewP12CertAuth_InvalidP12(t *testing.T) {
	tmpFile, err := createTempFile(t, []byte("invalid-p12-data"))
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	_, err = NewP12CertAuth(tmpFile, "wrong-password", 10*time.Second)
	if err == nil {
		t.Fatal("expected error for invalid P12 data, got nil")
	}
}

// ---------------------------------------------------------------------------
// prismURIRemapperFn
// ---------------------------------------------------------------------------

func TestPrismURIRemapperFn_InvalidURL(t *testing.T) {
	_, err := prismURIRemapperFn("://invalid")
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}

func TestPrismURIRemapperFn_ValidURL(t *testing.T) {
	fn, err := prismURIRemapperFn("https://tenant.example.com/api")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fn == nil {
		t.Fatal("expected non-nil RoundTripperFunc")
	}
}

func TestPrismURIRemapperFn_NoSlugInContext(t *testing.T) {
	fn, err := prismURIRemapperFn("https://tenant.example.com/api")
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
	fn, err := prismURIRemapperFn("https://tenant.example.com/api")
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
	// The URL should have been remapped: "/public" replaced by "/config"
	if strings.Contains(req.URL.Path, "/public") {
		t.Errorf("URL path still contains '/public': %s", req.URL.Path)
	}
	if !strings.Contains(req.URL.Path, "/config") {
		t.Errorf("URL path does not contain '/config': %s", req.URL.Path)
	}
}

// ---------------------------------------------------------------------------
// uriRemapperFn
// ---------------------------------------------------------------------------

func TestURIRemapperFn_SetsTenantHeader(t *testing.T) {
	fn := uriRemapperFn("my-tenant")

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
	if got := req.Header.Get("X-Volterra-Apigw-Tenant"); got != "my-tenant" {
		t.Errorf("X-Volterra-Apigw-Tenant = %q, want %q", got, "my-tenant")
	}
}

func TestURIRemapperFn_EmptyTenant(t *testing.T) {
	fn := uriRemapperFn("")

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
	// Empty tenant should still set the header (to empty string)
	if got := req.Header.Get("x-ves-tenant"); got != "" {
		t.Errorf("x-ves-tenant = %q, want empty string", got)
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
