# Example: Dns Lb Pool - Full

data "f5xc_dns_lb_pool" "example" {
  name      = f5xc_dns_lb_pool.example.name
  namespace = f5xc_dns_lb_pool.example.namespace
}
