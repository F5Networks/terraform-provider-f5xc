# Example: Healthcheck - Full

resource "f5xc_healthcheck" "example" {
  name        = "example-healthcheck"
  namespace   = "default"
  description = "Full healthcheck example"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform"
  }

  timeout             = 3
  interval            = 15
  unhealthy_threshold = 1
  healthy_threshold   = 3

  health_check_choice = {
    http_health_check = {
      host_header_choice = {
        host_header = "example.com"
      }
      path = "/healthcheck"
      headers = {
        "X-Health" = "true"
      }
      request_headers_to_remove = ["User-Agent"]
      use_http2                 = false
      expected_status_codes     = ["200", "204"]
      expected_response         = "0A0B"
    }
  }
}
