# Example: Tf Params Action - Basic

resource "f5xc_aws_vpc_site" "site" {
  name      = "example-tf-params-action-bn7a9d-909000"
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

resource "f5xc_tf_params_action" "example" {
  namespace       = f5xc_aws_vpc_site.site.namespace
  view_kind       = "aws_vpc_site"
  view_name       = f5xc_aws_vpc_site.site.name
  action          = ["PLAN"]
  wait_for_action = true

  timeouts = {
    create = "30m"
    read   = "10m"
    delete = "10m"
  }

  depends_on = [f5xc_aws_vpc_site.site]
}
