# Example: Cdn Loadbalancer - Basic

resource "f5xc_cdn_loadbalancer" "example" {
  name      = "example-cdn-lb"
  namespace = "default"
  domains   = ["example-cdn-lb.cdn-acc.example.com"]

  waf_choice = {
    disable_waf = true
  }

  l7_ddos_auto_mitigation_action_choice = {
    l7_ddos_action_default = true
  }

  loadbalancer_type_choice = {
    http = {
      port_choice = {
        port = 80
      }
    }
  }

  slow_ddos_mitigation_choice = {
    system_default_timeouts = true
  }

  origin_pool = {
    origin_servers = [{
      choice = {
        public_ip = {
          public_ip_choice = {
            ip = "203.0.113.10"
          }
        }
      }
    }]
    tls_choice = {
      no_tls = true
    }
  }

  service_policy_choice = {
    no_service_policies = true
  }
}
