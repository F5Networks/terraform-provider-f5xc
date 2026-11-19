# Example: External Connector with a GRE tunnel

# Replace the CE site, node, interface, and peer address with values from your environment.
resource "f5xc_external_connector" "example" {
  name        = "example-external-connector"
  namespace   = "system"
  description = "GRE external connector for an existing CE site"

  labels = {
    env  = "example"
    team = "network-security"
  }

  annotations = {
    owner = "platform-team"
  }

  ce_site_reference = {
    name      = "example-ce-site"
    namespace = "system"
  }

  connection_type_choice = {
    gre = {
      gre_parameters = {
        peer_ip_address = {
          addr = "203.0.113.10"
        }

        tunnel_mtu = 1370
        tunnel_eps = [{
          node             = "example-ce-node"
          interface        = "example-ce-interface"
          local_tunnel_ip  = "169.254.3.1/30"
          remote_tunnel_ip = "169.254.3.2/30"
        }]

        tunnel_source_vn_choice = {
          site_local_inside_network = true
        }
      }
    }
  }
}