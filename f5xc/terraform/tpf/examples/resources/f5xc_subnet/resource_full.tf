# Example: Subnet - Full

resource "f5xc_subnet" "example" {
  name      = "example-subnet"
  namespace = "system"

  labels = {
    env  = "example"
    team = "platform"
  }

  annotations = {
    owner = "example-owner"
  }

  connection_choice = {
    isolated_nw = true
  }

  site_subnet_params = [
    {
      site = {
        name      = "default-site"
        namespace = "system"
      }

      subnet_dhcp_server_params = {
        dhcp_networks = [
          {
            network_prefix_choice = {
              network_prefix = "10.25.230.0/24"
            }
          }
        ]
      }

      address_choice = {
        dhcp = true
      }
    }
  ]
}
