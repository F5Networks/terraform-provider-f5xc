resource "f5xc_network_connector" "example" {
  name                 = "example_network_connector"
  namespace            = "example_namespace"
  connector_type       = ["ENUM_VALUE"]
  outside_network_type = ["ENUM_VALUE"]
  inside_network_type  = ["ENUM_VALUE"]
}
