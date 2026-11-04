# Example: Service Credential - Namespace Roles

resource "f5xc_service_credential" "example" {
  name            = "example-service-credential"
  namespace       = "system"
  expiration_days = 7

  service_credential_choice = {
    api_token = true
  }

  namespace_roles = [
    {
      namespace = "system"
      role      = "ves-io-monitor-role"
    }
  ]
}
