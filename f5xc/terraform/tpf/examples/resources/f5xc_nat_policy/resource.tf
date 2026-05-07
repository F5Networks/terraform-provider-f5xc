resource "f5xc_nat_policy" "example" {
  name      = "example_nat_policy"
  namespace = "example_namespace"
  rules = [{
    action = {}
    enable_choice = {
      enable = true
    }
    name = "example_nat_policy"
    scope_choice = {
      node_interface = {
        node_interface = "example_node_interface"
      }
    }
  }]
  applies_to_choice = {
    site = {
      site = "example_site"
    }
  }
}
