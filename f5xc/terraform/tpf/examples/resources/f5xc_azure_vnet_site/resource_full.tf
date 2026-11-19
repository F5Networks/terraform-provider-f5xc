# Example: Azure Vnet Site - Full

resource "f5xc_azure_vnet_site" "example" {
  name      = "example-azure-vnet-site-sy4jyw"
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
      az_nodes = [
        {
          azure_az = "1"
          local_subnet = {
            choice = {
              subnet_param = {
                ipv4 = "10.30.1.0/24"
              }
            }
          }
        },
        {
          azure_az = "2"
          local_subnet = {
            choice = {
              subnet_param = {
                ipv4 = "10.30.2.0/24"
              }
            }
          }
        },
        {
          azure_az = "3"
          local_subnet = {
            choice = {
              subnet_param = {
                ipv4 = "10.30.3.0/24"
              }
            }
          }
        }
      ]
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
  disk_size    = 80

  worker_nodes_choice = {
    nodes_per_az = 1
  }

  logs_receiver_choice = {
    logs_streaming_disabled = true
  }

  blocked_services_choice = {
    block_all_services = true
  }
}
