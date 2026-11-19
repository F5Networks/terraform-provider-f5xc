# Example: Network Firewall - Full

data "f5xc_network_firewall" "example" {
  name      = f5xc_network_firewall.example.name
  namespace = f5xc_network_firewall.example.namespace
}
