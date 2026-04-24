resource "f5xc_securemesh_site_v2" "smv2" {
  name      = "ck-smv2-test-01"
  namespace = "system"
  labels = {"keyA": "valueA"}
  offline_survivability_mode = {
    offline_survivability_mode_choice = {
      no_offline_survivability_mode = true
    }
  }
  local_vrf = {
    slo_choice = {
      default_config = true
    }
  }
  logs_receiver_choice = {
    logs_streaming_disabled = true
  }
  performance_enhancement_mode = {
    perf_mode_choice = {
      perf_mode_l7_enhanced = {
        perf_mode_choice = {
          jumbo_enabled = true
        }
      }
    }
  }
  provider_choice = {
    azure = {
      orchestration_choice = {
        not_managed = {}
      }
    }
  }
  blocked_services_choice = {
    block_all_services = true
  }
  upgrade_settings = {
    kubernetes_upgrade_drain = {
      kubernetes_upgrade_drain_enable_choice = {
        disable_upgrade_drain = true
      }
    }
  }
  re_select = {
    re_selection_choice = {
      specific_re = {
        primary_re = "ves-io-paris"
        backup_re  = "ves-io-sanjose"
      }
    }
  }
  admin_user_credentials = {
    admin_password = {
      secret_info_choice = {
        clear_secret_info = {
          url = format("string:///%s", base64encode("passwordAbc123"))
        }
      }
    }
  }
  software_settings = {
    sw = {
      volterra_sw_version_choice = {
        default_sw_version = true
      }
    }
    os = {
      operating_system_version_choice = {
        default_os_version = true
      }
    }
  }
}