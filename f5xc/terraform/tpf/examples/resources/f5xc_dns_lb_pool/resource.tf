resource "f5xc_dns_lb_pool" "example" {
  name                = "example_dns_lb_pool"
  namespace           = "example_namespace"
  load_balancing_mode = ["ENUM_VALUE"]
  ttl_choice = {
    use_rrset_ttl = true
  }
  pool_type_choice = {
    aaaa_pool = {
      aaaa_pool = "example_aaaa_pool"
    }
  }
}
