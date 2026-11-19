# Example: Protected Application - Full

resource "f5xc_protected_application" "example" {
  name        = "example-protected-application"
  namespace   = "default"
  description = "Full protected application example"
  region      = ["US"]

  labels = {
    env  = "example"
    team = "platform"
  }

  annotations = {
    owner = "example-owner"
  }

  connector_type_choice = {
    custom_connector = true
  }
}
