# Example: Network Policy View - Full

data "f5xc_network_policy_view" "example" {
  name      = f5xc_network_policy_view.example.name
  namespace = f5xc_network_policy_view.example.namespace
}
