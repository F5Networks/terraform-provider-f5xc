# Example: Protected Domain - Full

data "f5xc_protected_domain" "example" {
  namespace        = f5xc_protected_domain.example.namespace
  protected_domain = f5xc_protected_domain.example.protected_domain
}
