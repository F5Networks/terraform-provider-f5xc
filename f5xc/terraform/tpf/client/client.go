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
	Url            string            // The base URL for the F5XC API endpoint.
	Test           bool              // Indicates whether the provider is running in test mode.
	Vesenv         bool              // Specifies if the provider is running in a Ves environment.
	tenant         string            // The tenant identifier for API requests.
	Timeout        string            // The timeout duration for API requests, as a string (e.g., "30s").
	TimeoutCreate  string            // Default timeout for Create operations (e.g. "10m"). Falls back to resource default if empty.
	TimeoutRead    string            // Default timeout for Read operations (e.g. "5m"). Falls back to resource default if empty.
	TimeoutUpdate  string            // Default timeout for Update operations (e.g. "10m"). Falls back to resource default if empty.
	TimeoutDelete  string            // Default timeout for Delete operations (e.g. "10m"). Falls back to resource default if empty.
	Throttling     *ThrottlingConfig // Backend-driven request throttling configuration.
	ApiCert        string            // The path to the API client certificate file.
	ApiKey         string            // The path to the API client key file.
	ApiToken       string            // The API token used for authentication.
	ApiCACert      string            // The path to the CA certificate file for API requests.
	ApiP12File     string            // The path to the PKCS#12 file containing client credentials.
	ApiP12Password string            // The password for the PKCS#12 file.
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

// Supported CRUD operations
const (
	Create = iota
	Delete
	Replace
	Get
)

// NewAuthClient creates a new client with a specified authentication strategy.
func NewAuthClient(url string, auth Authenticator) *AuthClient {
	return &AuthClient{
		URL:           url,
		Authenticator: auth,
	}
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
	filePath := strings.TrimPrefix(p12Path, "file:///")
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
		return nil, fmt.Errorf("failed to encode end-entity certificate from p12: %w", err)
	}
	for _, caCert := range caCerts {
		if errCaCert := pem.Encode(&certPEMBuf, &pem.Block{Type: "CERTIFICATE", Bytes: caCert.Raw}); errCaCert != nil {
			return nil, fmt.Errorf("failed to encode CA certificate from p12: %w", err)
		}
	}

	clientCert, err := tls.X509KeyPair(certPEMBuf.Bytes(), keyPEMBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS certificate from p12: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{clientCert},
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

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "F5XC-Terraform-Provider/1.0.0 (Go-Client)")
	req.Header.Set("Content-Type", "application/json")

	client := ac.Authenticator.GetClient()
	tflog.Info(ctx, fmt.Sprintf("Making request to %s with strategy: %T...\n", ac.URL, ac.Authenticator))

	tflog.Info(ctx, fmt.Sprintf("Method: %s URL: %s", method, url))
	tflog.Info(ctx, fmt.Sprintf("Req URL: %s", req.URL.String()))
	tflog.Info(ctx, fmt.Sprintf("Req Header: %s", req.Header))

	resourceKey := resolveRequestResourceKey(req)
	resp, err := doRequestWithThrottling(ctx, req, resourceKey, ac.ThrottlePolicy, ac.ThrottleRegistry, func(attemptReq *http.Request) (*http.Response, error) {
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

	// Log the full status string and code
	tflog.Info(ctx, fmt.Sprintf("DEBUG: Response Status: %s", resp.Status))
	tflog.Info(ctx, fmt.Sprintf("DEBUG: Response Code: %d", resp.StatusCode))

	// Use string(respBody) to convert the []byte to a string
	tflog.Info(ctx, fmt.Sprintf("DEBUG: Full Response Body: %s", string(respBody)))

	if resp.StatusCode != http.StatusOK {
		tflog.Error(ctx, fmt.Sprintf("API call returned status %s. Body: %s", resp.Status, string(respBody)))
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
	curlOut, err := http2curl.GetCurlCommand(req)
	if err == nil {
		log.Printf("[TRACE] Request Sent: \n%s\n", curlOut)
	}

	rsp, err := wrapTransportForVCR(rt).RoundTrip(req)
	if err != nil {
		log.Printf("[TRACE] Response Error: \n%s\n", err.Error())
	} else {
		if out, httpErr := httputil.DumpResponse(rsp, true); httpErr == nil {
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
func prismURIRemapperFn(apiGwURL string, policy throttlingPolicy, registry *ResourceThrottleRegistry) (stdlibClient.RoundTripperFunc, error) {
	u, err := url.Parse(apiGwURL)
	if err != nil {
		return nil, errors.Wrapf(err, "Parsing apigw url %s", apiGwURL)
	}
	rtFn := func(req *http.Request, rt http.RoundTripper) (*http.Response, error) {
		resourceKey := resolveRequestResourceKey(req)
		return doRequestWithThrottling(req.Context(), req, resourceKey, policy, registry, func(attemptReq *http.Request) (*http.Response, error) {
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
func uriRemapperFn(tenant string, policy throttlingPolicy, registry *ResourceThrottleRegistry) stdlibClient.RoundTripperFunc {
	rtFn := func(req *http.Request, rt http.RoundTripper) (*http.Response, error) {
		resourceKey := resolveRequestResourceKey(req)
		return doRequestWithThrottling(req.Context(), req, resourceKey, policy, registry, func(attemptReq *http.Request) (*http.Response, error) {
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
	remapperFn, err := prismURIRemapperFn(c.Url, policy, registry)
	if err != nil {
		return nil, err
	}
	if c.Vesenv {
		remapperFn = uriRemapperFn(c.tenant, policy, registry)
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

	if c.Test {
		clOpts, err = getTestClientOpts(c.Url, c.tenant)
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
		authClient = NewAuthClient(c.Url, tokenAuth)
	}

	if c.ApiP12File != "" && c.ApiP12Password != "" {
		p12Auth, err := NewP12CertAuth(c.ApiP12File, c.ApiP12Password, 30*time.Second)
		if err != nil {
			return nil, errors.Wrap(err, "Creating new F5XC APIClient")
		}
		authClient = NewAuthClient(c.Url, p12Auth)
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
