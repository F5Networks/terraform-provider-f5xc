# Example: Dns Domain - Full

resource "f5xc_dns_domain" "example" {
  name        = "example-dns-domain-3hjmzass.example.com"
  namespace   = "system"
  description = "DNS domain example"

  labels = {
    env  = "example"
    team = "platform"
  }

  annotations = {
    owner = "platform"
  }

  domain_choice = {
    volterra_managed = true
  }

  dnssec_mode = ["DNSSEC_DISABLE"]
}
