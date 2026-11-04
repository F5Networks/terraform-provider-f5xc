# Example: Protected Domain - Full

resource "f5xc_protected_domain" "example" {
  name             = "example-protected-domain"
  namespace        = "system"
  protected_domain = "example-protected-domain.com"
  description      = "Full protected domain example"

  labels = {
    env  = "example"
    team = "platform"
  }

  annotations = {
    owner = "example-owner"
  }
}
