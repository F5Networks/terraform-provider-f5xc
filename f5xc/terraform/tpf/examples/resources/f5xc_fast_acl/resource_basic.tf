# Example: Fast Acl - Basic

resource "f5xc_fast_acl" "example" {
  name      = "example-fast-acl"
  namespace = "system"

  site_choice = {
    site_acl = {
      network_choice = {
        outside_network = true
      }
      vip_choice = {
        all_services = true
      }
    }
  }
}
