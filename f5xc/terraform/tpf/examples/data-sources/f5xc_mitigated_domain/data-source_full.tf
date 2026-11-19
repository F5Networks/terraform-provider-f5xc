# Example: Mitigated Domain - Full

data "f5xc_mitigated_domain" "example" {
  namespace        = f5xc_mitigated_domain.example.namespace
  mitigated_domain = f5xc_mitigated_domain.example.mitigated_domain
}
