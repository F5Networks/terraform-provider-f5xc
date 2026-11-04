# Example: Service Policy Rule - Full

data "f5xc_service_policy_rule" "example" {
  name      = f5xc_service_policy_rule.example.name
  namespace = f5xc_service_policy_rule.example.namespace
}
