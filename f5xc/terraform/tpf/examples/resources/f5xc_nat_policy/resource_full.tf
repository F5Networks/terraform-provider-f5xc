# Example: Nat Policy - Full

resource "f5xc_securemesh_site_v2" "site" {
  name      = "example-nat-site-x8bgfj-164000"
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

resource "f5xc_nat_policy" "example" {
  name        = "example-nat-policy"
  namespace   = "default"
  description = "Full NAT policy example"

  labels      = { env = "example", owner = "network" }
  annotations = { owner = "platform-team" }

  applies_to_choice = {
    site = {
      refs = [{ name = f5xc_securemesh_site_v2.site.name, namespace = f5xc_securemesh_site_v2.site.namespace }]
    }
  }

  rules = [
    {
      name          = "example-rule-full"
      enable_choice = { enable = true }
      scope_choice  = { node_interface = {} }

      criteria = {
        protocol_choice = {
          udp = {
            source_port      = { port_match_choice = { port_ranges = "1000-2000" } }
            destination_port = { port_match_choice = { port = 53 } }
          }
        }
        network_choice   = { site_local_network = true }
        source_cidr      = ["10.20.0.0/16"]
        destination_cidr = ["172.16.0.0/12"]
      }

      action = {
        source_nat_choice = {
          dynamic = {
            pool_choice = {
              pools = {
                prefixes      = ["192.168.20.0/24"]
                ipv6_prefixes = ["fd48:fa09:d9d4:0000:0000:0000:0000:0000/48"]
              }
            }
          }
        }
      }
    }
  ]
}
