resource "f5xc_secret_policy_rule" "example" {
  name      = "example_secret_policy_rule"
  namespace = "example_namespace"
  action    = ["ENUM_VALUE"]
  client_choice = {
    client_name = "example_client_name"
  }
}
