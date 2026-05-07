resource "f5xc_voltstack_site" "example" {
  name      = "example_voltstack_site"
  namespace = "example_namespace"
  storage_cfg_choice = {
    default_storage_config = true
  }
  sriov_interface_choice = {
    default_sriov_interface = true
  }
  bond_choice = {
    no_bond_devices = true
  }
  usb_policy_choice = {
    usb_policy = {
      usb_policy = "example_usb_policy"
    }
  }
  network_cfg_choice = {
    default_network_config = true
  }
  k8s_cluster_choice = {
    k8s_cluster = {
      k8s_cluster = "example_k8s_cluster"
    }
  }
  logs_receiver_choice = {
    logs_streaming_disabled = true
  }
  blocked_services_choice = {
    default_blocked_services = true
  }
  master_node_configuration = [{
    name = "example_voltstack_site"
  }]
  gpu_choice = {
    enable_gpu = true
  }
}
