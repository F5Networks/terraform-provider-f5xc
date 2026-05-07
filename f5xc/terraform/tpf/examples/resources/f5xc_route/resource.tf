resource "f5xc_route" "example" {
  name      = "example_route"
  namespace = "example_namespace"
  routes = [{
    route_action = {
      route_destination = {
        route_destination = "example_route_destination"
      }
    }
  }]
}
