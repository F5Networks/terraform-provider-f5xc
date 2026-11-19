# Example: Appstack Site - Full

resource "f5xc_appstack_site" "example" {
  name        = "example-appstack-site"
  namespace   = "system"
  description = "Full appstack site configuration"

  labels = {
    "env"     = "example"
    "version" = "v1"
  }

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
    enable_gpu = true
  }

  storage_cfg_choice = {
    default_storage_config = true
  }

  usb_policy_choice = {
    allow_all_usb = true
  }

  blocked_services_choice = {
    blocked_services = {
      blocked_services_list = ["ssh"]
    }
  }

  logs_receiver_choice = {
    logs_streaming_disabled = true
  }

  network_cfg_choice = {
    default_network_config = true
  }

  master_node_configuration = [
    {
      name = "master-node-1"
    }
  ]
}
