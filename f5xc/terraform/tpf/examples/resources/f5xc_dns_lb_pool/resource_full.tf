# Example: Dns Lb Pool - Full

resource "f5xc_dns_lb_pool" "example" {
  name        = "example-dns-lb-pool-v2vvqqz4"
  namespace   = "system"
  description = "DNS LB pool example"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform"
  }

  pool_type_choice = {
    a_pool = {
      members = [
        {
          ip_endpoint = "9.9.9.9"
          name        = "member-1"
          ratio       = 10
          priority    = 5
        },
        {
          ip_endpoint = "8.8.8.8"
          name        = "member-2"
          ratio       = 20
          priority    = 10
        }
      ]
      max_answers = 2
      health_check_choice = {
        disable_health_check = true
      }
    }
  }

  load_balancing_mode = ["RATIO_MEMBER"]

  ttl_choice = {
    ttl = 180
  }
}
