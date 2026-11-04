# Example: Gcp Vpc Site Set Cloud Site Info - Basic

resource "f5xc_gcp_vpc_site" "site" {
  name      = "example-gcp-vpc-cloud-info-fxbiwu"
  namespace = "system"

  gcp_region = "us-west1"

  site_type_choice = {
    ingress_gw = {
      gcp_certified_hw = "gcp-byol-voltmesh"
      gcp_zone_names   = ["us-west1-a"]
      node_number      = 1
      local_network = {
        choice = {
          new_network = {
            name = "example-gcp-vpc-cloud-info-fxbiwu-network"
          }
        }
      }
      local_subnet = {
        choice = {
          new_subnet = {
            subnet_name  = "example-gcp-vpc-cloud-info-fxbiwu-subnet"
            primary_ipv4 = "10.20.1.0/24"
          }
        }
      }
    }
  }

  deployment_choice = {
    cloud_credentials = {
      name      = "example-gcp-cred"
      namespace = "system"
    }
  }

  private_connectivity_choice = {
    private_connect_disabled = true
  }

  instance_type = "n1-standard-4"
  ssh_key       = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDPqY7k9N5aI9vE2T8W7m5L3z8x6Qp4jW2eN8cM3sYk6u9p0q2r4t6v8w0x2y4z6gcp"

  logs_receiver_choice = {
    logs_streaming_disabled = true
  }

  blocked_services_choice = {
    block_all_services = true
  }

  lifecycle {
    ignore_changes = all
  }
}


resource "f5xc_gcp_vpc_site_set_cloud_site_info" "example" {
  name      = f5xc_gcp_vpc_site.site.name
  namespace = "system"

  aws_vpc_info = {
    public_ips  = ["34.10.10.10"]
    private_ips = ["10.10.10.10"]
  }
}
