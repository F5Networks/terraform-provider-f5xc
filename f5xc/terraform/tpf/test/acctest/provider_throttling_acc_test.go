// Copyright (c) 2026 F5, Inc. All rights reserved.

//go:build acceptance

package acctest_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/test/acctest"
)

// TestAccProviderThrottling_DisabledByDefault verifies that omitting the throttling
// block passes provider schema validation.
func TestAccProviderThrottling_DisabledByDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: `
provider "f5xc" {
  url       = "https://example.com"
  api_token = "dummy-token"
}

data "f5xc_namespace" "test" {
  name = "system"
}
`,
				// Passing provider schema validation leads to plan execution (which fails due to dummy URL).
				ExpectError: regexp.MustCompile(`(?s)(dial tcp|no such host|Client Error)`),
			},
		},
	})
}

// TestAccProviderThrottling_EnabledWithDefaults verifies that declaring throttling = {}
// passes provider schema validation with default parameters.
func TestAccProviderThrottling_EnabledWithDefaults(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: `
provider "f5xc" {
  url       = "https://example.com"
  api_token = "dummy-token"
  throttling = {}
}

data "f5xc_namespace" "test" {
  name = "system"
}
`,
				ExpectError: regexp.MustCompile(`(?s)(dial tcp|no such host|Client Error)`),
			},
		},
	})
}

// TestAccProviderThrottling_CustomParameters verifies valid custom throttling parameter combinations.
func TestAccProviderThrottling_CustomParameters(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: `
provider "f5xc" {
  url       = "https://example.com"
  api_token = "dummy-token"
  throttling = {
    max_retries         = 10
    initial_backoff     = "500ms"
    max_backoff         = "30s"
    multiplier          = 2.5
    jitter_ratio        = 0.3
    respect_retry_after = false
  }
}

data "f5xc_namespace" "test" {
  name = "system"
}
`,
				ExpectError: regexp.MustCompile(`(?s)(dial tcp|no such host|Client Error)`),
			},
		},
	})
}

// TestAccProviderThrottling_ZeroRetries verifies max_retries = 0 is accepted by schema validation.
func TestAccProviderThrottling_ZeroRetries(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: `
provider "f5xc" {
  url       = "https://example.com"
  api_token = "dummy-token"
  throttling = {
    max_retries = 0
  }
}

data "f5xc_namespace" "test" {
  name = "system"
}
`,
				ExpectError: regexp.MustCompile(`(?s)(dial tcp|no such host|Client Error)`),
			},
		},
	})
}

// TestAccProviderThrottling_InvalidNegativeRetries verifies that negative max_retries
// is caught by schema validation.
func TestAccProviderThrottling_InvalidNegativeRetries(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: `
provider "f5xc" {
  url       = "https://example.com"
  api_token = "dummy-token"
  throttling = {
    max_retries = -1
  }
}

data "f5xc_namespace" "test" {
  name = "system"
}
`,
				ExpectError: regexp.MustCompile(`(?s)must be at least 0`),
			},
		},
	})
}

// TestAccProviderThrottling_InvalidJitterRatio verifies that jitter_ratio out of range [0, 1]
// is caught by schema validation.
func TestAccProviderThrottling_InvalidJitterRatio(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: `
provider "f5xc" {
  url       = "https://example.com"
  api_token = "dummy-token"
  throttling = {
    jitter_ratio = 1.5
  }
}

data "f5xc_namespace" "test" {
  name = "system"
}
`,
				ExpectError: regexp.MustCompile(`(?s)jitter_ratio.*must be between`),
			},
		},
	})
}
