# Example: Securemesh Site V2 - Basic

resource "f5xc_securemesh_site_v2" "example" {
  name      = "example-securemesh-site-v2"
  namespace = "system"

  logs_receiver_choice = {
    logs_streaming_disabled = true
  }

  blocked_services_choice = {
    block_all_services = true
  }

  provider_choice = {
    kvm = {
      orchestration_choice = {
        not_managed = {}
      }
    }
  }
}
