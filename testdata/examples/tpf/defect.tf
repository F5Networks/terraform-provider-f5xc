/*
// XC-19280
resource "f5xc_network_interface" "interface1" {
  name         = "ck-test-interface-1"
  namespace    = "system"
  type         = ["NETWORK_INTERFACE_ETHERNET"]
  dhcp_server  = ["NETWORK_INTERFACE_DHCP_SERVER_DISABLE"]
  dhcp_address = ["NETWORK_INTERFACE_DHCP_DISABLE"]
  device_name  = "eth0"
}
*/

/*
resource "f5xc_api_discovery" "apid1" {
  name      = "ck-apid-1"
  namespace = "default"
  custom_auth_types = [
    {
      parameter_type = ["HEADER"]
      parameter_name = "auth"
    }
  ]
}
*/
