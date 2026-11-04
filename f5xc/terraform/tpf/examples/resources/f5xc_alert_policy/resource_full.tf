# Example: Alert Policy - Full

resource "f5xc_alert_receiver" "example" {
  name      = "example-alert-receiver"
  namespace = "default"

  receiver_choice = {
    email = {
      email = "alerts@example.com"
    }
  }
}

resource "f5xc_alert_policy" "example" {
  name        = "example-alert-policy"
  namespace   = "default"
  description = "Alert policy full example"

  labels = {
    env = "example"
  }

  receivers = [
    {
      name      = f5xc_alert_receiver.example.name
      namespace = f5xc_alert_receiver.example.namespace
    }
  ]

  routes = [
    {
      matcher_choice = {
        alertname_regex = "^Service.*"
      }
      action_choice = {
        dont_send = true
      }
      notification_parameters = {
        group_by_choice = {
          custom = {
            labels = ["alertname"]
          }
        }
        group_wait      = "45s"
        group_interval  = "2m"
        repeat_interval = "2h"
      }
    }
  ]

  notification_parameters = {
    group_by_choice = {
      custom = {
        labels = ["severity"]
      }
    }
    group_wait      = "30s"
    group_interval  = "1m"
    repeat_interval = "4h"
  }
}
