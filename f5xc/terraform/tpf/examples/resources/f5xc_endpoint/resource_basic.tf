# Example: Endpoint - Basic

resource "f5xc_endpoint" "example" {
  name      = "example-endpoint"
  namespace = "default"

  endpoint_address_choice = {
    ip = "10.10.10.10"
  }
}
