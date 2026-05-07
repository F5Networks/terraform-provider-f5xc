resource "f5xc_virtual_site" "example" {
  name      = "example_virtual_site"
  namespace = "example_namespace"
  site_selector = {
    expressions = ["example_virtual_site"]
  }
  site_type = ["ENUM_VALUE"]
}
