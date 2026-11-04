# Example: Discovery - Basic

resource "f5xc_discovery" "example" {
  name        = "example-discovery"
  namespace   = "default"
  description = "Discovery basic example"

  cluster_identifier_choice = {
    no_cluster_id = true
  }

  where = {
    ref_or_selector_choice = {
      virtual_site = {
        internet_vip_choice = {
          disable_internet_vip = true
        }
        ref = [{
          name      = "ves-io-all-res"
          namespace = "shared"
        }]
      }
    }
  }

  discovery_choice = {
    discovery_consul = {
      access_info = {
        connection_info = {
          api_server = "consul.example.com:8500"
        }
      }
      publish_info = {
        publish_choice = {
          publish = true
        }
      }
    }
  }
}
