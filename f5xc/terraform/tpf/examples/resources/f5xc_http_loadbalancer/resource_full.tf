# Example: Http Loadbalancer - Full

resource "f5xc_origin_pool" "example" {
  name                   = "example-http-op-n602nzk4"
  namespace              = "default"
  description            = "Acceptance test origin pool for HTTP load balancer"
  loadbalancer_algorithm = ["ROUND_ROBIN"]
  endpoint_selection     = ["LOCAL_PREFERRED"]

  port_choice = {
    port = 8080
  }

  origin_servers = [
    {
      choice = {
        public_name = {
          dns_name = "example.com"
        }
      }
    }
  ]

  tls_choice = {
    no_tls = true
  }

  upstream_conn_pool_reuse_type = {
    map_downstream_to_upstream_conn_pool_type_choice = {
      disable_conn_pool_reuse = true
    }
  }
}

resource "f5xc_http_loadbalancer" "example" {
  name      = "example-http-lb-n602nzk4"
  namespace = "default"

  description = "Acceptance test HTTP load balancer updated"
  labels = { env = "staging", tier = "edge" }

  domains = ["example-http-lb-n602nzk4.example.com"]

  loadbalancer_type_choice = {
    http = {
      port_choice = {
        port = 8880
      }
    }
  }

  advertise_choice = {
    advertise_on_public_default_vip = true
  }

  default_route_pools = [
    {
      pool_choice = {
        pool = {
          name      = f5xc_origin_pool.example.name
          namespace = f5xc_origin_pool.example.namespace
        }
      }
      weight   = 1
      priority = 1
    }
  ]

  waf_choice = {
    disable_waf = true
  }

  challenge_type_choice = {
    js_challenge = {
      js_script_delay = 2000
    }
  }

  user_id_choice = {
    user_id_client_ip = true
  }

  rate_limit_choice = {
    disable_rate_limit = true
  }

  service_policy_choice = {
    service_policies_from_namespace = true
  }

  hash_policy_choice = {
    least_active = true
  }

  trust_client_ip_headers_choice = {
    disable_trust_client_ip_headers = true
  }

  malicious_user_detection_choice = {
    disable_malicious_user_detection = true
  }

  api_discovery_choice = {
    disable_api_discovery = true
  }

  bot_defense_choice = {
    disable_bot_defense = true
  }

  api_definition_choice = {
    disable_api_definition = true
  }

  sensitive_data_policy_choice = {
    default_sensitive_data_policy = true
  }

  api_testing_choice = {
    disable_api_testing = true
  }

  threat_mesh_choice = {
    disable_threat_mesh = true
  }

  malware_protection_choice = {
    disable_malware_protection = true
  }

  
  
  
}
