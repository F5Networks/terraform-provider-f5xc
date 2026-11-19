# Example: Dns Zone - Full

data "f5xc_dns_zone" "example" {
  name      = f5xc_dns_zone.example.name
  namespace = f5xc_dns_zone.example.namespace
}
