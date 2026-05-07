resource "f5xc_service_policy" "example" {
  name      = "example_service_policy"
  namespace = "example_namespace"
  rule_choice = {
    deny_all_requests = true
  }
  server_choice = {
    server_name = "example_server_name"
  }
}
