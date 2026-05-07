resource "f5xc_v1_dns_monitor" "example" {
  name      = "example_v1_dns_monitor"
  namespace = "example_namespace"
  on_failure_to_choice = {
    on_failure_to_all = true
  }
  domain = "example_v1_dns_monitor"
  interval_choice = {
    interval_1_hour = true
  }
  external_sources = [{
    source_choice = {
      f5xc = {
        f5xc = "example_f5xc"
      }
    }
  }]
}
