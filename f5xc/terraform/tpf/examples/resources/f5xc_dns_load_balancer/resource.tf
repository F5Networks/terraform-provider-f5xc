resource "f5xc_dns_load_balancer" "example" {
  name        = "example_dns_load_balancer"
  namespace   = "example_namespace"
  record_type = ["ENUM_VALUE"]
  rule_list = {
    rules = [{
      action_choice = {
        pool = {
          pool = "example_pool"
        }
      }
      client_choice = {
        ip_prefix_list = {
          ip_prefix_list = "example_ip_prefix_list"
        }
      }
    }]
  }
}
