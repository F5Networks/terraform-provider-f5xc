# Example: Active Alert Policies - Policy

resource "f5xc_alert_receiver" "example" {
  name      = "example-alert-receiver"
  namespace = "default"

  receiver_choice = {
    email = {
      email = "alerts@example.com"
    }
  }
}

resource "f5xc_alert_policy" "policy_one" {
  name        = "example-alert-policy"
  namespace   = "default"
  description = "Active alert policy example"

  receivers = [
    {
      name      = f5xc_alert_receiver.example.name
      namespace = f5xc_alert_receiver.example.namespace
    }
  ]

  routes = [
    {
      matcher_choice = {
        any = true
      }
      action_choice = {
        send = true
      }
    }
  ]

  notification_parameters = {
    group_by_choice = {
      individual = true
    }
    group_wait      = "30s"
    group_interval  = "1m"
    repeat_interval = "4h"
  }
}


resource "f5xc_active_alert_policies" "example" {
  namespace = "default"

  alert_policies = [
    {
      name      = f5xc_alert_policy.policy_one.name
      namespace = f5xc_alert_policy.policy_one.namespace
    }
  ]
}
