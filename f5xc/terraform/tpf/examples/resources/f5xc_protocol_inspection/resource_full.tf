# Example: Protocol Inspection - Full

resource "f5xc_dns_compliance_checks" "example" {
  name      = "example-dns-compliance"
  namespace = "default"

  domain_denylist = ["blocked-domain.com", "forbidden-domain.com"]
}

resource "f5xc_protocol_inspection" "example" {
  name        = "example-proto-inspect"
  namespace   = "default"
  description = "Full protocol inspection example"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "example-team"
  }

  action = ["DENY"]

  enable_disable_compliance_checks = {
    compliance_check_choice = {
      enable_compliance_checks = {
        name      = f5xc_dns_compliance_checks.example.name
        namespace = "default"
      }
    }
  }

  enable_disable_signatures = {
    signature_choice = {
      disable_signature = true
    }
  }
}
