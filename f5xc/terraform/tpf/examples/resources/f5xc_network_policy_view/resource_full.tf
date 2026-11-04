# Example: Network Policy View - Full

resource "f5xc_network_policy_view" "example" {
  name        = "example-npv"
  namespace   = "system"
  description = "Full network policy view example"

  labels = {
    env  = "example"
    team = "example-team"
  }

  annotations = {
    owner = "platform-team"
  }

  endpoint = {
    endpoint_choice = {
      outside_endpoints = true
    }
  }

  ingress_rules = [{
    metadata = {
      name = "allow-tcp"
    }
    action = ["ALLOW"]
    other_endpoint_choice = {
      any = true
    }
    traffic_choice = {
      all_tcp_traffic = true
    }
  }]

  egress_rules = [{
    metadata = {
      name = "deny-udp"
    }
    action = ["DENY"]
    other_endpoint_choice = {
      any = true
    }
    traffic_choice = {
      all_udp_traffic = true
    }
  }]
}
