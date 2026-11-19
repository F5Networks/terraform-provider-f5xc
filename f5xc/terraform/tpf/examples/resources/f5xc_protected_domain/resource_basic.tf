# Example: Protected Domain - Basic

resource "f5xc_protected_domain" "example" {
  name             = "example-protected-domain"
  namespace        = "system"
  protected_domain = "example-protected-domain.com"
}
