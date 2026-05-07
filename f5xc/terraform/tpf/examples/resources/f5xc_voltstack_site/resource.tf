resource "f5xc_voltstack_site" "example" {
  name      = "example_voltstack_site"
  namespace = "example_namespace"
  sriov_interface_choice = {
    default_sriov_interface = true
  }
  gpu_choice = {
    disable_gpu = true
  }
  usb_policy_choice = {
    deny_all_usb = true
  }
  blocked_services_choice = {
    default_blocked_services = true
  }
  master_node_configuration = [{
    name = "example_voltstack_site"
  }]
  network_cfg_choice = {
    default_network_config = true
  }
  bond_choice = {
    no_bond_devices = true
  }
  storage_cfg_choice = {
    custom_storage_config = {
      custom_storage_config = "example_custom_storage_config"
    }
  }
  k8s_cluster_choice = {
    k8s_cluster = {
      k8s_cluster = "example_k8s_cluster"
    }
  }
  logs_receiver_choice = {
    logs_streaming_disabled = true
  }
}
