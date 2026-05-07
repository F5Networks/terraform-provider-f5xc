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
        asn_list = {
          asn_list = "example_asn_list"
        }
      }
    }]
  }
}
