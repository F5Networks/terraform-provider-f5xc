resource "f5xc_dns_lb_pool" "example" {
  name      = "example_dns_lb_pool"
  namespace = "example_namespace"
  pool_type_choice = {
    a_pool = {
      a_pool = "example_a_pool"
    }
  }
  load_balancing_mode = ["ENUM_VALUE"]
  ttl_choice = {
    ttl = 1
  }
}
