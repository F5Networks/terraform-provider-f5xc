# Example: App Type - Feature Type Transition

resource "f5xc_app_type" "example" {
  name      = "example-app-type"
  namespace = "shared"

  features = [
    {
      type = ["USER_BEHAVIOR_ANALYSIS"]
    }
  ]
}
