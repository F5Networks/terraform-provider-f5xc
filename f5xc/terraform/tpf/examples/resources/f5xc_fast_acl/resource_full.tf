# Example: Fast Acl - Full

resource "f5xc_fast_acl" "example" {
  name        = "example-fast-acl"
  namespace   = "system"
  description = "Full fast ACL example"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform"
  }

  site_choice = {
    site_acl = {
      network_choice = {
        outside_network = true
      }
      vip_choice = {
        all_services = true
      }
      fast_acl_rules = [
        {
          metadata = {
            name        = "rule-allow"
            description = "allow rule"
          }
          action = {
            action_choice = {
              simple_action = ["ALLOW"]
            }
          }
          port = [
            {
              port_value_type_choice = {
                user_defined = 8080
              }
            }
          ]
          source_choice = {
            prefix = {
              prefix = ["10.0.0.0/8"]
            }
          }
        },
        {
          metadata = {
            name = "rule-deny"
          }
          action = {
            action_choice = {
              simple_action = ["DENY"]
            }
          }
          port = [
            {
              port_value_type_choice = {
                dns = true
              }
            }
          ]
          source_choice = {
            prefix = {
              prefix = ["192.168.0.0/16"]
            }
          }
        }
      ]
    }
  }
}
