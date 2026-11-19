# Example: Secret Policy Rule - Full

resource "f5xc_secret_policy_rule" "example" {
  name        = "example-sp-rule"
  namespace   = "system"
  description = "Full secret policy rule example"

  labels = {
    env  = "example"
    team = "platform"
  }

  annotations = {
    owner = "example-owner"
  }

  action = ["ALLOW"]

  client_choice = {
    client_name_matcher = {
      exact_values = ["example-client"]
      regex_values = ["^client-.*"]
    }
  }
}
