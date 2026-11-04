# Example: Appstack Site - Network Config

resource "f5xc_appstack_site" "example" {
  name        = "example-appstack-site"
  namespace   = "system"
  description = "Updated with network config"

  sriov_interface_choice = {
    default_sriov_interface = true
  }

  bond_choice = {
    no_bond_devices = true
  }

  k8s_cluster_choice = {
    no_k8s_cluster = true
  }

  gpu_choice = {
    disable_gpu = true
  }

  storage_cfg_choice = {
    default_storage_config = true
  }

  usb_policy_choice = {
    allow_all_usb = true
  }

  blocked_services_choice = {
    default_blocked_services = true
  }

  logs_receiver_choice = {
    logs_streaming_disabled = true
  }

  network_cfg_choice = {
    custom_network_config = {
      slo_choice = {
        default_config = true
      }
      site_mesh_group_choice = {
        sm_connection_pvt_ip = true
      }
      interface_choice = {
        default_interface_config = true
      }
      network_policy_choice = {
        no_network_policy = true
      }
      forward_proxy_choice = {
        no_forward_proxy = true
      }
      global_network_choice = {
        no_global_network = true
      }
      vip_vrrp_mode = ["VIP_VRRP_DISABLE"]
      tunnel_dead_timeout = 0
    }
  }

  master_node_configuration = [
    {
      name = "master-node-1"
    }
  ]
}
