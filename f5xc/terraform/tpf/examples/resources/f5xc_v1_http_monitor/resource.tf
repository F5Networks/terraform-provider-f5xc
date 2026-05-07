resource "f5xc_v1_http_monitor" "example" {
  name      = "example_v1_http_monitor"
  namespace = "example_namespace"
  method = {
    options = true
  }
  external_sources = [{
    source_choice = {
      aws = {
        aws = "example_aws"
      }
    }
  }]
  response_codes = ["example_v1_http_monitor"]
  url            = "example_v1_http_monitor"
  interval_choice = {
    interval_5_mins = true
  }
}
