# Example: Dns Proxy - Full

resource "f5xc_dns_proxy" "example" {
  name      = "example-dns-proxy-ctfl2vwr"
  namespace = "default"

  transport_type = ["TCP"]

  cache_profile = {
    cache_profile_choice = {
      cache_size = 256
    }
  }

  ddos_profile = {
    ddos_mitigation_choice = {
      disable_ddos_mitigation = true
    }
  }

  origin_servers = {
    health_checks = {
      health_check = [{
        choice = {
          icmp_health_check = true
        }
      }]
    }

    origin_servers = [{
      choice = {
        public_name = {
          dns_name = "example.com"
        }
      }
    }]
  }

  description = "DNS proxy full example"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform-team"
  }
}
