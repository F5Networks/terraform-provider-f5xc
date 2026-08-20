# Example: Origin Pool - Full

resource "f5xc_origin_pool" "example" {
  name                   = "example-origin-pool"
  namespace              = "system"
  description            = "Origin pool full example"
  loadbalancer_algorithm = ["LEAST_REQUEST"]
  endpoint_selection     = ["LOCAL_PREFERRED"]

  labels = {
    "env"     = "acceptance-test"
    "version" = "v2"
  }

  port_choice = {
    lb_port = true
  }

  health_check_port_choice = {
    same_as_endpoint_port = true
  }

  origin_servers = [
    {
      choice = {
        public_name = {
          dns_name         = "primary.example.com"
          refresh_interval = 300
        }
      }
    },
    {
      choice = {
        public_name = {
          dns_name = "secondary.example.com"
        }
      }
    }
  ]

  tls_choice = {
    use_tls = {
      sni_choice = {
        disable_sni = true
      }
      tls_config = {
        choice = {
          default_security = true
        }
      }
      server_validation_choice = {
        skip_server_verification = true
      }
      mtls_choice = {
        no_mtls = true
      }
      max_session_keys_type_choice = {
        default_session_key_caching = true
      }
    }
  }

  advanced_options = {
    max_requests_per_connection_choice = {
      max_requests_per_connection = 100
    }
    circuit_breaker_choice = {
      circuit_breaker = {
        retries          = 3
        max_requests     = 1024
        connection_limit = 1024
        pending_requests = 1024
        priority         = ["DEFAULT"]
      }
    }
    outlier_detection_choice = {
      outlier_detection = {
        base_ejection_time          = 30000
        max_ejection_percent        = 10
        consecutive_gateway_failure = 5
        consecutive_5xx             = 5
        interval                    = 10000
      }
    }
    subset_choice = {
      disable_subsets = true
    }
    http_protocol_type_choice = {
      http2_options = {
        enabled = true
      }
    }
    panic_threshold_type_choice = {
      no_panic_threshold = true
    }
    connection_timeout = 5000
  }

  upstream_conn_pool_reuse_type = {
    map_downstream_to_upstream_conn_pool_type_choice = {
      disable_conn_pool_reuse = true
    }
  }
}
