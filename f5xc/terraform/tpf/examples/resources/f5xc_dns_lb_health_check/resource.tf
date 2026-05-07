resource "f5xc_dns_lb_health_check" "example" {
  name      = "example_dns_lb_health_check"
  namespace = "example_namespace"
  health_check = {
    udp_health_check = {
      udp_health_check = "example_udp_health_check"
    }
  }
}
