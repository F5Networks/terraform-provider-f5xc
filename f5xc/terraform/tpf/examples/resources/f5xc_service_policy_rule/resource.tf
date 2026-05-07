resource "f5xc_service_policy_rule" "example" {
  name      = "example_service_policy_rule"
  namespace = "example_namespace"
  asn_choice = {
    any_asn = true
  }
  action = ["ENUM_VALUE"]
  waf_action = {
    action_type = {
      none = true
    }
  }
  client_choice = {
    any_client = true
  }
  ip_choice = {
    any_ip = true
  }
}
