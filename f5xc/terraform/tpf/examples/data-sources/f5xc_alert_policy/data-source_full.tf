# Example: Alert Policy - Full

data "f5xc_alert_policy" "example" {
  name      = f5xc_alert_policy.example.name
  namespace = f5xc_alert_policy.example.namespace
}
