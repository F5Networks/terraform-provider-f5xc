# Example: Secret Policy - Basic

data "f5xc_secret_policy" "example" {
  name      = f5xc_secret_policy.example.name
  namespace = f5xc_secret_policy.example.namespace
}