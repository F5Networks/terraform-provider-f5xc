# Example: Api Discovery - Basic

resource "f5xc_api_discovery" "example" {
  name      = "example-api-discovery"
  namespace = "default"

  user_defined_api_discovery_policy = {
    default_behavior_choice = {
      inclusive = true
    }
  }
}
