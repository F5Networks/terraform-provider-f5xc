# Example: Active Network Policies - Policy

resource "f5xc_network_policy" "policy_one" {
  name      = "example-active-network-policy-one-m9n7y06c"
  namespace = "default"

  endpoint = {
    endpoint_choice = {
      prefix_list = {
        prefixes = ["10.10.0.0/16"]
      }
    }
  }
}


resource "f5xc_active_network_policies" "example" {
  namespace = "default"

  network_policies = [
    {
      name      = f5xc_network_policy.policy_one.name
      namespace = f5xc_network_policy.policy_one.namespace
    }
  ]
}
