# Example: Bgp Routing Policy - Full

resource "f5xc_bgp_routing_policy" "example" {
  name        = "example-bgp-routing-policy"
  namespace   = "system"
  description = "BGP routing policy example"

  labels = {
    env  = "acceptance"
    team = "example-team"
  }

  annotations = {
    owner = "network"
  }

  rules = [{
    match = {
      type_of_match_choice = {
        ip_prefixes = {
          prefixes = [{
            ip_prefixes = "10.20.0.0/16"
            prefix_length_match_choice = {
              exact_match = true
            }
          }]
        }
      }
    }
    action = {
      action_type_choice = {
        local_preference = 150
      }
    }
  }]
}
