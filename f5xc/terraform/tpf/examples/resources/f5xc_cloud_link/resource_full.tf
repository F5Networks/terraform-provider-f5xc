# Example: Cloud Link - Full

resource "f5xc_cloud_link" "example" {
  name        = "example-cloud-link"
  namespace   = "system"
  description = "cloud link full configuration"

  labels = {
    env  = "example"
    team = "network"
  }

  annotations = {
    owner = "platform"
  }

  cloud_provider_choice = {
    aws = {
      aws_cred = {
        name      = "orch-test-user-site-views-aws-vpc-dc-new"
        namespace = "system"
      }

      cloud_link_type_choice = {
        byoc = {
          connections = [{
            auth_key = {
              secret_info_choice = {
                clear_secret_info = {
                  url = "string:///RlVMTF9TRUNSRVQ="
                }
              }
            }
            bgp_asn       = 35280
            connection_id = "dxcon-fgjqgra5"
            ip_type_choice = {
              ipv4 = {
                aws_router_peer_address = "84.54.63.4/31"
                router_peer_address     = "84.54.63.5/31"
              }
            }
            metadata = {
              name        = "full-conn"
              description = "full connection"
            }
            region = "ap-southeast-1"
            tags = {
              purpose = "acceptance"
            }
            virtual_interface_type = ["PRIVATE"]
            vlan                   = 354
          }]
        }
      }

      direct_connect_gateway_asn_choice = {
        custom_asn = 4293918901
      }
    }
  }

  enable_connection_to_re_choice = {
    disabled = true
  }

}
