# Example: Dns Zone - Full

resource "f5xc_dns_zone" "example" {
  name        = "example-dns-zone-vvg93b-248000.example.com"
  namespace   = "system"
  description = "DNS zone example"

  labels = {
    env  = "acceptance"
    team = "example-team"
  }

  annotations = {
    owner    = "dns-team"
    scenario = "list-growth"
  }

  dns_type_choice = {
    primary = {
      soa_record_parameters_choice = {
        default_soa_parameters = true
      }
      default_rr_set_group = [
        {
          description = "public web record"
          ttl         = 300
          type_record_set_choice = {
            a_record = {
              name   = "www.example-dns-zone-vvg93b-248000.example.com"
              values = ["192.168.1.10"]
            }
          }
        }

      ]
      rr_set_group = [
        {
          metadata = {
            name        = "app-records"
            description = "application records"
          }
          rr_set = [
            {
              description = "app record"
              ttl         = 120
              type_record_set_choice = {
                a_record = {
                  name   = "app.example-dns-zone-vvg93b-248000.example.com"
                  values = ["10.10.0.10"]
                }
              }
            }
          ]
        },
        {
          metadata = {
            name        = "api-records"
            description = "api records"
          }
          rr_set = [
            {
              description = "api record"
              ttl         = 300
              type_record_set_choice = {
                a_record = {
                  name   = "api.example-dns-zone-vvg93b-248000.example.com"
                  values = ["10.10.0.20"]
                }
              }
            }
          ]
        }
      ]
    }
  }
}
