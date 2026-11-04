# Example: Certificate Chain - Full

data "f5xc_certificate_chain" "example" {
  name      = f5xc_certificate_chain.example.name
  namespace = f5xc_certificate_chain.example.namespace
}
