resource "f5xc_v1_dns_monitor" "example" {
  name      = "example_v1_dns_monitor"
  namespace = "example_namespace"
  domain    = "example_v1_dns_monitor"
  interval_choice = {
    interval_30_secs = true
  }
  external_sources = [{
    source_choice = {
      aws = {
        aws = "example_aws"
      }
    }
  }]
  on_failure_to_choice = {
    on_failure_to_any = true
  }
}
