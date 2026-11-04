# Example: Sensitive Data Policy - Full

resource "f5xc_sensitive_data_policy" "example" {
  name        = "example-sensitive-data-policy"
  namespace   = "default"
  description = "sensitive data policy full"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "example-owner"
  }

  compliances = ["PCI_DSS", "HIPAA"]

  disabled_predefined_data_types = ["credit_card", "ssn"]
}
