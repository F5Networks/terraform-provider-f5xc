# Example: Network Policy View - Basic

resource "f5xc_network_policy_view" "example" {
  name      = "example-npv"
  namespace = "system"

  endpoint = {
    endpoint_choice = {
      outside_endpoints = true
    }
  }
}
