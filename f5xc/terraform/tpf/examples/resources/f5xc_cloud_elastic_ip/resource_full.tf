# Example: Cloud Elastic Ip - Full

resource "f5xc_azure_vnet_site" "site" {
  name      = "example-azure-vnet-site"
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

resource "f5xc_cloud_elastic_ip" "example" {
  name      = "example-cloud-elastic-ip"
  namespace = "system"

  site_ref = [
    {
      name      = f5xc_azure_vnet_site.site.name
      namespace = f5xc_azure_vnet_site.site.namespace
    }
  ]

  count_cloud_elastic_ip = 2
}
