# Example: Policy Based Routing - Full

resource "f5xc_policy_based_routing" "example" {
  name        = "example-pbr"
  namespace   = "default"
  description = "Full policy based routing example"

  labels = {
    env  = "example"
    team = "example-team"
  }

  annotations = {
    owner = "network-team"
  }

  policy_choice = {
    network_pbr = {
      source_choice = {
        any = true
      }
      network_pbr_rules = [{
        metadata = {
          name = "allow-all-tcp"
        }
        forwarding_class_list = [{
          name      = "example-fc"
          namespace = "default"
        }]
        destination_choice = {
          any = true
        }
        traffic_choice = {
          all_tcp_traffic = true
        }
      }]
    }
  }

  forwarding_class_list = [{
    name      = "example-fc"
    namespace = "default"
  }]
}
