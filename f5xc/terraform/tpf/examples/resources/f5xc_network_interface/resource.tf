resource "f5xc_network_interface" "example" {
  name         = "example_network_interface"
  namespace    = "example_namespace"
  type         = ["ENUM_VALUE"]
  DHCP_server  = ["ENUM_VALUE"]
  dhcp_address = ["ENUM_VALUE"]
  device_name  = "example_network_interface"
}
