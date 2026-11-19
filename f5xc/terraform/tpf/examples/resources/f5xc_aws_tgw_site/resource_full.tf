# Example: Aws Tgw Site - Full

resource "f5xc_aws_tgw_site" "example" {
  name      = "example-aws-tgw-site-bif8ao-528000"
  namespace = "system"

  labels = {
    env   = "dev"
    owner = "platform"
  }

  tags = {
    environment = "dev"
    owner       = "platform"
  }

  aws_parameters = {
    aws_region = "us-west-2"
    az_nodes = [{
      aws_az_name = "us-west-2a"
      choice = {
        reserved_inside_subnet = true
      }
      outside_subnet = {
        choice = {
          subnet_param = {
            ipv4 = "10.10.1.0/24"
          }
        }
      }
    }]
    deployment_choice = {
      aws_cred = {
        name      = "volterra-dev-test"
        namespace = "system"
      }
    }
    instance_type = "t3.xlarge"
    internet_vip_choice = {
      enable_internet_vip = true
    }
    security_group_choice = {
      f5xc_security_group = true
    }
    service_vpc_choice = {
      new_vpc = {
        name_choice = {
          autogenerate = true
        }
        primary_ipv4 = "10.10.0.0/16"
      }
    }
    ssh_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3b9ZcE4fVYI7xQ1jQk7n4k3bM1XlI3oVf9v2yW1sYvWm2n3z4k5w6x7y8z9acc"
    tgw_choice = {
      new_tgw = {
        asn_choice = {
          system_generated = true
        }
      }
    }
    tgw_cidr_choice = {
      reserved_tgw_cidr = true
    }
    worker_nodes_choice = {
      no_worker_nodes = true
    }
  }

  blocked_services_choice = {
    block_all_services = true
  }

  direct_connect_choice = {
    direct_connect_disabled = true
  }

  logs_receiver_choice = {
    logs_streaming_disabled = true
  }
}
