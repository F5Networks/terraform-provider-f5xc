# Example: Network Policy - Basic

resource "f5xc_network_policy" "example" {
  name      = "example-network-policy"
  namespace = "default"

  endpoint = {
    endpoint_choice = {
      any = true
    }
  }
}
