# Example: Fast Acls For Internet Vips - Basic

resource "f5xc_fast_acl" "acl_one" {
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


resource "f5xc_fast_acls_for_internet_vips" "example" {
  namespace = "system"

  fast_acls = [
    {
      name      = f5xc_fast_acl.acl_one.name
      namespace = f5xc_fast_acl.acl_one.namespace
    }
  ]
}
