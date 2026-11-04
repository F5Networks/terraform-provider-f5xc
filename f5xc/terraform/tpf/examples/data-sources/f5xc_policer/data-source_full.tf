# Example: Policer - Full

data "f5xc_policer" "example" {
  name      = f5xc_policer.example.name
  namespace = f5xc_policer.example.namespace
}
