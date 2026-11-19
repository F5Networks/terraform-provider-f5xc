# Example: Dns Load Balancer - Full

data "f5xc_dns_load_balancer" "example" {
  name      = f5xc_dns_load_balancer.example.name
  namespace = f5xc_dns_load_balancer.example.namespace
}
