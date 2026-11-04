# Example: Policy Based Routing - Full

data "f5xc_policy_based_routing" "example" {
  name      = f5xc_policy_based_routing.example.name
  namespace = f5xc_policy_based_routing.example.namespace
}
