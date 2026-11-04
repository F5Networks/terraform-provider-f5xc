# Example: Bot Infrastructure

resource "f5xc_bot_infrastructure" "example" {
  name             = "example-bot-infrastructure"
  namespace        = "default"
  description      = "Production bot defense infrastructure"
  environment_type = ["PRODUCTION"]
  traffic_type     = ["WEB"]
  cluster_state    = ["ACTIVE"]
  deployment_mode  = ["REVERSE_PROXY"]

  labels = {
    environment = "example"
    team        = "security"
  }

  annotations = {
    owner = "platform-team"
  }

  bot_infra_choice = {
    physical_hosted = {
      infra_host_name = "N/A"
    }
  }

  create_bot_infra_choice = {
    create_cloud_hosted = {
      type_choice = {
        production_v2 = {
          type_choice_production_choice = {
            active_active_region = {
              region_1 = "US_WEST_2"
              region_2 = "US_EAST_1"
            }
          }
        }
      }

      ip_address_list = {
        ip_addresses = ["192.0.2.11/32"]
      }
    }
  }
}