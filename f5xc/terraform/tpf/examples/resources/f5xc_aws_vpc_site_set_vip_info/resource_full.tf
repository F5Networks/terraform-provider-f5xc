# Example: Aws Vpc Site Set Vip Info - Full

resource "f5xc_aws_vpc_site" "site" {
  name      = "example-aws-vpc-vip-info-8h69v5-44000"
  namespace = "system"

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
      name      = "arbell-aws"
      namespace = "system"
    }
  }

  instance_type = "t3.xlarge"
  aws_region    = "us-west-2"
  ssh_key       = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3b9ZcE4fVYI7xQ1jQk7n4k3bM1XlI3oVf9v2yW1sYvWm2n3z4k5w6x7y8z9acc"

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
    disable_internet_vip = true
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

  lifecycle {
    ignore_changes = all
  }
}


resource "f5xc_aws_vpc_site_set_vip_info" "example" {
  name      = f5xc_aws_vpc_site.site.name
  namespace = "system"

  vip_params_per_az = [
    {
      az_name           = "us-east-1a"
      outside_vip       = ["40.70.70.70", "40.70.70.71"]
      inside_vip        = ["10.70.70.70", "10.70.70.71"]
      outside_vip_v6    = ["2001:db8::70"]
      inside_vip_v6     = ["2001:db8::71"]
      outside_vip_cname = "outside-full.example.ves.io"
      inside_vip_cname  = "inside-full.example.ves.io"
    },
    {
      az_name     = "us-east-1b"
      outside_vip = ["40.70.70.72"]
      inside_vip  = ["10.70.70.72"]
    }
  ]
}
