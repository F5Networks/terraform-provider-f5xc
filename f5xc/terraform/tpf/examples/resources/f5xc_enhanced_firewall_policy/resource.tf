resource "f5xc_enhanced_firewall_policy" "example" {
  name      = "example_enhanced_firewall_policy"
  namespace = "example_namespace"
  rule_choice = {
    allowed_destinations = {
      allowed_destinations = "example_allowed_destinations"
    }
  }
}
