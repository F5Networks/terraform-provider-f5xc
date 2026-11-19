# Example: Network Policy - Full

resource "f5xc_network_policy" "example" {
  name        = "example-network-policy"
  namespace   = "default"
  description = "Full network policy example"

  labels = {
    env  = "example"
    team = "platform"
  }

  annotations = {
    owner = "example-owner"
  }

  endpoint = {
    endpoint_choice = {
      prefix_list = {
        prefixes = ["10.0.0.0/24"]
      }
    }
  }

  rule_choice = {
    rules = {
      ingress_rules = [
        {
          metadata = {
            name        = "example-ingress-rule"
            description = "Ingress rule example"
          }
          action = ["DENY"]
          adv_action = {
            action = ["LOG"]
          }
          traffic_choice = {
            protocol_port_range = {
              protocol    = "TCP"
              port_ranges = ["80", "443"]
            }
          }
          other_endpoint_choice = {
            prefix_list = {
              prefixes = ["192.168.1.0/24"]
            }
          }
          label_matcher = {
            keys = ["environment"]
          }
        }
      ]
      egress_rules = [
        {
          metadata = {
            name = "example-egress-rule"
          }
          action = ["DENY"]
          traffic_choice = {
            all_udp_traffic = true
          }
          other_endpoint_choice = {
            any = true
          }
        }
      ]
    }
  }
}
