# Example: Api Discovery - Full

resource "f5xc_api_discovery" "example" {
  name        = "example-api-discovery"
  namespace   = "default"
  description = "API discovery example"

  labels = {
    env  = "example"
    team = "appsec"
  }

  annotations = {
    owner = "platform"
  }

  custom_auth_types = [
    {
      parameter_type = ["HEADER"]
      parameter_name = "x-auth-token"
    }
  ]

  user_defined_api_discovery_policy = {
    discovery_rules = [
      {
        metadata = {
          name        = "exclude-healthz"
          description = "Ignore health-check paths"
        }
        rule_properties = {
          rule_type_choice = {
            exclusion = {
              action_choice = {
                ignore = true
              }
            }
          }
          criteria_choice = {
            pattern = "/healthz"
          }
        }
      },
      {
        metadata = {
          name = "include-api-v2"
        }
        rule_properties = {
          rule_type_choice = {
            inclusion = true
          }
          criteria_choice = {
            http_header_criteria = {
              field_name = "content-type"
              match_type = ["EXACT_MATCH"]
              value      = "application/json"
              location   = ["REQUEST"]
            }
          }
        }
      }
    ]
    default_behavior_choice = {
      exclusive = {
        action_choice = {
          archive = true
        }
      }
    }
  }
}
