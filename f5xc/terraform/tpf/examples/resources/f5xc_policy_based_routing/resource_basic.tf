# Example: Policy Based Routing - Basic

resource "f5xc_policy_based_routing" "example" {
  name      = "example-pbr"
  namespace = "default"

  policy_choice = {
    network_pbr = {}
  }

  forwarding_class_list = [{
    name      = "example-fc"
    namespace = "default"
  }]
}
