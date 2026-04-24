terraform {
  required_providers {
    f5xc = {
      source  = "localhost/f5/f5xc"
      version = "0.1.1"
    }
  }
}

variable "f5xc_api_url" {
  type = string
}

variable "f5xc_api_p12_file" {
  type = string
}

variable "f5xc_api_token" {
  type = string
}

provider "f5xc" {
  url     = var.f5xc_api_url
  timeout = "30s"
  # api_token = var.f5xc_api_token
  api_p12_file = var.f5xc_api_p12_file
  limiter = {
    rate  = 30
    burst = 1
  }
}


resource "f5xc_healthcheck" "hc1" {
  name        = "test-hc-1"
  namespace   = "default"
  description = "healthcheck test1"
  health_check_choice = {
    http_health_check = {
      host_header_choice = {
        use_origin_server_name = true
      }
    }
  }
}

resource "f5xc_origin_pool" "op" {
  name                   = "test-op-123"
  namespace              = "default"
  description            = "origin pool test"
  loadbalancer_algorithm = ["ROUND_ROBIN"]
  tls_choice = {
    no_tls = true
  }
  port_choice = {
    port = 8080
  }
  endpoint_selection = ["LOCAL_PREFERRED"]
  origin_servers = [
    {
      choice = {
        private_ip = {
          private_ip_choice = {
            ip = "192.168.1.1"
          }
          site_locator = {
            choice = {
              site = {
                name      = "test-site-1"
                namespace = "default"
              }
            }
          }
          network_choice = {
            inside_network = true
          }
        }
      }
    }
  ]
  healthcheck = [
    {
      name      = f5xc_healthcheck.hc1.name
      namespace = f5xc_healthcheck.hc1.namespace
    }
  ]
}

resource "f5xc_alert_policy" "ap1" {
  name      = "test-alert-policy-1"
  namespace = "default"
  receivers = [
    {
      namespace = "default"
      name      = "ck-test-01"
    }
  ]
  routes = [
    {
      action_choice = {
        send = true
      }
    }
  ]
}


resource "f5xc_alert_receiver" "ar1" {
  name      = "ck-test-ar-1"
  namespace = "system"
  receiver_choice = {
    pagerduty = {
      routing_key = {
        secret_info_choice = {
          clear_secret_info = {
            // url = base64encode("string:///base64encodedstring125")
            url = "string://base64encodedstring126"
          }
        }
      }
      url = "https://events.pagerduty.com/v2/enqueue"
    }
  }
}


resource "f5xc_token" "t1" {
  name      = "ck-test-token-01"
  namespace = "system"
}

resource "f5xc_namespace" "ns1" {
  name = "ck-test-ns1"
}

resource "f5xc_virtual_k8s" "vk8s1" {
  name      = "test-vk8s-1"
  namespace = f5xc_namespace.ns1.name
}

resource "f5xc_virtual_network" "vn1" {
  name      = "test-vn-1"
  namespace = "system"
  network_choice = {
    // global_network = true
    legacy_type = ["VIRTUAL_NETWORK_SITE_LOCAL"] // ["A_VALUE_WHICH_IS_NOT_ENUM_MEMBER"]
  }
}

resource "f5xc_voltstack_site" "vss1" {
  name      = "test-site-1"
  namespace = "system"
  sriov_interface_choice = {
    default_sriov_interface = true
  }
  bond_choice = {
    no_bond_devices = true
  }
  k_8s_cluster_choice = {
    no_k8s_cluster = true
  }
  gpu_choice = {
    disable_gpu = true
  }
  storage_cfg_choice = {
    default_storage_config = true
  }
  usb_policy_choice = {
    deny_all_usb = true
  }
  blocked_services_choice = {
    default_blocked_services = true
  }
  logs_receiver_choice = {
    logs_streaming_disabled = true
  }
  network_cfg_choice = {
    default_network_config = true
  }
  master_node_configuration = [
    {
      name = "test-master-node-1"
    }
  ]
}

resource "f5xc_virtual_site" "vs1" {
  name      = "test-virtual-site-1"
  namespace = "system"
  site_selector = {
    expressions = ["example_virtual_site"]
  }
  site_type = ["REGIONAL_EDGE"]
}

resource "f5xc_enhanced_firewall_policy" "efp1" {
  name      = "ck-test-efp1"
  namespace = "system"
  rule_choice = {
    allow_all = true
  }
}

resource "f5xc_network_firewall" "nfw1" {
  name      = "ck-test-nfw-1"
  namespace = "system"
}

resource "f5xc_network_policy_rule" "nfwr1" {
  name      = "ck-test-nfwr1"
  namespace = "system"
}

resource "f5xc_service_policy" "sp1" {
  name      = "ck-sp-01"
  namespace = "default"
  rule_choice = {
    allow_all_requests = true
  }
  server_choice = {
    any_server = true
  }
}

resource "f5xc_forward_proxy_policy" "fpp1" {
  name      = "ck-fpp-01"
  namespace = "system"
  proxy_choice = {
    any_proxy = true
  }
  rule_choice = {
    deny_list = {
      default_action_choice = {
        default_action_deny = true
      }
      dest_list = [
        {
          prefixes = ["192.168.0.0/24"]
        }
      ]
    }
  }
}

resource "f5xc_v1_dns_monitor" "v1_dns_mon1" {
  name      = "ck-test-dns-mon-01"
  namespace = "system"
  domain    = "example.com"
  interval_choice = {
    interval_6_hours = true
  }
  on_failure_to_choice = {
    on_failure_to_all = true
  }
  external_sources = [
    {
      source_choice = {
        aws = {
          regions = ["us-east-2"]
        }
      }
    }
  ]
}

resource "f5xc_site_mesh_group" "smg1" {
  name      = "ck-smg-01"
  namespace = "system"
  mesh_choice = {
    full_mesh = {
      full_mesh_choice = {
        control_and_data_plane_mesh = true
      }
    }
  }
  re_fallback_choice = {
    enable_re_fallback = true
  }
  bfd_choice = {
    bfd_enabled = {}
  }
}

resource "f5xc_app_firewall" "app-fw1" {
  name = "ck-app-fw-1"
  namespace = "system"
  enforcement_mode_choice = {
    monitoring = true
  }
  detection_setting_choice = {
    default_detection_settings = true
  }
  allowed_response_codes_choice = {
    allow_all_response_codes = true
  }
  anonymization_setting_choice = {
    default_anonymization = true
  }
  blocking_page_choice = {
    use_default_blocking_page = true
  }
  enhance_with_ai_choice = {
    disable_ai_enhancements = true
  }
}

resource "f5xc_app_setting" "apps-1" {
  name      = "ck-app-setting-1"
  namespace = "default"
  app_type_settings = [{
    app_type_ref = [{
      name      = "ck-app-type-ref-1"
      namespace = "default"
    }]
  }]
}

resource "f5xc_nat_policy" "natp1" {
  name      = "ck-nat-policy-1"
  namespace = "system"
  applies_to_choice = {
    site = {
      refs = [
        {
          name      = "pg-llm-gw"
          namespace = "system"
        }
      ]
    }
  }
  rules = [
    {
      action = {
        source_nat_choice = {
          dynamic = {
            pool_choice = {
              pools = {
                prefixes = ["192.168.0.0/24"]
              }
            }
          }
        }
      }
      enable_choice = {
        enable = true
      }
      name = "ck-nat-policy-rule-1"
      scope_choice = {
        virtual_network = {
          virtual_network = f5xc_virtual_network.vn1.name
        }
      }
      criteria = {
        /*
        protocol_choice = {
          any = true
        }
         */
        source_cidr = ["192.168.0.0/24"]
      }
    }
  ]
}

resource "f5xc_api_definition_uploader" "swagger" {
  metadata = {
    name = "test126"
    namespace = "default"
  }
  string_value = file("/Users/c.klewar/Projects/gitlab.com/F5/volterra/ves.io/terraform-provider-f5xc/testdata/examples/tpf/openapi.json")
}
