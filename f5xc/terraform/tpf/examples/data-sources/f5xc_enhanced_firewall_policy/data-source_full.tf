# Example: Enhanced Firewall Policy - Full

data "f5xc_enhanced_firewall_policy" "example" {
  name      = f5xc_enhanced_firewall_policy.example.name
  namespace = f5xc_enhanced_firewall_policy.example.namespace
}
