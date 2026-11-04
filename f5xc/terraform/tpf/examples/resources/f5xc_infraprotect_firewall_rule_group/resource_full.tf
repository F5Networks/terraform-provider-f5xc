# Example: Infraprotect Firewall Rule Group - Full

resource "f5xc_infraprotect_firewall_rule_group" "example" {
  name                     = "example-infraprotect-firewall-rule-group"
  namespace                = "system"
  description              = "Firewall rule group for Infraprotect DDoS transit protection"
  firewall_rule_group_name = "example-firewall-rule-group"

  labels = {
    env  = "example"
    team = "network-security"
  }

  annotations = {
    owner = "platform-team"
  }
}
