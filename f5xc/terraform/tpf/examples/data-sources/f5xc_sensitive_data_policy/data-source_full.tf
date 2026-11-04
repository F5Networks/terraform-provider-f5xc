# Example: Sensitive Data Policy - Full

data "f5xc_sensitive_data_policy" "example" {
  name      = f5xc_sensitive_data_policy.example.name
  namespace = f5xc_sensitive_data_policy.example.namespace
}
