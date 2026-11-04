# Example: Tcp Loadbalancer - Full

data "f5xc_tcp_loadbalancer" "example" {
  name      = f5xc_tcp_loadbalancer.example.name
  namespace = f5xc_tcp_loadbalancer.example.namespace
}
