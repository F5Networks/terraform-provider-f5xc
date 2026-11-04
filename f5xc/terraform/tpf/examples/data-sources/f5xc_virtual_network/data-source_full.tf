# Example: Virtual Network - Full

data "f5xc_virtual_network" "example" {
  name      = f5xc_virtual_network.example.name
  namespace = f5xc_virtual_network.example.namespace
}
