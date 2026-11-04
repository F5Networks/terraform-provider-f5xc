# Example: Virtual Network - Full

resource "f5xc_virtual_network" "example" {
  name        = "example-virtual-network"
  namespace   = "system"
  description = "Virtual network full example"

  labels = {
    env  = "example"
    team = "platform"
  }

  annotations = {
    owner = "example-owner"
  }

  network_choice = {
    site_local_network = true
  }

  static_routes = [
    {
      ip_prefixes = ["10.20.0.0/24"]
      next_hop_choice = {
        ip_address = "10.20.0.1"
      }
      attrs = ["ROUTE_ATTR_ADVERTISE"]
    }
  ]
}
