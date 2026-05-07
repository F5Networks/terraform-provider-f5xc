resource "f5xc_healthcheck" "example" {
  name      = "example_healthcheck"
  namespace = "example_namespace"
  health_check = {
    tcp_health_check = {
      tcp_health_check = "example_tcp_health_check"
    }
  }
}
