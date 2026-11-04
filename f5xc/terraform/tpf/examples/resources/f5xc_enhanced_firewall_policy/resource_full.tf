# Example: Enhanced Firewall Policy - Full

resource "f5xc_enhanced_firewall_policy" "example" {
  name        = "example-enhanced-firewall-policy"
  namespace   = "system"
  description = "Full enhanced firewall policy example"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform"
  }

  rule_choice = {
    rule_list = {
      rules = [
        {
          metadata = {
            name        = "example-rule-full"
            description = "Full rule example"
          }

          action_choice = {
            allow = true
          }

          source_choice = {
            source_prefix_list = {
              prefixes      = ["10.0.0.0/8"]
              ipv6_prefixes = ["2001:0db8:0002:0000:0000:0000:0000:0000/64"]
            }
          }

          destination_choice = {
            destination_prefix_list = {
              prefixes      = ["172.16.0.0/12"]
              ipv6_prefixes = ["2001:0db8:0003:0000:0000:0000:0000:0000/64"]
            }
          }

          traffic_choice = {
            all_tcp_traffic = true
          }

          advanced_action = {
            action = ["NOLOG"]
          }

          label_matcher = {
            keys = ["env"]
          }
        }
      ]
    }
  }

  segment_policy = {
    src_segment_choice = {
      src_any = true
    }

    dst_segment_choice = {
      dst_any = true
    }
  }
}
