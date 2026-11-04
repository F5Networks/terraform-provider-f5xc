# Example: Network Connector - Full

data "f5xc_network_connector" "example" {
  name      = f5xc_network_connector.example.name
  namespace = f5xc_network_connector.example.namespace
}
