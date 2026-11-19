# Example: Mobile Base Config - Full

resource "f5xc_mobile_base_config" "example" {
  name        = "example-mobile-base-config"
  namespace   = "default"
  os          = ["ANDROID"]
  fqdn        = "a5u3nqr8.full.example.com"
  description = "Full mobile base config example"

  labels = {
    env   = "example"
    owner = "mobile"
  }

  annotations = {
    owner = "mobile-security-team"
  }
}
