# Example: Aws Vpc Site - Full

resource "f5xc_aws_vpc_site" "example" {
  name      = "example-aws-vpc-site-97fdyl"
  namespace = "system"

  labels = {
    env   = "dev"
    owner = "platform"
  }

  tags = {
    environment = "dev"
    owner       = "platform"
  }

  address = "123 Main Street, Seattle, US"

  vpc = {
    choice = {
      new_vpc = {
        name_choice = {
          autogenerate = true
        }
        primary_ipv4 = "10.10.0.0/16"
      }
    }
  }

  site_type_choice = {
    ingress_gw = {
      az_nodes = [{
        aws_az_name = "us-west-2a"
        local_subnet = {
          choice = {
            subnet_param = {
              ipv4 = "10.10.1.0/24"
            }
          }
        }
      }]
      aws_certified_hw = "aws-byol-voltmesh"
    }
  }

  deployment_choice = {
    aws_cred = {
      name      = "volterra-dev-test"
      namespace = "system"
    }
  }

  instance_type = "t3.xlarge"
  aws_region    = "us-west-2"
  ssh_key       = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3b9ZcE4fVYI7xQ1jQk7n4k3bM1XlI3oVf9v2yW1sYvWm2n3z4k5w6x7y8z9acc"
  disk_size     = 80

  worker_nodes_choice = {
    no_worker_nodes = true
  }

  logs_receiver_choice = {
    logs_streaming_disabled = true
  }

  blocked_services_choice = {
    block_all_services = true
  }

  direct_connect_choice = {
    direct_connect_disabled = true
  }

  internet_vip_choice = {
    enable_internet_vip = true
  }

  egress_gateway_choice = {
    egress_gateway_default = true
  }

  security_group_choice = {
    f5xc_security_group = true
  }

  routing_type_choice = {
    f5_orchestrated_routing = true
  }
}
