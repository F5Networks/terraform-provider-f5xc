resource "f5xc_alert_receiver" "example" {
  name      = "example_alert_receiver"
  namespace = "example_namespace"
  receiver = {
    opsgenie = {
      opsgenie = "example_opsgenie"
    }
  }
}
