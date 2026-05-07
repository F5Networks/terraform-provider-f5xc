resource "f5xc_securemesh_site_v2" "example" {
  name      = "example_securemesh_site_v2"
  namespace = "example_namespace"
  blocked_services_choice = {
    blocked_services = {
      blocked_services = "example_blocked_services"
    }
  }
  provider_choice = {
    azure = {
      azure = "example_azure"
    }
  }
  logs_receiver_choice = {
    logs_streaming_disabled = true
  }
}
