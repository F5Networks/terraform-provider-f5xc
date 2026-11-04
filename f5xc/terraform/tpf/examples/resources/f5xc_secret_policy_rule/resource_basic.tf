# Example: Secret Policy Rule - Basic

resource "f5xc_secret_policy_rule" "example" {
  name      = "example-sp-rule"
  namespace = "system"

  action = ["ALLOW"]

  client_choice = {
    client_name = "example-client"
  }
}
