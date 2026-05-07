resource "f5xc_enhanced_firewall_policy" "example" {
  name      = "example_enhanced_firewall_policy"
  namespace = "example_namespace"
  rule_choice = {
    allow_all = true
  }
}
