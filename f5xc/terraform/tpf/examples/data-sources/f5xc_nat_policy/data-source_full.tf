# Example: Nat Policy - Full

data "f5xc_nat_policy" "example" {
  name      = f5xc_nat_policy.example.name
  namespace = f5xc_nat_policy.example.namespace
}
