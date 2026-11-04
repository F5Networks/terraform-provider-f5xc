# Example: Bgp - Full

resource "f5xc_bgp" "example" {
  name        = "example-bgp"
  namespace   = "system"
  description = "BGP example"

  labels = {
    env  = "example"
    team = "network"
  }

  annotations = {
    owner = "platform"
  }

  where = {
    ref_or_selector_choice = {
      site = {
        ref = [
          {
            name      = "example-site"
            namespace = "system"
          }
        ]
        network_type = ["VIRTUAL_NETWORK_SITE_LOCAL"]
        internet_vip_choice = {
          disable_internet_vip = true
        }
      }
    }
  }

  peers = [
    {
      metadata = {
        name        = "peer-1"
        description = "Primary peer"
      }
      bfd_choice = {
        bfd_enabled = {
          transmit_interval_milliseconds = 3000
          receive_interval_milliseconds  = 3000
          multiplier                     = 3
        }
      }
      type_choice = {
        external = {
          asn = 65001
          address_choice = {
            address = "192.168.1.1"
          }
          address_choice_v6_choice = {
            disable_v6 = true
          }
          interface_choice = {
            interface = {
              name      = "eth0"
              namespace = "system"
            }
          }
          auth_choice = {
            no_authentication = true
          }
          family_inet = {
            enable_choice = {
              enable = {
                aggregation = [
                  {
                    ip_prefix = "10.0.0.0/8"
                    options = [
                      {
                        aggregation_choice = {
                          summary_only = true
                        }
                      }
                    ]
                  }
                ]
              }
            }
          }
        }
      }
      passive_choice = {
        passive_mode_disabled = true
      }
      enable_choice = {
        routing_policies = {}
      }
    }
  ]

  bgp_parameters = {
    asn = 64512
    router_id_choice = {
      ip_address = "10.0.0.1"
    }
  }
}
