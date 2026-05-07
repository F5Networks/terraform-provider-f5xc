resource "f5xc_route" "example" {
  name      = "example_route"
  namespace = "example_namespace"
  routes = [{
    route_action = {
      route_direct_response = {
        route_direct_response = "example_route_direct_response"
      }
    }
  }]
}
