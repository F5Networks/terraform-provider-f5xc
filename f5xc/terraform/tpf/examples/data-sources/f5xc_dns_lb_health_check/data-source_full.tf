# Example: Dns Lb Health Check - Full

data "f5xc_dns_lb_health_check" "example" {
  name      = f5xc_dns_lb_health_check.example.name
  namespace = f5xc_dns_lb_health_check.example.namespace
}
