# Example: Crl - Basic

resource "f5xc_crl" "example" {
  name           = "example-crl-4g2j00ut"
  namespace      = "default"
  server_address = "mycrl.example.com"
}
