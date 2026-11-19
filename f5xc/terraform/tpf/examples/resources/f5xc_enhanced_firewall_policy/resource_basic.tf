# Example: Enhanced Firewall Policy - Basic

resource "f5xc_enhanced_firewall_policy" "example" {
  name      = "example-enhanced-firewall-policy"
  namespace = "system"

  rule_choice = {
    deny_all = true
  }
}
