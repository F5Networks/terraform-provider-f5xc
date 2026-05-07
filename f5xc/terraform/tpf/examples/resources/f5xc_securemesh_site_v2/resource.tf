resource "f5xc_securemesh_site_v2" "example" {
  name      = "example_securemesh_site_v2"
  namespace = "example_namespace"
  logs_receiver_choice = {
    log_receiver_with_net = {
      log_receiver_with_net = "example_log_receiver_with_net"
    }
  }
  blocked_services_choice = {
    block_all_services = true
  }
  provider_choice = {
    gcp = {
      gcp = "example_gcp"
    }
  }
}
