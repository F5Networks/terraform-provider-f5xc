# Example: Forward Proxy Policy - Basic

resource "f5xc_forward_proxy_policy" "example" {
  name      = "example-forward-proxy-policy"
  namespace = "system"

  proxy_choice = {
    any_proxy = true
  }

  rule_choice = {
    allow_all = true
  }
}
