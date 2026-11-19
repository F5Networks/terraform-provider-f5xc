# Example: Network Interface - Full

resource "f5xc_network_interface" "example" {
  name        = "example-network-interface"
  namespace   = "system"
  description = "Full network interface example"

  labels = {
    env  = "example"
    team = "platform"
  }

  annotations = {
    owner = "example-owner"
  }

  interface_choice = {
    ethernet_interface = {
      device = "eth0"

      address_choice = {
        dhcp_client = true
      }
      network_choice = {
        site_local_network = true
      }
      node_choice = {
        cluster = true
      }

      primary_choice = {
        not_primary = true
      }
      vlan_choice = {
        untagged = true
      }
    }
  }
}
