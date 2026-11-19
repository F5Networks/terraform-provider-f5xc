# Example: V1 Dns Monitor - Full

data "f5xc_v1_dns_monitor" "example" {
  name      = f5xc_v1_dns_monitor.example.name
  namespace = f5xc_v1_dns_monitor.example.namespace
}
