# Example: V1 Http Monitor - Full

resource "f5xc_v1_http_monitor" "example" {
  name      = "example-v1-http-monitor"
  namespace = "system"
  url       = "https://example.example.com/full"

  interval_choice = {
    interval_15_mins = true
  }

  method_choice = {
    post = { request_body = "{\"message\":\"ok\"}" }
  }

  external_sources = [
    {
      source_choice = {
        f5xc = { regions = ["ves-io-newyork"] }
      }
    }
  ]

  response_codes = ["2**", "3**"]

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform"
  }

  request_headers = [
    {
      key   = "Content-Type"
      value = "application/json"
    }
  ]

  on_failure_count          = 2
  receive                   = "ok"
  ignore_cert_errors        = true
  follow_redirects          = true
  response_timeout          = 3000
  source_critical_threshold = 1

}
