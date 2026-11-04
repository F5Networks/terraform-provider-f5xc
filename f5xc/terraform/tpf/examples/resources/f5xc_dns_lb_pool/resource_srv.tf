# Example: Dns Lb Pool - Srv

resource "f5xc_dns_lb_pool" "example" {
  name      = "example-dns-lb-pool-cabig8e4"
  namespace = "system"

  pool_type_choice = {
    srv_pool = {
      members = [
        {
          target            = "svc.example.com"
          port              = 443
          priority          = 10
          weight            = 10
          ratio             = 1
          name              = "srv-member"
          final_translation = false
        }
      ]
      max_answers = 1
    }
  }

  load_balancing_mode = ["ROUND_ROBIN"]

  ttl_choice = {
    ttl = 30
  }
}
