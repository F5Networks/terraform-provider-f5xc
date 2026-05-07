resource "f5xc_origin_pool" "example" {
  name      = "example_origin_pool"
  namespace = "example_namespace"
  origin_servers = [{
    choice = {
      k8s_service = {
        k8s_service = "example_k8s_service"
      }
    }
  }]
  tls_choice = {
    no_tls = true
  }
  port_choice = {
    automatic_port = true
  }
  endpoint_selection     = ["ENUM_VALUE"]
  loadbalancer_algorithm = ["ENUM_VALUE"]
}
