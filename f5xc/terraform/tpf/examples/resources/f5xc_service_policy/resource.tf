resource "f5xc_service_policy" "example" {
  name      = "example_service_policy"
  namespace = "example_namespace"
  rule_choice = {
    rule_list = {
      rule_list = "example_rule_list"
    }
  }
  server_choice = {
    any_server = true
  }
}
