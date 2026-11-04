# Example: Dns Zone Record - Full

resource "f5xc_dns_zone" "zone" {
  name      = "example-dns-zone-91b5rt-454000.example.com"
  namespace = "system"

  dns_type_choice = {
    primary = {
      soa_record_parameters_choice = {
        default_soa_parameters = true
      }
    }
  }
}


resource "f5xc_dns_zone_record" "example" {
  dns_zone_name = f5xc_dns_zone.zone.name
  group_name    = "default"

  rrset = {
    description = "DNS zone record full example"
    ttl         = 300

    type_record_set_choice = {
      a_record = {
        name   = "api"
        values = ["1.2.3.10"]
      }
    }
  }
}
