# Example: Infraprotect Firewall Rule Group - Full

data "f5xc_infraprotect_firewall_rule_group" "example" {
  name      = f5xc_infraprotect_firewall_rule_group.example.name
  namespace = f5xc_infraprotect_firewall_rule_group.example.namespace
}
