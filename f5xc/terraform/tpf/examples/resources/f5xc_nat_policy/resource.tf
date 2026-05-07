resource "f5xc_nat_policy" "example" {
  name      = "example_nat_policy"
  namespace = "example_namespace"
  applies_to_choice = {
    site = {
      site = "example_site"
    }
  }
  rules = [{
    action = {}
    enable_choice = {
      enable = true
    }
    name = "example_nat_policy"
    scope_choice = {
      cloud_connect = {
        cloud_connect = "example_cloud_connect"
      }
    }
  }]
}
