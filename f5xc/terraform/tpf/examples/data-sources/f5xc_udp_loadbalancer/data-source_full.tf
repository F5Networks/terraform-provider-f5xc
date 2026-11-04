# Example: Udp Loadbalancer - Full

data "f5xc_udp_loadbalancer" "example" {
  name      = f5xc_udp_loadbalancer.example.name
  namespace = f5xc_udp_loadbalancer.example.namespace
}
