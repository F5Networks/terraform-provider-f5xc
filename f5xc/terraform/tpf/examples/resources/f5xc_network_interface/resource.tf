resource "f5xc_network_interface" "example" {
  name         = "example_network_interface"
  namespace    = "example_namespace"
  dhcp_address = ["ENUM_VALUE"]
  DHCP_server  = ["ENUM_VALUE"]
  type         = ["ENUM_VALUE"]
  device_name  = "example_network_interface"
}
