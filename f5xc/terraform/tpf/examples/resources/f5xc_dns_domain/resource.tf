resource "f5xc_dns_domain" "example" {
  name      = "example_dns_domain"
  namespace = "example_namespace"
  domain_choice = {
    volterra_managed = true
  }
}
