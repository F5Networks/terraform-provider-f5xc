# Example: Service Policy - Full

resource "f5xc_service_policy" "example" {
  name      = "example-service-policy"
  namespace = "default"

  labels = {
    env  = "example"
    team = "platform"
  }

  annotations = {
    owner = "example-owner"
  }

  server_choice = {
    server_name = "svc.example.com"
  }

  rule_choice = {
    rule_list = {
      rules = [
        {
          metadata = {
            name        = "inline-example-rule"
            description = "inline rule"
          }

          spec = {
            action = ["ALLOW"]

            ip_choice = {
              any_ip = true
            }

            asn_choice = {
              any_asn = true
            }

            client_choice = {
              any_client = true
            }

            waf_action = {
              action_type_choice = {
                none = true
              }
            }
          }
        }
      ]
    }
  }
}
