resource "f5xc_forward_proxy_policy" "example" {
  name      = "example_forward_proxy_policy"
  namespace = "example_namespace"
  proxy_choice = {
    drp_http_connect = true
  }
  rule_choice = {
    deny_list = {
      deny_list = "example_deny_list"
    }
  }
}
