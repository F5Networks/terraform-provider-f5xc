# Example: Network Firewall - Enhanced Firewall Dependency

resource "f5xc_enhanced_firewall_policy" "dep" {
  name      = "example-network-firewall-efp"
  namespace = "system"

  rule_choice = {
    allow_all = true
  }
}

resource "f5xc_network_firewall" "example" {
  name      = "example-network-firewall"
  namespace = "system"

  labels = {
    env  = "example"
    team = "platform"
  }

  annotations = {
    owner      = "example-owner"
    dependency = f5xc_enhanced_firewall_policy.dep.name
  }

  depends_on = [f5xc_enhanced_firewall_policy.dep]
}
