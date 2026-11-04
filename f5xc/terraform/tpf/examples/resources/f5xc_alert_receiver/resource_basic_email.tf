# Example: Alert Receiver - Basic Email

resource "f5xc_alert_receiver" "example" {
  name        = "example-alert-receiver"
  namespace   = "default"
  description = "Alert receiver example"

  receiver_choice = {
    email = {
      email = "alerts@example.com"
    }
  }
}
