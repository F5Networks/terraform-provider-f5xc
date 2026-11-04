# Example: Appstack Site - Full

data "f5xc_appstack_site" "example" {
  name      = f5xc_appstack_site.example.name
  namespace = f5xc_appstack_site.example.namespace
}
