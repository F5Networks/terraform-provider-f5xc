# Example: Trusted Ca List - Full

data "f5xc_trusted_ca_list" "example" {
  name      = f5xc_trusted_ca_list.example.name
  namespace = f5xc_trusted_ca_list.example.namespace
}
