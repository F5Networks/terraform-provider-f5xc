# Example: Network Connector - Full

resource "f5xc_network_connector" "example" {
  name        = "example-netconn"
  namespace   = "system"
  description = "Full network connector example"

  connector_choice = {
    sli_to_slo_snat = {
      pool_choice = {
        interface_ip = true
      }
      routing_choice = {
        default_gw_snat = true
      }
    }
  }

  forward_proxy_choice = {
    disable_forward_proxy = true
  }

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform-team"
  }
}
