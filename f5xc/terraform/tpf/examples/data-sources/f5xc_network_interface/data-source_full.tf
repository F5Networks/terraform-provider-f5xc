# Example: Network Interface - Full

data "f5xc_network_interface" "example" {
  name      = f5xc_network_interface.example.name
  namespace = f5xc_network_interface.example.namespace
}
