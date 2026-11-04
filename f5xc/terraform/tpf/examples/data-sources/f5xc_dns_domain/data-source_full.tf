# Example: Dns Domain - Full

data "f5xc_dns_domain" "example" {
  name      = f5xc_dns_domain.example.name
  namespace = f5xc_dns_domain.example.namespace
}
