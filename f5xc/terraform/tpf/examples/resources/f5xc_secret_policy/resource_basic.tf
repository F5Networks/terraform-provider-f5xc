# Example: Secret Policy - Basic

resource "f5xc_secret_policy" "example" {
  name        = "example-secret-policy"
  namespace   = "default"
  description = "Secret policy example"

  rule_choice = {
    rule_list = {
      rules = [
        {
          metadata = {
            name = "example-rule"
          }
          spec = {
            action = ["ALLOW"]
            client_choice = {
              client_name = "example-client"
            }
          }
        }
      ]
    }
  }
}