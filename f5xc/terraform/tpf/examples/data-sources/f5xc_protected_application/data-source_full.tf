# Example: Protected Application - Full

data "f5xc_protected_application" "example" {
  name      = f5xc_protected_application.example.name
  namespace = f5xc_protected_application.example.namespace
}
