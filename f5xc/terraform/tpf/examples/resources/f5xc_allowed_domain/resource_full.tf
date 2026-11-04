# Example: Allowed Domain - Full

resource "f5xc_allowed_domain" "example" {
  name           = "example-domain-6lkr6d7y"
  namespace      = "system"
  allowed_domain = "full-6lkr6d7y.allowed-test.com"
  description    = "Secure domain with full configuration"

  labels = {
    env      = "production"
    security = "high"
  }

  annotations = {
    owner      = "platform-team"
    managed-by = "terraform"
  }
}
