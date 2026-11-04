# Example: Mitigated Domain - Full

resource "f5xc_mitigated_domain" "example" {
  name             = "example-mitigated-domain"
  namespace        = "system"
  mitigated_domain = "md-full-hnywyte8.mitigated-test.com"
  description      = "Full mitigated domain example"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform-team"
  }
}
