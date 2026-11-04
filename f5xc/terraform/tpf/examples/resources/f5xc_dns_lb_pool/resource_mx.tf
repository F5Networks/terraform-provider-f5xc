# Example: Dns Lb Pool - Mx

resource "f5xc_dns_lb_pool" "example" {
  name      = "example-dns-lb-pool-uvol2hfz"
  namespace = "system"

  pool_type_choice = {
    aaaa_pool = {
      members = [
        {
          ip_endpoint = "2001:db8::1"
          name        = "aaaa-member"
          ratio       = 1
          priority    = 10
        }
      ]
      max_answers = 1
    }
  }

  load_balancing_mode = ["RATIO_MEMBER"]

  ttl_choice = {
    ttl = 60
  }
}
