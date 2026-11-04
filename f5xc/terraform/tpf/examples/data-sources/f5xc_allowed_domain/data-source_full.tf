# Example: Allowed Domain - Full

data "f5xc_allowed_domain" "example" {
  namespace      = f5xc_allowed_domain.example.namespace
  allowed_domain = f5xc_allowed_domain.example.allowed_domain
}
