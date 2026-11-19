# Example: Dns Zone Record - Full

data "f5xc_dns_zone_record" "example" {
  dns_zone_name = f5xc_dns_zone_record.example.dns_zone_name
  group_name    = f5xc_dns_zone_record.example.group_name
  record_name   = f5xc_dns_zone_record.example.record_name
  type          = f5xc_dns_zone_record.example.type
}
