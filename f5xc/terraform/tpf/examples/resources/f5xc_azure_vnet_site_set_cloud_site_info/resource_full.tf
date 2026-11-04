# Example: Azure Vnet Site Set Cloud Site Info - Full

resource "f5xc_azure_vnet_site" "site" {
  name      = "example-azure-vnet-cloud-info-jdsn8k"
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
  ssh_key      = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC44d8pO/5vLD4s+roHdypafe7z1WHKMxXwOTzr1Qnlx4KIa3+hiPq8dXAWNO1IEjuVOAuhwbuxH/9yHT6TD3oOUHMU7H2XqHsqZRSXDjWCE+dk97lS79Ny+J4ORa/pwfY1PD8lFKfWv4DlXfeW4lhLurZ65G/65Q3tdpaD2W3dW9KTyTVKPCrD4OMhOWaCUMSDFaVqQ5IgFuHRnscAlkGdym+yCIjO7A0euuco7YnZSgDKc4vVgPCIeWSCMtqgXKM3uC2GO/ZkMAi2LOv1dxC3KBf+28p7WSUBYNyT6oHaayjZBrgvHFj/52nyM8cj40uBGA/gtF3TbyAx/Ru9Xi9DdZ1CFqFWon61iRzJIgoc7n8z7qLedIAv+YQxdDKS8E+BX6SIbp4seyJWP5/b/vOEX19AUxrlFplzieRCIc7M7JfF35uu8Wozyoqd0mowNGPgEboZ6slRGk8sZPNsjRpjNRJt31t1HyNeVPUxC0j7pr6YXdX+WdYHSlPxPiLtOCM="

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


resource "f5xc_azure_vnet_site_set_cloud_site_info" "example" {
  name      = f5xc_azure_vnet_site.site.name
  namespace = "system"

  azure_vnet_info = {
    public_ips  = ["20.50.50.50", "20.50.50.51", "20.50.50.52"]
    private_ips = ["10.50.50.50", "10.50.50.51", "10.50.50.52"]
    express_route_info = {
      route_server_asn = 65445
      route_server_ips = ["10.50.0.4", "10.50.0.5"]
    }
    node_info = [
      {
        node_id            = "ves-node-id-2001"
        node_instance_name = "vm-node-1"
      },
      {
        node_id            = "ves-node-id-2002"
        node_instance_name = "vm-node-2"
      }
    ]
    spoke_vnet_prefix_info = [
      {
        prefixes = ["10.51.0.0/24"]
        vnet = {
          resource_group = "spoke-rg-b"
          vnet_name      = "spoke-vnet-b"
          routing_type_choice = {
            f5_orchestrated_routing = true
          }
        }
      },
      {
        prefixes = ["10.52.0.0/24"]
        vnet = {
          resource_group = "spoke-rg-c"
          vnet_name      = "spoke-vnet-c"
          routing_type_choice = {
            manual_routing = true
          }
        }
      }
    ]
    vnet = {
      vnet_name   = "services-vnet-a"
      resource_id = "/subscriptions/11111111-2222-3333-4444-555555555555/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/services-vnet-a"
    }
  }
}
