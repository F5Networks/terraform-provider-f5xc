# Example: Waf Exclusion Policy - Full

resource "f5xc_waf_exclusion_policy" "example" {
  name        = "example-waf-exclusion-policy"
  namespace   = "default"
  description = "Waf exclusion policy full example"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform"
  }

  waf_exclusion_rules = [{
    domain_choice = {
      exact_value = "shop.example.com"
    }
    path_choice = {
      path_prefix = "/checkout"
    }
    methods              = ["POST"]
    expiration_timestamp = "2035-01-01T00:00:00Z"
    waf_advanced_configuration_choice = {
      app_firewall_detection_control = {
        exclude_violation_contexts = [{
          exclude_violation = ["VIOL_METHOD"]
          context           = ["CONTEXT_URL"]
          context_name      = "hdr_xff"
        }]
      }
    }
    metadata = {
      name        = "example-rule"
      description = "Waf exclusion rule example"
    }
  }]
}
