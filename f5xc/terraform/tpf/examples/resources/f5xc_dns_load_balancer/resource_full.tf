# Example: Dns Load Balancer - Full

resource "f5xc_dns_lb_pool" "test_pool" {
  name      = "example-dnslbpool-full-c53lww-52000"
  namespace = "system"

  load_balancing_mode = ["ROUND_ROBIN"]
  pool_type_choice = {
    a_pool = {
      health_check_choice = {
        disable_health_check = true
      }
      members = [
        {
          ip_endpoint = "1.2.3.4"
          name        = "member-1"
          priority    = 1
          ratio       = 1
        }
      ]
    }
  }
  ttl_choice = {
    ttl = 30
  }
}

resource "f5xc_dns_load_balancer" "example" {
  name        = "example-dnslb-full-c53lww-42000"
  namespace   = "system"
  description = "Full acceptance test DNS load balancer"

  labels = {
    "env"  = "example"
    "tier" = "dns"
  }

  record_type = ["A"]

  fallback_pool = {
    name      = f5xc_dns_lb_pool.test_pool.name
    namespace = "system"
    tenant    = "customer1"
  }

  response_cache = {
    response_cache_parameters_choice = {
      response_cache_parameters = {
        cache_cidr_ipv4 = 24
        cache_cidr_ipv6 = 56
        cache_ttl       = 3600
      }
    }
  }

  rule_list = {
    rules = [
      {
        action_choice = {
          pool = {
            name      = f5xc_dns_lb_pool.test_pool.name
            namespace = "system"
            tenant    = "customer1"
          }
        }
        client_choice = {
          asn_list = {
            as_numbers = [64512]
          }
        }
        score = 200
      },
      {
        action_choice = {
          pool = {
            name      = f5xc_dns_lb_pool.test_pool.name
            namespace = "system"
            tenant    = "customer1"
          }
        }
        client_choice = {
          ip_prefix_list = {
            ip_prefixes = ["10.0.0.0/8"]
          }
        }
        score = 100
      }
    ]
  }
}
