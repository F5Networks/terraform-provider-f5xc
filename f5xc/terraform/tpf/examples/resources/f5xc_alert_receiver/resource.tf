resource "f5xc_alert_receiver" "example" {
  name      = "example_alert_receiver"
  namespace = "example_namespace"
  receiver = {
    email = {
      email = "example_email"
    }
  }
}
