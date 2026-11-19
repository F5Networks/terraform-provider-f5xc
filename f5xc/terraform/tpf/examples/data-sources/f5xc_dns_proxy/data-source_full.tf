# Example: Dns Proxy - Full

data "f5xc_dns_proxy" "example" {
  name      = f5xc_dns_proxy.example.name
  namespace = f5xc_dns_proxy.example.namespace
}
