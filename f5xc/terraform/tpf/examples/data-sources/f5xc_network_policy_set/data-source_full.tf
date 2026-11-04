# Example: Network Policy Set - Full

data "f5xc_network_policy_set" "example" {
  name      = "default"
  namespace = "default"
}
