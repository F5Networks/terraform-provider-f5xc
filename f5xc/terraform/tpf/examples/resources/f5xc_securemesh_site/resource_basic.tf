# Example: Securemesh Site - Basic

resource "f5xc_securemesh_site" "example" {
  name      = "example-sms"
  namespace = "system"

  volterra_certified_hw = "isv-8000-series-voltmesh"

  master_node_configuration = [
    {
      name = "master-node-1"
    }
  ]

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
}
