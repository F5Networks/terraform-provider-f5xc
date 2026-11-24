# Example: External Connector with an IPSec tunnel using F5 XC system-shared IKE profiles

# F5 XC predefined IKE Phase 1 and Phase 2 profiles reside in the "shared" namespace under
# tenant "ves-io". When referencing them from a non-shared namespace (such as "system"),
# you must explicitly set tenant = "ves-io".
resource "f5xc_external_connector" "ipsec_example" {
  name        = "example-ipsec-external-connector"
  namespace   = "system"
  description = "IPSec external connector for an existing CE site"

  ce_site_reference = {
    name      = "example-ce-site"
    namespace = "system"
  }

  connection_type_choice = {
    ipsec = {
      ike_parameters = {
        ike_phase1_profile = {
          name      = "ves-io-f5-ike-phase1-default-profile"
          namespace = "shared"
          tenant    = "ves-io"
        }

        ike_phase2_profile = {
          name      = "ves-io-f5-ike-phase2-default-profile"
          namespace = "shared"
          tenant    = "ves-io"
        }

        mode_choice = {
          initiator = true
        }

        dpd_choice = {
          dpd_keep_alive_timer = {
            timeout = 3
          }
        }

        local_ike_id_choice = {
          use_default_local_ike_id = true
        }

        remote_ike_id_choice = {
          use_default_remote_ike_id = true
        }
      }

      ipsec_tunnel_parameters = {
        peer_ip_address = {
          addr = "203.0.113.10"
        }

        psk        = "<pre-shared-key>"
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
