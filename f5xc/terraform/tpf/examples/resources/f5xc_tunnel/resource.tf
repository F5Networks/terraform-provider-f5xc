resource "f5xc_tunnel" "example" {
  name        = "example_tunnel"
  namespace   = "example_namespace"
  tunnel_type = ["ENUM_VALUE"]
  local_ip    = {}
  remote_ip   = {}
  tunnel_attribute_choice = {
    default_tunnel_attribute = true
  }
}
