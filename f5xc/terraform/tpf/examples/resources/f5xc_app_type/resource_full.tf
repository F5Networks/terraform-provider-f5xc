# Example: App Type - Full

resource "f5xc_app_type" "example" {
  name        = "example-app-type"
  namespace   = "shared"
  description = "Full app type"

  labels = {
    env  = "example"
    tier = "api"
  }

  features = [
    {
      type = ["BUSINESS_LOGIC_MARKUP"]
    },
    {
      type = ["USER_BEHAVIOR_ANALYSIS"]
    }
  ]

  business_logic_markup_setting = {
    discovered_api_settings = {
      purge_duration_for_inactive_discovered_apis = 3
    }
    learn_from_redirect_traffic_choice = {
      enable = true
    }
  }
}
