# Example: Aws Tgw Site Set Vip Info - Full

resource "f5xc_aws_tgw_site" "site" {
  name      = "example-aws-tgw-vip-info-0sgele"
  namespace = "system"

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
      disable_internet_vip = true
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

  lifecycle {
    ignore_changes = all
  }
}


resource "f5xc_aws_tgw_site_set_vip_info" "example" {
  name      = f5xc_aws_tgw_site.site.name
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
