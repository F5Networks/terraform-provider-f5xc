# Example: Virtual Network - Basic

resource "f5xc_virtual_network" "example" {
  name      = "example-virtual-network"
  namespace = "system"

  network_choice = {
    site_local_network = true
  }
}
