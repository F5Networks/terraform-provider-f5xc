# Example: Securemesh Site V2 - Full

resource "f5xc_securemesh_site_v2" "example" {
  name        = "example-securemesh-site-v2"
  namespace   = "system"
  description = "securemesh full example"
  tunnel_type = ["SITE_TO_SITE_TUNNEL_IPSEC"]

  labels = {
    env  = "example"
    team = "platform"
  }

  annotations = {
    owner = "example-owner"
  }

  network_policy_choice = {
    no_network_policy = true
  }

  forward_proxy_choice = {
    no_forward_proxy = true
  }

  proxy_bypass_choice = {
    no_proxy_bypass = true
  }

  log_anonymization_choice = {
    disable_log_anonymization = true
  }

  url_categorization_choice = {
    enable_url_categorization = true
  }

  offline_survivability_mode = {
    offline_survivability_mode_choice = {
      enable_offline_survivability_mode = true
    }
  }

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
