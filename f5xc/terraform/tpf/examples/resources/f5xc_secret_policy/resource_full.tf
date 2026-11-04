# Example: Secret Policy - Full

resource "f5xc_secret_policy" "example" {
  name                  = "example-secret-policy"
  namespace             = "default"
  description           = "Secret policy full example"
  allow_f5xc            = true
  decrypt_cache_timeout = "2h0m0s"

  labels = {
    env  = "example"
    team = "platform"
  }

  annotations = {
    owner = "platform"
  }

  rule_choice = {
    rule_list = {
      rules = [
        {
          metadata = {
            name        = "example-rule"
            description = "Secret policy rule example"
          }
          spec = {
            action = ["ALLOW"]
            client_choice = {
              client_name_matcher = {
                regex_values = ["^example-client-.*"]
                transformers = ["LOWER_CASE"]
              }
            }
          }
        }
      ]
    }
  }
}