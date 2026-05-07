resource "f5xc_http_loadbalancer" "example" {
  name      = "example_http_loadbalancer"
  namespace = "example_namespace"
  user_id_choice = {
    user_id_client_ip = true
  }
  hash_policy_choice = {
    least_active = true
  }
  trust_client_ip_headers_choice = {
    disable_trust_client_ip_headers = true
  }
  malware_protection = {
    disable_malware_protection = true
  }
  loadbalancer_type = {
    http = {
      http = "example_http"
    }
  }
  threat_mesh_choice = {
    disable_threat_mesh = true
  }
  advertise_choice = {
    advertise_on_public = {
      advertise_on_public = "example_advertise_on_public"
    }
  }
  waf_choice = {
    app_firewall = {
      app_firewall = "example_app_firewall"
    }
  }
  rate_limit_choice = {
    disable_rate_limit = true
  }
  api_definition_choice = {
    disable_api_definition = true
  }
  challenge_type = {
    js_challenge = {
      js_challenge = "example_js_challenge"
    }
  }
  service_policy_choice = {
    active_service_policies = {
      active_service_policies = "example_active_service_policies"
    }
  }
  api_discovery_choice = {
    enable_api_discovery = {
      enable_api_discovery = "example_enable_api_discovery"
    }
  }
  api_testing_choice = {
    disable_api_testing = true
  }
  domains = ["example_http_loadbalancer"]
  malicious_user_detection_choice = {
    enable_malicious_user_detection = true
  }
  sensitive_data_policy_choice = {
    default_sensitive_data_policy = true
  }
}
