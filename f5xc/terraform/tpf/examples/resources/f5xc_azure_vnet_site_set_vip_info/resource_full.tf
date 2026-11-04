# Example: Azure Vnet Site Set Vip Info - Full

resource "f5xc_azure_vnet_site" "site" {
  name      = "example-azure-vnet-vip-info-z1s2lr"
  namespace = "system"

  resource_group = "example-azure-rg"

  region_choice = {
    azure_region = "eastus2"
  }

  vnet = {
    choice = {
      new_vnet = {
        name_choice = {
          autogenerate = true
        }
        primary_ipv4 = "10.30.0.0/16"
      }
    }
  }

  site_type_choice = {
    ingress_gw = {
      azure_certified_hw = "azure-byol-voltmesh"
      az_nodes = [{
        azure_az = "1"
        local_subnet = {
          choice = {
            subnet_param = {
              ipv4 = "10.30.1.0/24"
            }
          }
        }
      }]
    }
  }

  deployment_choice = {
    azure_cred = {
      name      = "example-azure-cred"
      namespace = "system"
    }
  }

  machine_type = "Standard_D4s_v4"
  ssh_key      = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDa7x9o2L3kN5mI4vR8F2cP6qT0eYj1WnZmXpQs3uBhCvDtEr0gK2fL7oM8nA4sazure"

  worker_nodes_choice = {
    no_worker_nodes = true
  }

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

resource "f5xc_azure_vnet_site_set_vip_info" "example" {
  name      = f5xc_azure_vnet_site.site.name
  namespace = "system"

  vip_params_per_az = [
    {
      az_name           = "1"
      outside_vip       = ["40.70.70.70", "40.70.70.71"]
      inside_vip        = ["10.70.70.70", "10.70.70.71"]
      outside_vip_v6    = ["2001:db8::70"]
      inside_vip_v6     = ["2001:db8::71"]
      outside_vip_cname = "outside-full.example.ves.io"
      inside_vip_cname  = "inside-full.example.ves.io"
    },
    {
      az_name     = "2"
      outside_vip = ["40.70.70.72"]
      inside_vip  = ["10.70.70.72"]
    }
  ]
}
