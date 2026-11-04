# Example: V1 Http Monitor - Full

data "f5xc_v1_http_monitor" "example" {
  name      = f5xc_v1_http_monitor.example.name
  namespace = f5xc_v1_http_monitor.example.namespace
}
