# Example: NGINX Service Discovery - Full

data "f5xc_nginx_service_discovery" "example" {
  name      = f5xc_nginx_service_discovery.example.name
  namespace = f5xc_nginx_service_discovery.example.namespace
}
