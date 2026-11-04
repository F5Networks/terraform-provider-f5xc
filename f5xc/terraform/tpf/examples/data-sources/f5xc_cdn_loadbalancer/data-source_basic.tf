# Example: Cdn Loadbalancer - Basic

data "f5xc_cdn_loadbalancer" "example" {
  name      = f5xc_cdn_loadbalancer.example.name
  namespace = f5xc_cdn_loadbalancer.example.namespace
}
