resource "f5xc_healthcheck" "example" {
  name      = "example_healthcheck"
  namespace = "example_namespace"
  health_check = {
    http_health_check = {
      http_health_check = "example_http_health_check"
    }
  }
}
