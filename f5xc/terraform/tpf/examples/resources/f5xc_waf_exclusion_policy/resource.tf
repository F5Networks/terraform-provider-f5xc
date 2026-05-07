resource "f5xc_waf_exclusion_policy" "example" {
  name      = "example_waf_exclusion_policy"
  namespace = "example_namespace"
  waf_exclusion_rules = [{
    domain_choice = {
      any_domain = true
    }
    metadata = {
      name = "example_waf_exclusion_policy"
    }
    path_choice = {
      any_path = true
    }
  }]
}
