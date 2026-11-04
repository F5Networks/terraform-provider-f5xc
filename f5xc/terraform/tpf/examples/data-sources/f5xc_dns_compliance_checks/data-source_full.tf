# Example: Dns Compliance Checks - Full

data "f5xc_dns_compliance_checks" "example" {
  name      = f5xc_dns_compliance_checks.example.name
  namespace = f5xc_dns_compliance_checks.example.namespace
}
