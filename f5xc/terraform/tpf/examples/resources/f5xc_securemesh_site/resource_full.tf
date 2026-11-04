# Example: Securemesh Site - Full

resource "f5xc_securemesh_site" "example" {
  name      = "example-sms"
  namespace = "system"

  labels = {
    env   = "dev"
    owner = "platform"
  }

  annotations = {
    purpose = "acceptance"
  }

  address = "123 Main Street, Seattle, US"

  volterra_certified_hw = "isv-8000-series-voltmesh"

  waf_signatures = {
    signatures_update_mode_choice = {
      manual = true
    }
  }

  master_node_configuration = [
    {
      name      = "master-node-1"
      public_ip = "192.0.2.10"
    },
    {
      name      = "master-node-2"
      public_ip = "192.0.2.11"
    },
    {
      name      = "master-node-3"
      public_ip = "192.0.2.12"
    }
  ]

  worker_nodes = ["worker-node-1"]

  bond_choice = {
    no_bond_devices = true
  }

  network_cfg_choice = {
    default_network_config = true
  }

  logs_receiver_choice = {
    logs_streaming_disabled = true
  }

  blocked_services_choice = {
    default_blocked_services = true
  }

  offline_survivability_mode = {
    offline_survivability_mode_choice = {
      enable_offline_survivability_mode = true
    }
  }

  performance_enhancement_mode = {
    perf_mode_choice = {
      perf_mode_l3_enhanced = {
        perf_mode_choice = {
          jumbo = true
        }
      }
    }
  }

  kubernetes_upgrade_drain = {
    kubernetes_upgrade_drain_enable_choice = {
      disable_upgrade_drain = true
    }
  }

  timeouts = {
    create = "30m"
    read   = "5m"
    update = "30m"
    delete = "30m"
  }
}
