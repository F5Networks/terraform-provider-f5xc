# Example: Network Policy Rule - Full

data "f5xc_network_policy_rule" "example" {
  name      = f5xc_network_policy_rule.example.name
  namespace = f5xc_network_policy_rule.example.namespace
}
