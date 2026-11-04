# Example: Alert Receiver - Full

data "f5xc_alert_receiver" "example" {
  name      = f5xc_alert_receiver.example.name
  namespace = f5xc_alert_receiver.example.namespace
}
