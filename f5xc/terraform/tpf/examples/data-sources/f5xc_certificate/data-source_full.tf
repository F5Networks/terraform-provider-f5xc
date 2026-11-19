# Example: Certificate - Full

data "f5xc_certificate" "example" {
  name      = f5xc_certificate.example.name
  namespace = f5xc_certificate.example.namespace
}
