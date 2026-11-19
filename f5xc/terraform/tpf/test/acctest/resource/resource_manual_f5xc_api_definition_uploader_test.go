//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

//go:build acceptance

package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/client"
	acctest "gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/test/acctest"
)

func TestAccResourceManualF5XCApiDefinitionUploader_basic(t *testing.T) {
	t.Parallel()

	namespace := "shared"
	name := "api-uploader-" + acctest.RandString(t, 8)
	specV1 := `{"openapi":"3.0.0","info":{"title":"Test API","version":"1.0.0"},"paths":{"/test":{"get":{"summary":"Test endpoint","responses":{"200":{"description":"Success"}}}}}}`
	specV2 := `{"openapi":"3.0.0","info":{"title":"Test API","version":"2.0.0"},"paths":{"/test":{"get":{"summary":"Test endpoint","responses":{"200":{"description":"Success"}}}}}}`

	resourceName := "f5xc_api_definition_uploader.test"
	initialDesc := "Explicit description preserved via Terraform"
	updatedDesc := "Updated description via Terraform"

	acctest.VcrTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.AccTestPreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		Steps: []resource.TestStep{
			// Step 1: Create with explicit description
			{
				Config: testAccResourceManualF5XCApiDefinitionUploaderConfig(name, namespace, initialDesc, specV1, 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.namespace", namespace),
					resource.TestCheckResourceAttr(resourceName, "metadata.description", initialDesc),
					resource.TestMatchResourceAttr(resourceName, "metadata.version", regexp.MustCompile(`^v\d+`)),
					resource.TestMatchResourceAttr(resourceName, "metadata.url", regexp.MustCompile(fmt.Sprintf(`/stored_objects/swagger/%s/`, name))),
					resource.TestCheckResourceAttrSet(resourceName, "content_hash"),
					resource.TestCheckResourceAttr(resourceName, "retain_versions", "2"),
					resource.TestCheckNoResourceAttr(resourceName, "previous_version_url"),
				),
			},
			// Step 2: Update with modified OpenAPI spec content and updated description
			{
				Config: testAccResourceManualF5XCApiDefinitionUploaderConfig(name, namespace, updatedDesc, specV2, 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.namespace", namespace),
					resource.TestCheckResourceAttr(resourceName, "metadata.description", updatedDesc),
					resource.TestMatchResourceAttr(resourceName, "metadata.version", regexp.MustCompile(`^v\d+`)),
					resource.TestCheckResourceAttrSet(resourceName, "content_hash"),
					resource.TestCheckResourceAttrSet(resourceName, "previous_version_url"),
				),
			},
			// Step 3: Re-apply same config - must be idempotent
			{
				Config: testAccResourceManualF5XCApiDefinitionUploaderConfig(name, namespace, updatedDesc, specV2, 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.namespace", namespace),
					resource.TestCheckResourceAttr(resourceName, "metadata.description", updatedDesc),
				),
			},
			// Step 4: Clear description back to empty string without content change
			{
				Config: testAccResourceManualF5XCApiDefinitionUploaderConfig(name, namespace, "", specV2, 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.namespace", namespace),
					resource.TestCheckResourceAttr(resourceName, "metadata.description", ""),
				),
			},
		},
	})
}

func testAccResourceManualF5XCApiDefinitionUploaderConfig(name, namespace, description, specContent string, retainVersions int) string {
	return fmt.Sprintf(`
resource "f5xc_api_definition_uploader" "test" {
  metadata = {
    name        = %q
    namespace   = %q
    description = %q
  }
  string_value    = %q
  retain_versions = %d
}
`, name, namespace, description, specContent, retainVersions)
}

// TestAccResourceManualF5XCApiDefinitionUploader_RetentionPruning verifies end-to-end
// historical version pruning across repeated OpenAPI spec updates with retain_versions = 2
// and verifies disabling pruning with retain_versions = 0 (XC-35500 / XC-94246).
func TestAccResourceManualF5XCApiDefinitionUploader_RetentionPruning(t *testing.T) {
	t.Parallel()

	namespace := "shared"
	name := "api-uploader-ret-" + testAccRunSuffix(t)
	specTemplate := `{"openapi":"3.0.0","info":{"title":"XC35500 TC07","version":%q},"paths":{"/ping":{"get":{"summary":"Ping","operationId":"ping","responses":{"200":{"description":"ok"}}}}}}`

	resourceName := "f5xc_api_definition_uploader.test"

	acctest.VcrTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.AccTestPreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		CheckDestroy:             testAccCheckApiDefinitionUploaderDestroy(namespace, name),
		Steps: []resource.TestStep{
			// Step 1: Create v1.0.0 -> 1 backend version
			{
				Config: testAccResourceManualF5XCApiDefinitionUploaderConfigWithForceDelete(
					name, namespace, fmt.Sprintf(specTemplate, "1.0.0"), 2, true,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.namespace", namespace),
					resource.TestMatchResourceAttr(resourceName, "metadata.version", regexp.MustCompile(`^v\d+`)),
					resource.TestCheckResourceAttr(resourceName, "retain_versions", "2"),
					resource.TestCheckResourceAttr(resourceName, "force_delete", "true"),
					testAccCheckApiDefinitionUploaderRetainedVersions(namespace, name, 1),
				),
			},
			// Step 2: Update v2.0.0 -> 2 backend versions
			{
				Config: testAccResourceManualF5XCApiDefinitionUploaderConfigWithForceDelete(
					name, namespace, fmt.Sprintf(specTemplate, "2.0.0"), 2, true,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(resourceName, "metadata.version", regexp.MustCompile(`^v\d+`)),
					resource.TestCheckResourceAttrSet(resourceName, "previous_version_url"),
					testAccCheckApiDefinitionUploaderRetainedVersions(namespace, name, 2),
				),
			},
			// Step 3: Update v3.0.0 -> pruned to 2 backend versions
			{
				Config: testAccResourceManualF5XCApiDefinitionUploaderConfigWithForceDelete(
					name, namespace, fmt.Sprintf(specTemplate, "3.0.0"), 2, true,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(resourceName, "metadata.version", regexp.MustCompile(`^v\d+`)),
					resource.TestCheckResourceAttrSet(resourceName, "previous_version_url"),
					testAccCheckApiDefinitionUploaderRetainedVersions(namespace, name, 2),
				),
			},
			// Step 4: Update v4.0.0 -> still pruned to 2 backend versions
			{
				Config: testAccResourceManualF5XCApiDefinitionUploaderConfigWithForceDelete(
					name, namespace, fmt.Sprintf(specTemplate, "4.0.0"), 2, true,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(resourceName, "metadata.version", regexp.MustCompile(`^v\d+`)),
					resource.TestCheckResourceAttr(resourceName, "retain_versions", "2"),
					testAccCheckApiDefinitionUploaderRetainedVersions(namespace, name, 2),
				),
			},
			// Step 5: Update v5.0.0 with retain_versions = 0 -> pruning disabled, 3 backend versions exist
			{
				Config: testAccResourceManualF5XCApiDefinitionUploaderConfigWithForceDelete(
					name, namespace, fmt.Sprintf(specTemplate, "5.0.0"), 0, true,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(resourceName, "metadata.version", regexp.MustCompile(`^v\d+`)),
					resource.TestCheckResourceAttr(resourceName, "retain_versions", "0"),
					testAccCheckApiDefinitionUploaderRetainedVersions(namespace, name, 3),
				),
			},
		},
	})
}

// testAccStoredObjectListResponse represents the F5 XC Object Store list response schema.
type testAccStoredObjectListResponse struct {
	Items []struct {
		Name     string `json:"name"`
		Versions []struct {
			Version string `json:"version"`
		} `json:"versions"`
	} `json:"items"`
}

// newAccTestUploaderAPIClient constructs an authenticated F5 XC API client from acceptance test env vars.
//
// Why: Acceptance assertions verify actual backend stored_objects version counts out-of-band from Terraform state.
//
// How: Populates client.ProviderConfig from F5XCProvider_* environment variables matching acctest.DeleteIfExists
// and constructs the REST client via client.NewVesAPIClient.
//
// Parameters:
//   - None (reads F5XCProvider_URL, F5XCProvider_API_TOKEN, F5XCProvider_API_P12_FILE, VES_P12_PASSWORD,
//     F5XCProvider_API_CERT, F5XCProvider_API_KEY)
//
// Returns:
//   - *client.APIClient: initialized REST API client
//   - error: non-nil if client construction fails
//
// Example:
//
//	apiClient, err := newAccTestUploaderAPIClient()
func newAccTestUploaderAPIClient() (*client.APIClient, error) {
	cfg := &client.ProviderConfig{
		Url:     os.Getenv("F5XCProvider_URL"),
		Timeout: "30s",
	}
	if token := os.Getenv("F5XCProvider_API_TOKEN"); token != "" {
		cfg.ApiToken = token
	}
	if p12 := os.Getenv("F5XCProvider_API_P12_FILE"); p12 != "" {
		cfg.ApiP12File = p12
		cfg.ApiP12Password = os.Getenv("VES_P12_PASSWORD")
	}
	if cert := os.Getenv("F5XCProvider_API_CERT"); cert != "" {
		cfg.ApiCert = cert
		cfg.ApiKey = os.Getenv("F5XCProvider_API_KEY")
	}

	return client.NewVesAPIClient(cfg)
}

// fetchUploaderBackendVersions retrieves all remaining stored object version identifiers for namespace/name.
//
// Why: Enables acceptance TestCheckFunc and CheckDestroy hooks to verify exact backend retention and cleanup.
//
// How: Executes a GET request against object_store/namespaces/<ns>/stored_objects/swagger and returns
// the versions slice for the item matching either "<name>" or "<namespace>/<name>".
//
// Parameters:
//   - ctx: context for the HTTP request
//   - apiClient: initialized F5 XC API client
//   - namespace: target F5 XC namespace
//   - name: stored object name (without namespace prefix)
//
// Returns:
//   - []string: version identifiers currently present on the backend (nil/empty if object is absent)
//   - error: non-nil if the list request or JSON decoding fails
//
// Example:
//
//	versions, err := fetchUploaderBackendVersions(ctx, apiClient, "shared", "my-api")
func fetchUploaderBackendVersions(ctx context.Context, apiClient *client.APIClient, namespace, name string) ([]string, error) {
	listURL := fmt.Sprintf("%s/object_store/namespaces/%s/stored_objects/swagger",
		apiClient.RestClient.URL, namespace)

	respBytes, err := apiClient.RestClient.Execute(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "Not Found") {
			return nil, nil
		}
		return nil, err
	}

	var listResp testAccStoredObjectListResponse
	if unmarshalErr := json.Unmarshal(respBytes, &listResp); unmarshalErr != nil {
		return nil, unmarshalErr
	}

	prefixedName := namespace + "/" + name
	for _, item := range listResp.Items {
		if item.Name == name || item.Name == prefixedName {
			versions := make([]string, 0, len(item.Versions))
			for _, v := range item.Versions {
				versions = append(versions, v.Version)
			}
			return versions, nil
		}
	}

	return nil, nil
}

// testAccCheckApiDefinitionUploaderRetainedVersions returns a TestCheckFunc asserting the backend version count.
//
// Why: Terraform state only tracks the current version; verifying retain_versions requires querying
// the backend object store directly in live/recording runs.
//
// How: Skips when VCR replay is active; otherwise lists stored object versions and asserts len(versions) == expectedCount.
//
// Parameters:
//   - namespace: F5 XC namespace of the stored object
//   - name: stored object name (without namespace prefix)
//   - expectedCount: expected number of retained versions on the backend
//
// Returns:
//   - resource.TestCheckFunc: check function evaluated after the TestStep applies
//
// Example:
//
//	Check: testAccCheckApiDefinitionUploaderRetainedVersions("shared", "my-api", 2)
func testAccCheckApiDefinitionUploaderRetainedVersions(namespace, name string, expectedCount int) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		if acctest.IsVCRReplaying() {
			return nil
		}

		apiClient, err := newAccTestUploaderAPIClient()
		if err != nil {
			return fmt.Errorf("failed to build API client for version check: %w", err)
		}

		versions, err := fetchUploaderBackendVersions(context.Background(), apiClient, namespace, name)
		if err != nil {
			return fmt.Errorf("failed to list backend versions for %s/%s: %w", namespace, name, err)
		}

		if len(versions) != expectedCount {
			return fmt.Errorf("expected %d retained backend version(s) for %s/%s, got %d (%v)",
				expectedCount, namespace, name, len(versions), versions)
		}

		return nil
	}
}

// testAccCheckApiDefinitionUploaderDestroy returns a TestCheckFunc verifying all versions are removed after destroy.
//
// Why: Ensures force_delete = true cleans up all stored object versions from the F5 XC backend on destroy.
//
// How: Skips when VCR replay is active; otherwise verifies no stored object versions remain for namespace/name.
//
// Parameters:
//   - namespace: F5 XC namespace of the stored object
//   - name: stored object name (without namespace prefix)
//
// Returns:
//   - resource.TestCheckFunc: destroy verification function for resource.TestCase.CheckDestroy
//
// Example:
//
//	CheckDestroy: testAccCheckApiDefinitionUploaderDestroy("shared", "my-api")
func testAccCheckApiDefinitionUploaderDestroy(namespace, name string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		if acctest.IsVCRReplaying() {
			return nil
		}

		apiClient, err := newAccTestUploaderAPIClient()
		if err != nil {
			return fmt.Errorf("failed to build API client for destroy check: %w", err)
		}

		versions, err := fetchUploaderBackendVersions(context.Background(), apiClient, namespace, name)
		if err != nil {
			return fmt.Errorf("failed to verify destroy for %s/%s: %w", namespace, name, err)
		}

		if len(versions) > 0 {
			return fmt.Errorf("expected stored object %s/%s to be completely destroyed, still found versions: %v",
				namespace, name, versions)
		}

		return nil
	}
}

// testAccResourceManualF5XCApiDefinitionUploaderConfigWithForceDelete renders HCL with retain_versions and force_delete.
//
// Why: Supports acceptance test steps that explicitly configure both retain_versions and force_delete.
//
// How: Formats a complete f5xc_api_definition_uploader resource block using the supplied parameters.
//
// Parameters:
//   - name: resource metadata.name
//   - namespace: resource metadata.namespace
//   - specContent: raw OpenAPI string assigned to string_value
//   - retainVersions: retain_versions attribute value
//   - forceDelete: force_delete attribute value
//
// Returns:
//   - string: Terraform HCL configuration snippet
//
// Example:
//
//	cfg := testAccResourceManualF5XCApiDefinitionUploaderConfigWithForceDelete("my-api", "shared", spec, 2, true)
func testAccResourceManualF5XCApiDefinitionUploaderConfigWithForceDelete(
	name, namespace, specContent string,
	retainVersions int,
	forceDelete bool,
) string {
	return fmt.Sprintf(`
resource "f5xc_api_definition_uploader" "test" {
  metadata = {
    name      = %q
    namespace = %q
  }
  string_value    = %q
  retain_versions = %d
  force_delete    = %t
}
`, name, namespace, specContent, retainVersions, forceDelete)
}
