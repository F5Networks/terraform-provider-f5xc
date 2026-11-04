# Example: Fleet - Full

resource "f5xc_fleet" "example" {
  name        = "example-fleet"
  namespace   = "system"
  description = "Full fleet example"

  fleet_label = "example-fleet"

  labels = {
    env  = "example"
    tier = "full"
  }

  enable_default_fleet_config_download = true

  bond_choice = {
    no_bond_devices = true
  }

  dc_cluster_group_choice = {
    no_dc_cluster_group = true
  }

  gpu_choice = {
    disable_gpu = true
  }

  interface_choice = {
    default_config = true
  }

  logs_receiver_choice = {
    logs_streaming_disabled = true
  }

  sriov_interface_choice = {
    default_sriov_interface = true
  }

  storage_class_choice = {
    default_storage_class = true
  }

  storage_device_choice = {
    no_storage_device = true
  }

  storage_interface_choice = {
    no_storage_interfaces = true
  }

  storage_static_routes_choice = {
    no_storage_static_routes = true
  }

  usb_policy_choice = {
    deny_all_usb = true
  }

  vm_choice = {
    disable_vm = true
  }
}
