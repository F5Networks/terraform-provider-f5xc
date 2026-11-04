# Example: Service Policy Rule - Full

resource "f5xc_service_policy_rule" "example" {
  name        = "example-service-policy-rule"
  namespace   = "default"
  description = "service policy rule full"
  action      = ["ALLOW"]

  labels = {
    env  = "example"
    team = "platform"
  }

  annotations = {
    owner = "example-owner"
  }

  path = {
    prefix_values = ["/api"]
  }

  http_method = {
    methods = ["GET"]
  }

  ip_choice = {
    any_ip = true
  }

  asn_choice = {
    any_asn = true
  }

  client_choice = {
    client_name_matcher = {
      exact_values = ["example-client"]
    }
  }

  waf_action = {
    action_type_choice = {
      waf_skip_processing = true
    }
  }
}
