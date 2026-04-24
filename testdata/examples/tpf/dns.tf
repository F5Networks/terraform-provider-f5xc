/*
resource "f5xc_dns_domain" "example" {
  name      = "example.com"
  namespace = "system"
  description = "test domain"
  //txt_record = "abcdefgf"
  domain_choice = {
    volterra_managed = true
  }
  dnssec_mode = ["DNSSEC_ENABLE"]
}
*/

/*
resource "f5xc_dns_zone" "zone" {
  name = "zoneA"
  namespace = "system"
  annotations = {
    a = "b"
    c = "d"
  }
  description = "a zone record"
  dns_type_choice = {
    num_of_dns_records = 1
    primary = {
      soa_record_parameters_choice = {
        default_soa_parameters = true
      }
      rr_set_group = [
        {
          metadata = {
            name        = "meta1"
            description = "a zone example"
          }
          rr_set = [
            {
              description = "rr_set1"
              ttl         = 3600
            }
          ]
        }
      ]
      dnssec_mode = {
        mode_choice = {
          enable = true
        }
      }
      serial = 122312342
      admin  = "admin@example.com"
    }
  }
}
*/

/*
resource "f5xc_dns_zone" "rrset" {
  name = "test.f5xc.app"
  namespace = "system"
  dns_type_choice = {
    primary = {
      dnssec_mode = {
        mode_choice = {
          enable = true
        }
      }
      soa_record_parameters_choice = {
        default_soa_parameters = true
      }

 */

//default_rr_set_group = []
/*
      rr_set_group = [
        {
          metadata = {
            name = "rr-set-group-1"
            description = "rr-set-group-1"
          }
          rr_set = [
            {
              type_record_set_choice = {
                a_record = {
                  values = ["192.168.2.1"]
                  name = "hostA.test.f5xc.app"
                }
              }
            }
          ]
        }
      ]
    }
  }
}
*/