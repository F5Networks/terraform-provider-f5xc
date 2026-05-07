resource "f5xc_forward_proxy_policy" "example" {
  name      = "example_forward_proxy_policy"
  namespace = "example_namespace"
  rule_choice = {
    allow_all = true
  }
  proxy_choice = {
    network_connector = {
      network_connector = "example_network_connector"
    }
  }
}
