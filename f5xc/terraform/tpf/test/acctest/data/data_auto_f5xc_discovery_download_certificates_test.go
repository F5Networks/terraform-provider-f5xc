// Copyright (c) 2026 F5, Inc. All rights reserved.

//go:build acceptance

package data

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	acctest "gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/test/acctest"
	discoverySchema "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/discovery"
)

// TestAccDataSourceAutoF5XCDiscoveryDownloadCertificates_full verifies that the
// download data source returns a certificate bundle for a third-party discovery.
func TestAccDataSourceAutoF5XCDiscoveryDownloadCertificates_full(t *testing.T) {
	t.Parallel()

	name := "tf-acc-discovery-download-certificates-" + acctest.RandString(t, 8)
	namespace := "default"

	acctest.VcrTestWithOptions(t, resource.TestCase{
		PreCheck:                 func() { acctest.AccTestPreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories(t),
		CheckDestroy:             acctest.CheckDestroyProducerReplayAware(t, "f5xc_discovery", discoverySchema.ObjectType, "discovery"),
		Steps: []resource.TestStep{
			{
				Config: testAccDiscoveryDownloadCertificatesDataSourceConfig(name, namespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.f5xc_discovery_download_certificates.example", "name", name),
					resource.TestCheckResourceAttr("data.f5xc_discovery_download_certificates.example", "namespace", namespace),
					resource.TestCheckResourceAttrSet("data.f5xc_discovery_download_certificates.example", "data_discovery_download_certificates"),
				),
			},
		},
	}, acctest.VcrTestOptions{
		DocExampleCapture: true,
		DocExampleConfig:  testAccDiscoveryDownloadCertificatesDataSourceBlock,
	})
}

// testAccDiscoveryDownloadCertificatesDataSourceConfig creates the third-party
// discovery required by DownloadCertificates and reads its certificate bundle.
func testAccDiscoveryDownloadCertificatesDataSourceConfig(name, namespace string) string {
	siteName := name + "-site"
	return fmt.Sprintf(`
resource "f5xc_securemesh_site_v2" "site" {
  name      = %q
  namespace = "system"

  logs_receiver_choice = {
    logs_streaming_disabled = true
  }

  blocked_services_choice = {
    block_all_services = true
  }

  provider_choice = {
    kvm = {
      orchestration_choice = {
        not_managed = {}
      }
    }
  }
}

resource "f5xc_discovery" "example" {
  name        = %q
  namespace   = %q
  description = "Discovery certificate download example"

  cluster_identifier_choice = {
    no_cluster_id = true
  }

  where = {
    ref_or_selector_choice = {
      site = {
        internet_vip_choice = {
          disable_internet_vip = true
        }
        ref = [{
          name      = f5xc_securemesh_site_v2.site.name
          namespace = f5xc_securemesh_site_v2.site.namespace
        }]
      }
    }
  }

  discovery_choice = {
    discovery_third_party = {
      applications = [%q]
      source_cidr  = ["10.0.0.0/24"]
    }
  }
}
`+testAccDiscoveryDownloadCertificatesDataSourceBlock, siteName, name, namespace, name)
}

const testAccDiscoveryDownloadCertificatesDataSourceBlock = `
data "f5xc_discovery_download_certificates" "example" {
  name      = f5xc_discovery.example.name
  namespace = f5xc_discovery.example.namespace
}
`
