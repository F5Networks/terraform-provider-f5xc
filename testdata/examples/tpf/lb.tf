resource "f5xc_http_loadbalancer" "lb" {
  name = "ck-lb-test-01"
  # labels      = { "a" : "b" }
  namespace   = "default"
  annotations = { "annoA" : "annob" }
  description = "lb_test_01"
  domains     = ["example.com"]

  rate_limit_choice = {
    disable_rate_limit = true
  }

  loadbalancer_type_choice = {
    http = {
      port_choice = {
        port = 8080
      }
    }
  }

  /*
  bot_defense_choice = {
    //disable_bot_defense = true
    bot_defense = {
      policy = {
        protected_app_endpoints = [
          {
            goodbot_choice = {
              allow_good_bots = true
            }
            http_methods = ["METHOD_PUT"]
            path = {
              path_match_choice = {
                prefix = "/api"
              }
            }
            mitigation = {

            }
            flow_label_choice = {
              undefined_flow_label = true
            }
            metadata = {
              name = "meta1"
            }
            app_traffic_type_choice = {
              web = true
            }
          }
        ]
        javascript_mode = ["ASYNC_JS_NO_CACHING"]
        java_script_choice = {
          disable_js_insert = true
        }
        mobile_sdk_choice = {
          disable_mobile_sdk = true
        }
      }
      regional_endpoint = ["AUTO"]
      //regional_endpoint = ["NOT_A_ENUM_MEMBER"]
      #timeout = 1000000000
    }
  }
   */

  waf_choice = {
    #disable_waf = true
    app_firewall = {
      name      = "foundational-policy"
      namespace = "default"
    }
  }

  hash_policy_choice = {
    round_robin = true
    //ring_hash = true
  }

  api_testing_choice = {
    disable_api_testing = true
  }

  trust_client_ip_headers_choice = {
    disable_trust_client_ip_headers = true
  }

  threat_mesh_choice = {
    disable_threat_mesh = true
  }

  advertise_choice = {
    do_not_advertise = true
  }

  sensitive_data_policy_choice = {
    default_sensitive_data_policy = true
  }

  malicious_user_detection_choice = {
    disable_malicious_user_detection = true
  }

  api_definition_choice = {
    disable_api_definition = true
  }

  user_id_choice = {
    user_id_client_ip = true
  }

  challenge_type_choice = {
    //no_challenge = true
    js_challenge = {
      //js_script_delay = 5
    }
  }

  api_discovery_choice = {
    disable_api_discovery = true
  }

  service_policy_choice = {
    no_service_policies = true
  }

  malware_protection_choice = {
    disable_malware_protection = true
  }
}
