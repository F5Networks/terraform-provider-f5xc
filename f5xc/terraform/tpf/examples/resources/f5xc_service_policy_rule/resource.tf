resource "f5xc_service_policy_rule" "example" {
  name      = "example_service_policy_rule"
  namespace = "example_namespace"
  waf_action = {
    action_type = {
      waf_skip_processing = true
    }
  }
  ip_choice = {
    ip_prefix_list = {
      ip_prefix_list = "example_ip_prefix_list"
    }
  }
  action = ["ENUM_VALUE"]
  client_choice = {
    client_name = "example_client_name"
  }
  asn_choice = {
    asn_list = {
      asn_list = "example_asn_list"
    }
  }
}
