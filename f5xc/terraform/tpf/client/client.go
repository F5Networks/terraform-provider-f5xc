// Package client F5XC
// Copyright (c) 2025 F5 Inc. All rights reserved.

package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	ves_io_schema_combined "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/combined"
	"software.sslmate.com/src/go-pkcs12"

	"github.com/gogo/protobuf/proto"
	"github.com/moul/http2curl"

	stdlibClient "gopkg.volterra.us/stdlib/client"
	"gopkg.volterra.us/stdlib/client/vesapi"
	"gopkg.volterra.us/stdlib/errors"
	"gopkg.volterra.us/stdlib/server"
)

var (
	// providerBinaryVersionRE extracts X.Y.Z from provider binaries or install paths.
	providerBinaryVersionRE = regexp.MustCompile(`(\d+\.\d+\.\d+)`)
	// providerBinaryNameRE extracts terraform-provider-<name> from the binary name.
	providerBinaryNameRE    = regexp.MustCompile(`^(terraform-provider-[^_]+)`)
	defaultProviderName     = "terraform-provider-f5xc"
	defaultProviderVersion  = "dev"
	defaultTerraformVersion = "unknown"
)

// Authenticator is the Strategy interface.
// All authentication methods must implement this.
type Authenticator interface {
	ApplyAuth(req *http.Request) error
	GetClient() *http.Client
	GetClientData() string
}

// AuthClient is the Context structure.
type AuthClient struct {
	URL              string
	Authenticator    Authenticator
	UserAgent        string
	ThrottlePolicy   throttlingPolicy
	ThrottleRegistry *ResourceThrottleRegistry
}

// APITokenAuth is a Concrete Strategy for API Token authentication.
type APITokenAuth struct {
	Token  string
	Client *http.Client // Storing the configured client
}

// P12CertAuth is a Concrete Strategy for P12 certificate authentication (mTLS).
type P12CertAuth struct {
	P12Path  string
	Password string
	Client   *http.Client // The client must be pre-configured for P12
}

// ClientTimeouts holds per-operation timeout overrides for CRUD operations.
// Each field is a Go duration string (e.g. "10m", "1h30m"). An empty string
// means "use the resource-level default".
type ClientTimeouts struct {
	TimeoutCreate string // Default timeout for Create operations.
	TimeoutRead   string // Default timeout for Read operations.
	TimeoutUpdate string // Default timeout for Update operations.
	TimeoutDelete string // Default timeout for Delete operations.
}

// ProviderConfig data for f5xc provider
type ProviderConfig struct {
	Url              string            // The base URL for the F5XC API endpoint.
	Test             bool              // Indicates whether the provider is running in test mode.
	TerraformVersion string            // The Terraform CLI version driving provider Configure.
	ProviderVersion  string            // The provider version injected at build time (e.g., "2.1.0" or "dev").
	Vesenv           bool              // Specifies if the provider is running in a Ves environment.
	tenant           string            // The tenant identifier for API requests.
	Timeout          string            // The timeout duration for API requests, as a string (e.g., "30s").
	TimeoutCreate    string            // Default timeout for Create operations (e.g. "10m"). Falls back to resource default if empty.
	TimeoutRead      string            // Default timeout for Read operations (e.g. "5m"). Falls back to resource default if empty.
	TimeoutUpdate    string            // Default timeout for Update operations (e.g. "10m"). Falls back to resource default if empty.
	TimeoutDelete    string            // Default timeout for Delete operations (e.g. "10m"). Falls back to resource default if empty.
	Throttling       *ThrottlingConfig // Backend-driven request throttling configuration.
	ApiCert          string            // The path to the API client certificate file.
	ApiKey           string            // The path to the API client key file.
	ApiToken         string            // The API token used for authentication.
	ApiCACert        string            // The path to the CA certificate file for API requests.
	ApiP12File       string            // The path to the PKCS#12 file containing client credentials.
	ApiP12Password   string            // The password for the PKCS#12 file.
}

// APIClient embeds vesapi.APIClient
type APIClient struct {
	APIClient  *vesapi.APIClient
	RestClient *AuthClient
	Config     *ClientTimeouts // Per-operation timeout overrides propagated from the provider configuration.
}

// ResourceF5XCApiDefinitionUploaderMetadataApiResponse represents metadata information for an F5XC API definition upload.
// This struct contains details such as the URL, name, version, namespace, description, and creation timestamp of the uploaded API definition.
type ResourceF5XCApiDefinitionUploaderMetadataApiResponse struct {
	Url               string `json:"url"`               // The URL where the API definition is accessible or stored.
	Name              string `json:"name"`              // The human-readable name of the API definition.
	Version           string `json:"version"`           // The version identifier for the API definition.
	Namespace         string `json:"namespace"`         // The namespace in which the API definition resides.
	Description       string `json:"description"`       // A description providing additional information about the API definition.
	CreationTimestamp string `json:"creationTimestamp"` // The timestamp when the API definition was created.
}

// ResourceF5XCApiDefinitionUploaderApiModel describes the API data model.
type ResourceF5XCApiDefinitionUploaderApiModel struct {
	Metadata    ResourceF5XCApiDefinitionUploaderMetadataApiResponse `json:"metadata"`
	StringValue string                                               `json:"string_value"`
}

// ResourceF5XCApiDefinitionUploaderCreateRequest represents the PUT request payload for F5 XC Object Store CreateObject.
// Unlike CreateObjectResponse and GetObjectResponse (which nest StoredObjectDescriptor under metadata),
// ves.io.schema.stored_object.CreateObjectRequest accepts description as a top-level JSON field and
// accepts either string_value or bytes_value (Base64) under the contents oneof.
type ResourceF5XCApiDefinitionUploaderCreateRequest struct {
	Description string `json:"description,omitempty"`  // Human-readable description for the stored object version.
	StringValue string `json:"string_value,omitempty"` // Raw OpenAPI specification content.
	BytesValue  string `json:"bytes_value,omitempty"`  // Base64-encoded OpenAPI specification content.
}

// Supported CRUD operations
const (
	Create = iota
	Delete
	Replace
	Get
)

// NewAuthClient creates a new client with a specified authentication strategy.
func NewAuthClient(url string, auth Authenticator, userAgent string) *AuthClient {
	return &AuthClient{
		URL:           url,
		Authenticator: auth,
		UserAgent:     userAgent,
	}
}

// providerBinaryInfo extracts the provider name and version from the running binary path.
//
// Why: The Terraform provider needs to identify itself in User-Agent headers and logs.
// The binary name and path may contain version information that helps track which
// provider version is executing a request.
//
// How: It attempts to extract the provider name from the binary filename using the
// providerBinaryNameRE regex. It then looks for version strings matching the pattern
// X.Y.Z in both the binary name and the full executable path. If extraction fails at
// any step, it falls back to default constants.
//
// Parameters: none
//
// Returns:
//   - string: the provider name (e.g., "terraform-provider-f5xc"), or defaultProviderName
//   - string: the provider version (e.g., "1.2.3"), or defaultProviderVersion
//
// Example:
//
//	name, version := providerBinaryInfo()
//	// name: "terraform-provider-f5xc", version: "1.0.0"
func providerBinaryInfo() (string, string) {
	execPath, err := os.Executable()
	if err != nil {
		return defaultProviderName, defaultProviderVersion
	}

	binaryName := strings.TrimSuffix(filepath.Base(execPath), filepath.Ext(execPath))
	if binaryName == "" {
		return defaultProviderName, defaultProviderVersion
	}

	name := defaultProviderName
	if matches := providerBinaryNameRE.FindStringSubmatch(binaryName); len(matches) > 1 {
		name = matches[1]
	}

	if matches := providerBinaryVersionRE.FindStringSubmatch(binaryName); len(matches) > 1 {
		return name, matches[1]
	}

	if matches := providerBinaryVersionRE.FindStringSubmatch(execPath); len(matches) > 1 {
		return name, matches[1]
	}

	return name, defaultProviderVersion
}

// providerUserAgent constructs a User-Agent header value for API requests using binary path heuristics.
//
// Why: HTTP User-Agent headers identify the client making requests to the API.
// This is essential for server-side logging, monitoring, and debugging to know
// which version of the provider is making requests.
//
// How: It delegates to providerUserAgentWithVersion with an empty configured
// provider version so that binary path extraction is used.
//
// Parameters:
//   - terraformVersion: the Terraform CLI version string (may be empty or contain whitespace)
//
// Returns:
//   - string: a User-Agent string suitable for HTTP request headers
//
// Example:
//
//	userAgent := providerUserAgent("1.5.0")
//	// Returns: "terraform/1.5.0 terraform-provider-f5xc/1.0.0"
func providerUserAgent(terraformVersion string) string {
	return providerUserAgentWithVersion(terraformVersion, "")
}

// providerUserAgentWithVersion constructs a User-Agent header value preferring an explicit provider version.
//
// Why: When the provider version is injected at build time via -ldflags, using it
// directly is more reliable than parsing the executable file path, which fails in
// local development builds, test binaries, or custom plugin cache layouts.
//
// How: It normalizes terraformVersion (falling back to defaultTerraformVersion if empty)
// and queries providerBinaryInfo() for the binary name and path-derived version. If
// configuredProviderVersion is non-empty and not equal to defaultProviderVersion ("dev"),
// it strips any leading "v" prefix and overrides the path-derived version.
//
// Parameters:
//   - terraformVersion:          the Terraform CLI version string
//   - configuredProviderVersion: the build-time provider version from ProviderConfig
//
// Returns:
//   - string: formatted User-Agent header ("terraform/<tfVersion> <providerName>/<providerVersion>")
//
// Example:
//
//	ua := providerUserAgentWithVersion("1.8.0", "v2.1.0")
//	// ua == "terraform/1.8.0 terraform-provider-f5xc/2.1.0"
func providerUserAgentWithVersion(terraformVersion, configuredProviderVersion string) string {
	terraformVersion = strings.TrimSpace(terraformVersion)
	if terraformVersion == "" {
		terraformVersion = defaultTerraformVersion
	}
	providerName, providerVersion := providerBinaryInfo()
	configured := strings.TrimPrefix(strings.TrimSpace(configuredProviderVersion), "v")
	if configured != "" && configured != defaultProviderVersion {
		providerVersion = configured
	}
	return fmt.Sprintf("terraform/%s %s/%s", terraformVersion, providerName, providerVersion)
}

// NewAPITokenAuth creates and initializes a new APITokenAuth instance for API token-based authentication.
// Parameters:
//   - token: The API token string used for authentication.
//   - timeout: The timeout duration for HTTP requests made by the client.
//
// Returns:
//   - *APITokenAuth: A pointer to the initialized APITokenAuth struct, ready for use in authenticated API requests.
func NewAPITokenAuth(token string, timeout time.Duration) *APITokenAuth {
	return &APITokenAuth{
		Token: token,
		Client: &http.Client{
			Transport: wrapTransportForVCR(&http.Transport{
				Proxy: http.ProxyFromEnvironment,
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
				},
			}),
			Timeout: timeout,
		},
	}
}

// buildTLSConfigFromP12 decodes a PKCS#12 file and returns a *tls.Config suitable for mTLS.
// It uses pkcs12.DecodeChain which accepts P12 files with certificate chains (1-2 content infos),
// unlike pkcs12.ToPEM (software.sslmate.com v0.4.0) which requires exactly 2 content infos and
// fails on P12 files that store the key and certificates in separate authenticated-safe items.
//
// Parameters:
//   - p12Path: Absolute path to the PKCS#12 file (may have a "file:///" prefix).
//   - password: Password protecting the PKCS#12 file.
//
// Returns:
//   - *tls.Config: A TLS configuration with the client certificate loaded.
//   - error: Any error encountered while reading or decoding the file.
func buildTLSConfigFromP12(p12Path, password string) (*tls.Config, error) {
	filePath := filepath.Clean(strings.TrimPrefix(p12Path, "file:///"))
	if strings.ContainsRune(filePath, 0) {
		return nil, fmt.Errorf("invalid p12 path: contains null byte")
	}
	// #nosec G304 -- filePath is cleaned via filepath.Clean and validated to contain no null bytes
	p12Data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read p12 file: %w", err)
	}

	// DecodeChain handles P12 files with certificate chains (including CA certs).
	// It accepts 1-2 items in the authenticated safe, which covers the common case
	// where the key and all certificates are stored together or in two separate items.
	privateKey, endEntityCert, caCerts, err := pkcs12.DecodeChain(p12Data, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decode p12 (check password/file): %w", err)
	}

	// Encode the private key to PKCS#8 PEM
	privKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key from p12: %w", err)
	}
	keyPEMBlock := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privKeyDER})

	// Build the certificate chain PEM: end-entity cert first, then CA certs
	var certPEMBuf bytes.Buffer
	if errPemChain := pem.Encode(&certPEMBuf, &pem.Block{Type: "CERTIFICATE", Bytes: endEntityCert.Raw}); errPemChain != nil {
		return nil, fmt.Errorf("failed to encode end-entity certificate from p12: %w", errPemChain)
	}
	for _, caCert := range caCerts {
		if errCaCert := pem.Encode(&certPEMBuf, &pem.Block{Type: "CERTIFICATE", Bytes: caCert.Raw}); errCaCert != nil {
			return nil, fmt.Errorf("failed to encode CA certificate from p12: %w", errCaCert)
		}
	}

	clientCert, err := tls.X509KeyPair(certPEMBuf.Bytes(), keyPEMBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS certificate from p12: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// NewP12CertAuth is a factory function to initialize the P12CertAuth strategy for mTLS authentication using a PKCS#12 file.
// Parameters:
//   - p12Path: The file path to the PKCS#12 certificate file. May be a plain absolute path
//     or a file URI (e.g. "file:////absolute/path" as produced by FormatToAbsolutePath).
//   - password: The password for the PKCS#12 file.
//   - timeout: The timeout duration for HTTP requests.
//
// Returns:
//   - *P12CertAuth: A pointer to the initialized P12CertAuth struct.
//   - error: An error if the file cannot be read or decoded.
func NewP12CertAuth(p12Path, password string, timeout time.Duration) (*P12CertAuth, error) {
	tlsConfig, err := buildTLSConfigFromP12(p12Path, password)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Transport: wrapTransportForVCR(&http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: tlsConfig,
		}),
		Timeout: timeout,
	}

	return &P12CertAuth{
		P12Path:  p12Path,
		Password: password,
		Client:   client,
	}, nil
}

// ApplyAuth implements the Authenticator interface for APITokenAuth.
// Sets the Authorization header for the request using the API token.
// Receiver: *APITokenAuth
// Parameters:
//   - req: The HTTP request to which authentication headers will be applied.
//
// Returns:
//   - error: Always nil for token authentication.
func (a *APITokenAuth) ApplyAuth(req *http.Request) error {
	cleanToken := strings.TrimSpace(a.Token)
	authHeaderValue := fmt.Sprintf("APIToken %s", cleanToken)
	req.Header.Set("Authorization", authHeaderValue)

	return nil
}

// GetClient implements the Authenticator interface for APITokenAuth.
// Returns a standard HTTP client configured for token authentication.
// Receiver: *APITokenAuth
// Returns:
//   - *http.Client: The HTTP client instance
func (a *APITokenAuth) GetClient() *http.Client {
	if a.Client != nil {
		return a.Client
	}

	a.Client = &http.Client{
		Timeout: 10 * time.Second,
		Transport: wrapTransportForVCR(&http.Transport{
			Proxy: http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		}),
	}

	return a.Client
}

// GetClientData implements the Authenticator interface for APITokenAuth.
// Returns the API token string used for authentication.
// Receiver: *APITokenAuth
// Returns:
//   - string: The API token.
func (a *APITokenAuth) GetClientData() string {
	return a.Token
}

// ApplyAuth implements the Authenticator interface for P12CertAuth.
// For mTLS, authentication is handled by the TLS handshake, so this method does not modify headers.
// Receiver: *P12CertAuth
// Parameters:
//   - req: The HTTP request to which authentication headers would be applied.
//
// Returns:
//   - error: Always nil for mTLS authentication.
func (a *P12CertAuth) ApplyAuth(_ *http.Request) error {
	// P12 (mTLS) auth is handled entirely by the Client's Transport layer,
	// so this method can remain empty or set other non-auth headers if needed.
	return nil
}

// GetClient implements the Authenticator interface for P12CertAuth.
// Returns the pre-configured HTTP client for mTLS authentication.
// Receiver: *P12CertAuth
// Returns:
//   - *http.Client: The HTTP client instance.
func (a *P12CertAuth) GetClient() *http.Client {
	return a.Client
}

// GetClientData implements the Authenticator interface for P12CertAuth.
// Returns an empty string, as mTLS does not use a token.
// Receiver: *P12CertAuth
// Returns:
//   - string: Always empty.
func (a *P12CertAuth) GetClientData() string {
	return ""
}

// Execute performs an HTTP request using the client provided by the current authentication strategy.
// Applies authentication, sets standard headers, and logs request/response details.
// Receiver: *AuthClient
// Parameters:
//   - ctx: The context for request lifecycle and logging.
//   - method: The HTTP method (e.g., "GET", "POST").
//   - url: The target URL for the request.
//   - body: The request body as a byte slice.
//
// Returns:
//   - []byte: The response body as a byte slice.
//   - error: Any error encountered during request creation, authentication, execution, or response reading.
func (ac *AuthClient) Execute(ctx context.Context, method, url string, body []byte) ([]byte, error) {
	req, httpErr := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(body))
	if httpErr != nil {
		return nil, fmt.Errorf("error creating request: %w", httpErr)
	}

	// The Context uses the Strategy to apply the necessary headers/config
	if err := ac.Authenticator.ApplyAuth(req); err != nil {
		return nil, fmt.Errorf("error applying authentication: %w", err)
	}

	userAgent := ac.UserAgent
	if userAgent == "" {
		userAgent = providerUserAgent("")
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json")

	client := ac.Authenticator.GetClient()
	tflog.Info(ctx, fmt.Sprintf("Making request to %s with strategy: %T...\n", ac.URL, ac.Authenticator))

	tflog.Info(ctx, fmt.Sprintf("Method: %s URL: %s", method, url))
	tflog.Info(ctx, fmt.Sprintf("Req URL: %s", req.URL.String()))
	sanitizedHeader := req.Header.Clone()
	if sanitizedHeader.Get("Authorization") != "" {
		sanitizedHeader.Set("Authorization", "APIToken [REDACTED]")
	}
	if sanitizedHeader.Get("X-Volterra-Token") != "" {
		sanitizedHeader.Set("X-Volterra-Token", "[REDACTED]")
	}
	if sanitizedHeader.Get("Cookie") != "" {
		sanitizedHeader.Set("Cookie", "[REDACTED]")
	}
	tflog.Debug(ctx, fmt.Sprintf("Req Header: %v", sanitizedHeader))

	resourceKey := resolveRequestResourceKey(req)
	resp, err := doRequestWithThrottling(ctx, req, resourceKey, ac.ThrottlePolicy, ac.ThrottleRegistry, func(attemptReq *http.Request) (*http.Response, error) {
		// #nosec G704 -- HTTP client executes requests to provider-configured F5 XC API endpoints
		return client.Do(attemptReq)
	})
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			tflog.Error(ctx, fmt.Sprintf("error closing body: %v", err))
		}
	}()

	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		tflog.Error(ctx, fmt.Sprintf("Error reading response body: %v", readErr))
		return nil, fmt.Errorf("error reading response body: %v", readErr)
	}

	tflog.Debug(ctx, fmt.Sprintf("Response Status: %s (Code: %d)", resp.Status, resp.StatusCode))

	if resp.StatusCode != http.StatusOK {
		tflog.Error(ctx, fmt.Sprintf("API call returned status %s. Body: %s", resp.Status, string(respBody)))
		if len(respBody) > 0 {
			return nil, fmt.Errorf("API call returned status %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
		}
		return nil, fmt.Errorf("API call returned status %s", resp.Status)
	}

	return respBody, nil
}

// CreateObject populates the correct slug and calls F5XC CreateObject
func (a *APIClient) CreateObject(ctx context.Context, objType string, req vesapi.CreateObjectRequest, opts ...vesapi.CallOpt) (vesapi.CreateObjectResponse, error) {
	ctx = contextFromResourceKey(ctx, objType)
	tflog.Info(ctx, fmt.Sprintf("Create request %s sent at %s\n", objType, time.Now().Format("15:04:05.000")))

	var err error
	ctx, err = addSvcSlugToContextForCRUD(ctx, objType)
	if err != nil {
		return nil, err
	}

	return a.APIClient.CreateObject(ctx, objType, req, opts...)
}

// GetObject populates the correct slug and calls F5XC GetObject
func (a *APIClient) GetObject(ctx context.Context, objType, namespace, name string, opts ...vesapi.CallOpt) (vesapi.GetObjectResponse, error) {
	ctx = contextFromResourceKey(ctx, objType)
	tflog.Info(ctx, fmt.Sprintf("Get request %s sent at %s\n", objType, time.Now().Format("15:04:05.000")))

	var err error
	ctx, err = addSvcSlugToContextForCRUD(ctx, objType)
	if err != nil {
		return nil, err
	}
	return a.APIClient.GetObject(ctx, objType, namespace, name, opts...)
}

// ReplaceObject populates the correct slug and calls F5XC ReplaceObject
func (a *APIClient) ReplaceObject(ctx context.Context, objType string, req vesapi.ReplaceObjectRequest, opts ...vesapi.CallOpt) error {
	ctx = contextFromResourceKey(ctx, objType)
	tflog.Info(ctx, fmt.Sprintf("Replace request %s sent at %s\n", objType, time.Now().Format("15:04:05.000")))

	var err error
	ctx, err = addSvcSlugToContextForCRUD(ctx, objType)
	if err != nil {
		return err
	}
	return a.APIClient.ReplaceObject(ctx, objType, req, opts...)

}

// DeleteObject populates the correct slug and calls F5XC DeleteObject
func (a *APIClient) DeleteObject(ctx context.Context, objType string, namespace, name string, opts ...vesapi.CallOpt) error {
	ctx = contextFromResourceKey(ctx, objType)
	tflog.Info(ctx, fmt.Sprintf("Delete request %s sent at %s\n", objType, time.Now().Format("15:04:05.000")))

	var err error
	ctx, err = addSvcSlugToContextForCRUD(ctx, objType)
	if err != nil {
		return err
	}
	return a.APIClient.DeleteObject(ctx, objType, namespace, name, opts...)
}

// ListObjects populates the correct slug and calls F5XC ListObjects
func (a *APIClient) ListObjects(ctx context.Context, objType, namespace string, opts ...vesapi.CallOpt) ([]vesapi.ListObjectItem, error) {
	ctx = contextFromResourceKey(ctx, objType)
	tflog.Info(ctx, fmt.Sprintf("List request %s sent at %s\n", objType, time.Now().Format("15:04:05.000")))

	var err error
	ctx, err = addSvcSlugToContextForCRUD(ctx, objType)
	if err != nil {
		return nil, err
	}
	return a.APIClient.ListObjects(ctx, objType, namespace, opts...)
}

// CustomAPI populates the correct slug and calls F5XC custom API
func (a *APIClient) CustomAPI(ctx context.Context, method, uri, rpcFQN, reqYml string, opts ...vesapi.CallOpt) (proto.Message, error) {
	ctx = contextFromResourceKey(ctx, resourceKeyForCustomRPC(rpcFQN))
	tflog.Info(ctx, fmt.Sprintf("Custom API request %s sent at %s\n", method, time.Now().Format("15:04:05.000")))

	var err error
	ctx, err = addSvcSlugToContextForCustom(ctx, rpcFQN)
	if err != nil {
		return nil, err
	}
	return a.APIClient.CustomAPI(ctx, method, uri, rpcFQN, reqYml, opts...)
}

// serviceSlugFromContext gets the api gateway service discriminator from the context
func serviceSlugFromContext(ctx context.Context) string {
	slug, ok := ctx.Value(ctxServiceSlug{}).(string)
	if !ok {
		return ""
	}
	return slug
}

// logAndRoundTrip executes an HTTP request, logs the curl command, response, and errors.
// Parameters:
//   - rt: The http.RoundTripper to use for sending the request.
//   - req: The *http.Request to be sent.
//
// Returns:
//   - rsp: The *http.Response received from the server.
//   - err: Any error encountered during the request.
func logAndRoundTrip(rt http.RoundTripper, req *http.Request) (*http.Response, error) {
	if reqForCurl := req.Clone(req.Context()); reqForCurl != nil {
		if req.Body != nil {
			var bodyBytes []byte
			if req.GetBody != nil {
				if rc, err := req.GetBody(); err == nil {
					bodyBytes, _ = io.ReadAll(rc)
					_ = rc.Close()
				}
			} else {
				var err error
				bodyBytes, err = io.ReadAll(req.Body)
				if err != nil {
					bodyBytes = nil
				}
			}
			// Independent reader for the real network request
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			req.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(bodyBytes)), nil
			}
			// Independent reader for the curl logger
			reqForCurl.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			reqForCurl.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(bodyBytes)), nil
			}
		}

		if reqForCurl.Header.Get("Authorization") != "" {
			reqForCurl.Header.Set("Authorization", "APIToken [REDACTED]")
		}
		if reqForCurl.Header.Get("X-Volterra-Token") != "" {
			reqForCurl.Header.Set("X-Volterra-Token", "[REDACTED]")
		}
		if reqForCurl.Header.Get("Cookie") != "" {
			reqForCurl.Header.Set("Cookie", "[REDACTED]")
		}
		if curlOut, err := http2curl.GetCurlCommand(reqForCurl); err == nil {
			// #nosec G706 -- trace request curl output for debugging
			log.Printf("[TRACE] Request Sent: \n%s\n", curlOut)
		}
	}

	rsp, err := wrapTransportForVCR(rt).RoundTrip(req)
	if err != nil {
		// #nosec G706 -- trace error log for debugging
		log.Printf("[TRACE] Response Error: \n%s\n", err.Error())
	} else {
		rspForDump := *rsp
		rspForDump.Header = rsp.Header.Clone()
		if rspForDump.Header.Get("Set-Cookie") != "" {
			rspForDump.Header.Set("Set-Cookie", "[REDACTED]")
		}
		if out, httpErr := httputil.DumpResponse(&rspForDump, true); httpErr == nil {
			rsp.Body = rspForDump.Body
			// #nosec G706 -- trace response log for debugging
			log.Printf("[TRACE] Response Output:\n%s\n", out)
		}
	}
	return rsp, err
}

// prismURIRemapperFn remaps the request URI for Prism API endpoints and logs the request and response.
// Parameters:
//   - rt: The http.RoundTripper used to execute the HTTP request.
//   - req: The *http.Request to be remapped and sent.
//
// Returns:
//   - *http.Response: The HTTP response received from the server.
//   - error: Any error encountered during the request or remapping process.
func prismURIRemapperFn(apiGwURL, userAgent string, policy throttlingPolicy, registry *ResourceThrottleRegistry) (stdlibClient.RoundTripperFunc, error) {
	u, err := url.Parse(apiGwURL)
	if err != nil {
		return nil, errors.Wrapf(err, "Parsing apigw url %s", apiGwURL)
	}
	rtFn := func(req *http.Request, rt http.RoundTripper) (*http.Response, error) {
		resourceKey := resolveRequestResourceKey(req)
		return doRequestWithThrottling(req.Context(), req, resourceKey, policy, registry, func(attemptReq *http.Request) (*http.Response, error) {
			attemptReq.Header.Set("User-Agent", userAgent)
			// convert https://x.y.volterra.us/api + /public/some/thing/here to
			//         https://x.y.volterra.us/api/config/some/thing/here
			srvURI := u.Path
			ctx := attemptReq.Context()
			slug := serviceSlugFromContext(ctx)
			if slug == "" {
				return nil, fmt.Errorf("no service slug defined for URI %s", srvURI)
			}
			toStrip := "/public"
			if len(srvURI) > 0 && srvURI[len(srvURI)-1] != '/' {
				// "/public" should be replaced by "/<slug>"
				slug = "/" + slug
			}
			uriPfx := fmt.Sprintf("%s%s", srvURI, toStrip)
			if strings.HasPrefix(attemptReq.URL.Path, uriPfx) {
				newURL := strings.Replace(attemptReq.URL.String(), toStrip, slug, 1)
				u, parseErr := url.Parse(newURL)
				if parseErr != nil {
					return nil, errors.Wrapf(parseErr, "Parsing remapped url %s", newURL)
				}
				attemptReq.URL = u
			}

			return logAndRoundTrip(rt, attemptReq)
		})
	}
	return rtFn, nil
}

// uriRemapperFn remaps the request URI for standard API endpoints and logs the request and response.
// Parameters:
//   - rt: The http.RoundTripper used to execute the HTTP request.
//   - req: The *http.Request to be remapped and sent.
//
// Returns:
//   - *http.Response: The HTTP response received from the server.
//   - error: Any error encountered during the request or remapping process.
func uriRemapperFn(tenant, userAgent string, policy throttlingPolicy, registry *ResourceThrottleRegistry) stdlibClient.RoundTripperFunc {
	rtFn := func(req *http.Request, rt http.RoundTripper) (*http.Response, error) {
		resourceKey := resolveRequestResourceKey(req)
		return doRequestWithThrottling(req.Context(), req, resourceKey, policy, registry, func(attemptReq *http.Request) (*http.Response, error) {
			attemptReq.Header.Set("User-Agent", userAgent)
			attemptReq.Header.Set(server.TenantHdrName, tenant)
			return logAndRoundTrip(rt, attemptReq)
		})
	}
	return rtFn
}

// getClientOpts constructs and returns a slice of client options for configuring the VesAPI client.
// Parameters:
//   - ctx: The context.Context for client configuration and request lifecycle.
//   - opts: A variadic list of VesAPIClientOpt for additional client customization.
//
// Returns:
//   - []VesAPIClientOpt: A slice of client options to be used when creating a VesAPI client.
func getClientOpts(c *ProviderConfig, policy throttlingPolicy, registry *ResourceThrottleRegistry) ([]vesapi.ConfigOpt, error) {
	userAgent := providerUserAgentWithVersion(c.TerraformVersion, c.ProviderVersion)
	remapperFn, err := prismURIRemapperFn(c.Url, userAgent, policy, registry)
	if err != nil {
		return nil, err
	}
	if c.Vesenv {
		remapperFn = uriRemapperFn(c.tenant, userAgent, policy, registry)
	}

	dur, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return nil, err
	}
	clOpts := []vesapi.ConfigOpt{
		vesapi.WithCfgRoundTripperFn(remapperFn),
		vesapi.WithCfgRPCTimeout(dur),
	}

	if c.ApiCert != "" && c.ApiKey != "" {
		clOpts = append(clOpts, vesapi.WithCfgCert(c.ApiCert))
		clOpts = append(clOpts, vesapi.WithCfgKey(c.ApiKey))
	}

	if c.ApiP12File != "" && c.ApiP12Password != "" {
		// Build the TLS config directly using pkcs12.DecodeChain instead of
		// tlsutil.BuildClientTLSConfig, which internally uses pkcs12.DecodeChain
		// from software.sslmate.com/src/go-pkcs12 with getSafeContents(..., 1, 2).
		// That fails on P12 files that have more than 2 items in the authenticated
		// safe (e.g. key + end-entity cert + CA cert stored as separate content infos).
		tlsCfg, err := buildTLSConfigFromP12(c.ApiP12File, c.ApiP12Password)
		if err != nil {
			return nil, errors.Wrapf(err, "Building TLS configuration, bundle %s", c.ApiP12File)
		}
		clOpts = append(clOpts, vesapi.WithTLSCfg(tlsCfg))
	}

	if c.ApiCACert != "" {
		clOpts = append(clOpts, vesapi.WithCfgCACert(c.ApiCACert))
	}

	if c.ApiToken != "" {
		clOpts = append(clOpts, vesapi.WithCfgApiToken(c.ApiToken))
	}

	return clOpts, nil
}

// NewVesAPIClient creates and initializes a new VesAPI client with the provided options.
// Parameters:
//   - ctx: The context.Context for client configuration and request lifecycle.
//   - opts: A variadic list of VesAPIClientOpt for client customization.
//
// Returns:
//   - *VesAPIClient: The initialized VesAPI client instance.
//   - error: Any error encountered during client initialization.
func NewVesAPIClient(c *ProviderConfig) (*APIClient, error) {
	throttlingPolicy, err := resolveThrottlingPolicy(c.Throttling)
	if err != nil {
		return nil, errors.Wrap(err, "Building throttling policy")
	}
	throttleRegistry := newResourceThrottleRegistry()

	var clOpts []vesapi.ConfigOpt
	clOpts, err = getClientOpts(c, throttlingPolicy, throttleRegistry)

	if err != nil {
		return nil, errors.Wrap(err, "Building client options")
	}

	userAgent := providerUserAgentWithVersion(c.TerraformVersion, c.ProviderVersion)

	if c.Test {
		clOpts, err = getTestClientOpts(c.Url, c.tenant, userAgent)
		if err != nil {
			return nil, errors.Wrap(err, "Building client options")
		}
	}

	apiClient, err := vesapi.NewAPIClient(c.Url, ves_io_schema_combined.MDR, clOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "Creating new F5XC APIClient")
	}

	var authClient *AuthClient

	if c.ApiToken != "" {
		tokenAuth := NewAPITokenAuth(c.ApiToken, 5*time.Second)
		authClient = NewAuthClient(c.Url, tokenAuth, userAgent)
	}

	if c.ApiP12File != "" && c.ApiP12Password != "" {
		p12Auth, err := NewP12CertAuth(c.ApiP12File, c.ApiP12Password, 30*time.Second)
		if err != nil {
			return nil, errors.Wrap(err, "Creating new F5XC APIClient")
		}
		authClient = NewAuthClient(c.Url, p12Auth, userAgent)
	}
	if authClient != nil {
		authClient.ThrottlePolicy = throttlingPolicy
		authClient.ThrottleRegistry = throttleRegistry
	}

	c1 := APIClient{
		APIClient:  apiClient,
		RestClient: authClient,
		Config: &ClientTimeouts{
			TimeoutCreate: c.TimeoutCreate,
			TimeoutRead:   c.TimeoutRead,
			TimeoutUpdate: c.TimeoutUpdate,
			TimeoutDelete: c.TimeoutDelete,
		},
	}

	return &c1, nil
}
