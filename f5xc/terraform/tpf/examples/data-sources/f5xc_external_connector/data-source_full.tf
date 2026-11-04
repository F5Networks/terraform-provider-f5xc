# Example: External Connector - Full

data "f5xc_external_connector" "example" {
  name      = f5xc_external_connector.example.name
  namespace = f5xc_external_connector.example.namespace
}
