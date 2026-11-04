# Example: Protocol Inspection - Full

resource "f5xc_dns_compliance_checks" "example" {
  name      = "example-dns-compliance"
  namespace = "terraform-acceptance-test"

  domain_denylist = ["blocked-domain.com", "forbidden-domain.com"]
}

resource "f5xc_protocol_inspection" "example" {
  name        = "example-proto-inspect"
  namespace   = "terraform-acceptance-test"
  description = "Full protocol inspection example"

  labels = {
    env = "staging"
  }

  annotations = {
    owner = "team-a"
  }

  action = ["DENY"]

  enable_disable_compliance_checks = {
    compliance_check_choice = {
      enable_compliance_checks = {
        name      = f5xc_dns_compliance_checks.example.name
        namespace = "terraform-acceptance-test"
      }
    }
  }

  enable_disable_signatures = {
    signature_choice = {
      disable_signature = true
    }
  }
}

data "f5xc_protocol_inspection" "example" {
  name      = f5xc_protocol_inspection.example.name
  namespace = f5xc_protocol_inspection.example.namespace
}
