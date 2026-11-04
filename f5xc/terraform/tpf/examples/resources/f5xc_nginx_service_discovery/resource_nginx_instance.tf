# Example: NGINX Service Discovery - NGINX Instance

resource "f5xc_nginx_service_discovery" "example" {
  name        = "example-nginx-service-discovery"
  namespace   = "default"
  description = "Discover HTTPS server blocks from an NGINX instance"

  discovery_target = {
    target_choice = {
      nginx_instance = {
        nginx_instance = [
          {
            name      = "nginx-instance-1"
            namespace = "default"
          }
        ]
      }
    }
  }

  server_block_filters = [
    {
      name_regex  = "^.*\\.example\\.com$"
      port_ranges = "80,443"
    }
  ]
}