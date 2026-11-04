# Example: Endpoint - Full

resource "f5xc_endpoint" "example" {
  name              = "example-endpoint"
  namespace         = "default"
  description       = "Full endpoint example"
  protocol          = "TCP"
  port              = 8443
  health_check_port = 9443

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform"
  }

  endpoint_address_choice = {
    dns_name_advanced = {
      name = "full.example.com"

      ttl_choice = {
        refresh_interval = 300
      }
    }
  }

  where = {
    ref_or_selector_choice = {
      virtual_site = {
        ref = [{
          name      = "ves-io-all-res"
          namespace = "shared"
          tenant    = "customer1"
        }]
        network_type = ["VIRTUAL_NETWORK_SITE_LOCAL"]
        internet_vip_choice = {
          disable_internet_vip = true
        }
      }
    }
  }

  snat_pool = {
    snat_pool_choice = {
      no_snat_pool = true
    }
  }
}
