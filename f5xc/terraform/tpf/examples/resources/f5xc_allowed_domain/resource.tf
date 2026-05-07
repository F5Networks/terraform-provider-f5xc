resource "f5xc_allowed_domain" "example" {
  name           = "example_allowed_domain"
  namespace      = "example_namespace"
  allowed_domain = "example_allowed_domain"
}
