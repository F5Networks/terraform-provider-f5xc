# Example: Http Loadbalancer - Full

data "f5xc_http_loadbalancer" "example" {
  name      = f5xc_http_loadbalancer.example.name
  namespace = f5xc_http_loadbalancer.example.namespace
}
