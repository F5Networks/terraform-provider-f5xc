resource "f5xc_dns_domain" "example" {
  namespace = "example_namespace"
  name      = "example_dns_domain"
  domain_choice = {
    volterra_managed = true
  }
}
