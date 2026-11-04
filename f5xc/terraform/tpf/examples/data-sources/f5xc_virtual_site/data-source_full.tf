# Example: Virtual Site - Full

data "f5xc_virtual_site" "example" {
  name      = f5xc_virtual_site.example.name
  namespace = f5xc_virtual_site.example.namespace
}
