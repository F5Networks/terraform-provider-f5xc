# Example: Protocol Inspection - Basic

resource "f5xc_protocol_inspection" "example" {
  name      = "example-proto-inspect"
  namespace = "default"

  action = ["ALLOW"]

  enable_disable_compliance_checks = {
    compliance_check_choice = {
      disable_compliance_checks = true
    }
  }

  enable_disable_signatures = {
    signature_choice = {
      enable_signature = true
    }
  }
}
