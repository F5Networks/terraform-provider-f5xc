# Example: Protected Application - Basic

resource "f5xc_protected_application" "example" {
  name      = "example-protected-application"
  namespace = "default"
  region    = ["US"]

  connector_type_choice = {
    big_ip_iapp = true
  }
}
