resource "f5xc_v1_http_monitor" "example" {
  name      = "example_v1_http_monitor"
  namespace = "example_namespace"
  method = {
    put = {
      put = "example_put"
    }
  }
  external_sources = [{
    source_choice = {
      aws = {
        aws = "example_aws"
      }
    }
  }]
  url            = "example_v1_http_monitor"
  response_codes = ["example_v1_http_monitor"]
  interval_choice = {
    interval_1_day = true
  }
}
