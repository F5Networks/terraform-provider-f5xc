# Example: Bigip Irule - Full

data "f5xc_bigip_irule" "example" {
  name      = f5xc_bigip_irule.example.backend_name
  namespace = f5xc_bigip_irule.example.namespace
}
