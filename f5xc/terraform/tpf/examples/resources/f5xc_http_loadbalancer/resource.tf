resource "f5xc_http_loadbalancer" "example" {
  name      = "example_http_loadbalancer"
  namespace = "example_namespace"
  waf_choice = {
    disable_waf = true
  }
  user_id_choice = {
    user_id_client_ip = true
  }
  trust_client_ip_headers_choice = {
    disable_trust_client_ip_headers = true
  }
  domains = ["example_http_loadbalancer"]
  advertise_choice = {
    advertise_on_public = {
      advertise_on_public = "example_advertise_on_public"
    }
  }
  api_definition_choice = {
    disable_api_definition = true
  }
  malware_protection = {
    disable_malware_protection = true
  }
  loadbalancer_type = {
    http = {
      http = "example_http"
    }
  }
  rate_limit_choice = {
    api_rate_limit = {
      api_rate_limit = "example_api_rate_limit"
    }
  }
  api_testing_choice = {
    disable_api_testing = true
  }
  challenge_type = {
    policy_based_challenge = {
      policy_based_challenge = "example_policy_based_challenge"
    }
  }
  hash_policy_choice = {
    ring_hash = {
      ring_hash = "example_ring_hash"
    }
  }
  threat_mesh_choice = {
    disable_threat_mesh = true
  }
  malicious_user_detection_choice = {
    enable_malicious_user_detection = true
  }
  sensitive_data_policy_choice = {
    default_sensitive_data_policy = true
  }
  service_policy_choice = {
    service_policies_from_namespace = true
  }
  api_discovery_choice = {
    enable_api_discovery = {
      enable_api_discovery = "example_enable_api_discovery"
    }
  }
}
