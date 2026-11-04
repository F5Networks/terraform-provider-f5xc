# Example: App Type - Redirect Learning Transition

resource "f5xc_app_type" "example" {
  name      = "example-app-type"
  namespace = "shared"

  business_logic_markup_setting = {
    learn_from_redirect_traffic_choice = {
      enable = true
    }
  }
}
