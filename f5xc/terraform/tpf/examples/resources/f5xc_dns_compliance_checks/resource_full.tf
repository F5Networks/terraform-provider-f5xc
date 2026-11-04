# Example: Dns Compliance Checks - Full

resource "f5xc_dns_compliance_checks" "example" {
  name        = "example-dns-compliance-checks"
  namespace   = "default"
  description = "DNS compliance checks example"

  domain_denylist                      = ["malicious-test.com"]
  disallowed_query_type_list           = ["IQUERY", "STATUS"]
  disallowed_resource_record_type_list = ["MD", "MF"]

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform-team"
  }
}
