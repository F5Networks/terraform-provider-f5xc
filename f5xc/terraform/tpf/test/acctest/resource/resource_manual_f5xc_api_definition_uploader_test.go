//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

//go:build acceptance

package resource

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	acctest "gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/test/acctest"
)

func TestAccResourceManualF5XCApiDefinitionUploader_basic(t *testing.T) {
	t.Parallel()

	namespace := "shared"
	name := "api-uploader-" + acctest.RandString(t, 8)
	specV1 := `{"openapi":"3.0.0","info":{"title":"Test API","version":"1.0.0"},"paths":{"/test":{"get":{"summary":"Test endpoint","responses":{"200":{"description":"Success"}}}}}}`
	specV2 := `{"openapi":"3.0.0","info":{"title":"Test API","version":"2.0.0"},"paths":{"/test":{"get":{"summary":"Test endpoint","responses":{"200":{"description":"Success"}}}}}}`

	resourceName := "f5xc_api_definition_uploader.test"

	acctest.VcrTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.AccTestPreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: testAccResourceManualF5XCApiDefinitionUploaderConfig(name, namespace, specV1, 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.namespace", namespace),
					resource.TestMatchResourceAttr(resourceName, "metadata.version", regexp.MustCompile(`^v\d+`)),
					resource.TestMatchResourceAttr(resourceName, "metadata.url", regexp.MustCompile(fmt.Sprintf(`/stored_objects/swagger/%s/`, name))),
					resource.TestCheckResourceAttrSet(resourceName, "content_hash"),
					resource.TestCheckResourceAttr(resourceName, "retain_versions", "2"),
					resource.TestCheckNoResourceAttr(resourceName, "previous_version_url"),
				),
			},
			// Step 2: Update with modified OpenAPI spec content
			{
				Config: testAccResourceManualF5XCApiDefinitionUploaderConfig(name, namespace, specV2, 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.namespace", namespace),
					resource.TestMatchResourceAttr(resourceName, "metadata.version", regexp.MustCompile(`^v\d+`)),
					resource.TestCheckResourceAttrSet(resourceName, "content_hash"),
					resource.TestCheckResourceAttrSet(resourceName, "previous_version_url"),
				),
			},
			// Step 3: Re-apply same config - must be idempotent
			{
				Config: testAccResourceManualF5XCApiDefinitionUploaderConfig(name, namespace, specV2, 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.namespace", namespace),
				),
			},
		},
	})
}

func testAccResourceManualF5XCApiDefinitionUploaderConfig(name, namespace, specContent string, retainVersions int) string {
	return fmt.Sprintf(`
resource "f5xc_api_definition_uploader" "test" {
  metadata = {
    name      = %q
    namespace = %q
  }
  string_value    = %q
  retain_versions = %d
}
`, name, namespace, specContent, retainVersions)
}
