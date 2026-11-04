# Example: Dns Lb Health Check - Full

resource "f5xc_dns_lb_health_check" "example" {
  name        = "example-dns-lb-health-check-ee5xwjdo"
  namespace   = "system"
  description = "DNS LB health check example"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform"
  }

  health_check_choice = {
    https_health_check = {
      health_check_port           = 443
      health_check_secondary_port = 8443
      send                        = "HEAD / HTTP/1.0\\r\\n\\r\\n"
      receive                     = "HTTP/1."
      virtual_host_choice = {
        virtual_host = "full.example.com"
      }
    }
  }
}
