resource "f5xc_alert_policy" "example" {
  name      = "example_alert_policy"
  namespace = "example_namespace"
  receivers = [{
    name      = "example_alert_policy"
    namespace = "example_namespace"
  }]
  routes = [{
    action = {
      send = true
    }
  }]
}
