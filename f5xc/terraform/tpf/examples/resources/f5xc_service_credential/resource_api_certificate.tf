# Example: Service Credential - Api Certificate

resource "f5xc_service_credential" "example" {
  name            = "example-service-credential"
  namespace       = "system"
  expiration_days = 30

  service_credential_choice = {
    api_certificate = {
      password = "ExamplePass1!"
    }
  }

  namespace_roles = [
    {
      namespace = "system"
      role      = "ves-io-monitor-role"
    }
  ]
}
