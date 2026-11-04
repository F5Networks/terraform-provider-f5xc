# Example: Gcp Vpc Site - Full

data "f5xc_gcp_vpc_site" "example" {
  name      = f5xc_gcp_vpc_site.example.name
  namespace = f5xc_gcp_vpc_site.example.namespace
}
