# Example: App API Group - Full

resource "f5xc_app_api_group" "example" {
  name        = "example-app-api-group"
  namespace   = "shared"
  description = "App API group example"

  labels = {
    env  = "example"
    tier = "api"
  }

  elements = [
    {
      methods    = ["GET", "POST"]
      path_regex = "^/api/v1/orders$"
    },
    {
      methods    = ["PUT", "DELETE"]
      path_regex = "^/api/v1/orders/.+$"
    }
  ]

  scope_choice = {
    http_loadbalancer = {
      http_loadbalancer = {
        name      = "app-http-lb"
        namespace = "shared"
      }
    }
  }
}
