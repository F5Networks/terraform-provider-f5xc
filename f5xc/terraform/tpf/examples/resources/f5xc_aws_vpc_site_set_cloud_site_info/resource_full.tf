# Example: Aws Vpc Site Set Cloud Site Info - Full

resource "f5xc_aws_vpc_site" "site" {
  name      = "example-aws-vpc-cloud-info-hb9ck7"
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
      name      = "eric-aws-credential"
      namespace = "system"
    }
  }

  instance_type = "t3.xlarge"
  aws_region    = "us-west-2"
  ssh_key       = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDwaJ154QAcu1rsnjxA21GaW0bdYHf2D+vot5XfcS7LbDok/2HEVBA3y2nVm6MQVHnx9n3QtOnVWkggz0jMruUqZ3Yghde4YjDXN7idZQmVNXT4ELL4WzcaskMdx+EJSL7VStPOo9uIIJDhimnkgVhG8O2euCEAo1xhlVbZTACR34isCpO9EsGfXnOkHPpxigl5f2bi+mLHAbRF38zWwx5F7yDLsHsGKlA5Ymf7NQLWGs/Sfy/eEy44Cen3JiHnBt4OjsTruU2S8UyHYJ03oRAdBYebf+FuhTKi2v1Z0ArGQd9OHZJfnDRtRqRNIr55RzUHpgqYlL3dl6NzMqHI7Z9R y.kuo@FRVQMCXG9D"

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


resource "f5xc_aws_vpc_site_set_cloud_site_info" "example" {
  name      = f5xc_aws_vpc_site.site.name
  namespace = "system"

  aws_vpc_info = {
    public_ips  = ["52.10.50.10", "52.10.50.11", "52.10.50.12"]
    private_ips = ["10.10.50.10", "10.10.50.11", "10.10.50.12"]
    vpc_id      = "vpc-1234abcd"
    vpc_name    = "acc-vpc-full-a"
    subnet_ids = [
      {
        outside_subnet = {
          az_name     = "us-west-2a"
          id          = "subnet-d345a789"
          ipv4_prefix = "10.50.1.0/24"
        }
        inside_subnet = {
          az_name     = "us-west-2a"
          id          = "subnet-e456b890"
          ipv4_prefix = "10.50.2.0/24"
        }
        workload_subnet = {
          az_name     = "us-west-2a"
          id          = "subnet-f567c901"
          ipv4_prefix = "10.50.3.0/24"
        }
      },
      {
        outside_subnet = {
          az_name     = "us-west-2b"
          id          = "subnet-0123dabc"
          ipv4_prefix = "10.50.11.0/24"
        }
        inside_subnet = {
          az_name     = "us-west-2b"
          id          = "subnet-1234ebcd"
          ipv4_prefix = "10.50.12.0/24"
        }
        workload_subnet = {
          az_name     = "us-west-2b"
          id          = "subnet-2345fcde"
          ipv4_prefix = "10.50.13.0/24"
        }
      },
      {
        outside_subnet = {
          az_name     = "us-west-2c"
          id          = "subnet-3456ddef"
          ipv4_prefix = "10.50.21.0/24"
        }
        inside_subnet = {
          az_name     = "us-west-2c"
          id          = "subnet-4567efgh"
          ipv4_prefix = "10.50.22.0/24"
        }
        workload_subnet = {
          az_name     = "us-west-2c"
          id          = "subnet-5678fghi"
          ipv4_prefix = "10.50.23.0/24"
        }
      }
    ]
  }

  direct_connect_info = {
    direct_connect_gateway_id = "f2a50c04-dddd-eeee-ffff-00000000a043"
    vgw_id                    = "vgw-5678ef12"
    asn                       = 64514
  }
}
