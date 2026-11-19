// Copyright (c) 2026 F5, Inc. All rights reserved.

//go:build acceptance

package resource

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	acctest "gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/test/acctest"
)

type cloudLinkAWSProfile struct {
	CredName        string
	ConnectionID    string
	AmazonAddress   string
	CustomerAddress string
	Region          string
	CustomASN       int
	Vlan            int
}

func testAccRunSuffix(t *testing.T) string {
	t.Helper()

	// In live mode, add a time-based suffix so repeated local runs do not collide
	// with resources that are still deleting. VCR modes must stay deterministic.
	if strings.TrimSpace(os.Getenv("VCR_MODE")) == "" {
		return fmt.Sprintf("%s-%d", acctest.RandString(t, 6), time.Now().UnixNano()%1000000)
	}

	return acctest.RandString(t, 8)
}

func testAccIsCRTTenant() bool {
	return strings.Contains(strings.ToLower(os.Getenv("F5XCProvider_URL")), ".crt.volterra.us")
}

// blindFoldKeyB64 returns the correct blindfold-encrypted key blob for the
// tenant identified by F5XCProvider_URL.
//
// Mapping:
//
//	playground.staging.volterra.us → testBlindFoldKeyB64Staging
//	customer1.crt.volterra.us      → testBlindFoldKeyB64Crt
func blindFoldKeyB64(t testing.TB) string {
	t.Helper()

	if v := strings.TrimSpace(os.Getenv("F5XC_BLINDFOLD_KEY_B64")); v != "" {
		return v
	}

	u := os.Getenv("F5XCProvider_URL")
	switch {
	case strings.Contains(u, "playground.staging.volterra.us"):
		return testBlindFoldKeyB64Staging
	case strings.Contains(u, "customer1.crt.volterra.us"):
		return testBlindFoldKeyB64Crt
	default:
		t.Skipf("blindFoldKeyB64: no blindfold key configured for URL %q — set F5XC_BLINDFOLD_KEY_B64 or add an entry to helpers.go", u)
		return ""
	}
}

// defaultCloudLinkAWSProfile returns the default pre-existing AWS Direct Connect
// profile used by cloud_link acceptance tests for known tenants.
//
// Mapping:
//
//	customer1.crt.volterra.us -> orch-test-user-site-views-aws-vpc-dc-new + ves-cloudlink-crt-1-2
func defaultCloudLinkAWSProfile(t testing.TB) cloudLinkAWSProfile {
	t.Helper()

	u := os.Getenv("F5XCProvider_URL")
	switch {
	case strings.Contains(u, "customer1.crt.volterra.us"):
		return cloudLinkAWSProfile{
			CredName:        "orch-test-user-site-views-aws-vpc-dc-new",
			ConnectionID:    "dxcon-fgjqgra5",
			AmazonAddress:   "84.54.63.4/31",
			CustomerAddress: "84.54.63.5/31",
			Region:          "ap-southeast-1",
			CustomASN:       4293918901,
			Vlan:            354,
		}
	default:
		t.Skipf("defaultCloudLinkAWSProfile: no default AWS cloud_link profile configured for URL %q — set F5XC_CLOUD_LINK_AWS_CRED_NAME if only the credential differs, otherwise add a tenant mapping in helpers.go", u)
		return cloudLinkAWSProfile{}
	}
}

// defaultInfraprotectApprovedASNRef returns the default pre-approved ASN
// reference used by infraprotect_asn_prefix acceptance tests for known tenants.
func defaultInfraprotectApprovedASNRef(t testing.TB) string {
	t.Helper()

	if v := strings.TrimSpace(os.Getenv("F5XC_INFRAPROTECT_APPROVED_ASN_NAME")); v != "" {
		return v
	}

	u := os.Getenv("F5XCProvider_URL")
	return defaultInfraprotectApprovedASNRefName(u)
}

// defaultInfraprotectApprovedASNRefName returns the default pre-approved ASN
// name for a given tenant URL used by infraprotect_asn_prefix acceptance tests.
func defaultInfraprotectApprovedASNRefName(url string) string {
	switch {
	case strings.Contains(url, "customer1.crt.volterra.us"):
		return "rr-asn"
	default:
		return ""
	}
}

func testAccSensitivePlanChecks(resourceAddress string, attributePath tfjsonpath.Path) resource.ConfigPlanChecks {
	return resource.ConfigPlanChecks{
		PreApply: []plancheck.PlanCheck{
			plancheck.ExpectSensitiveValue(resourceAddress, attributePath),
		},
	}
}
