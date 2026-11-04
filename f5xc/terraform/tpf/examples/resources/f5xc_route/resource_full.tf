# Example: Route - Full

resource "f5xc_route" "example" {
  name        = "example-route"
  namespace   = "default"
  description = "Full route example"

  labels = {
    env  = "example"
    team = "platform"
  }

  annotations = {
    owner = "example-owner"
  }

  routes = [
    {
      match = [
        {
          path = {
            path_match_choice = {
              path = "/app"
            }
          }
          headers = [
            {
              name = "x-env"
              value_match_choice = {
                exact = "example"
              }
            }
          ]
          http_method = ["GET"]
        }
      ]
      route_action_choice = {
        route_redirect = {
          response_code  = 302
          proto_redirect = "https"
          redirect_path_choice = {
            prefix_rewrite = "/v2"
          }
          query_params_choice = {
            remove_all_params = true
          }
        }
      }
    }
  ]
}
