# Example: Proxy - Basic

resource "f5xc_proxy" "example" {
  name      = "example-proxy"
  namespace = "default"

  proxy_choice = {
    http_proxy = {
      http_https_choice = {
        enable_http = true
      }
    }
  }

  site_choice = {
    do_not_advertise = true
  }

  network_choice = {
    site_local_network = true
  }

  proxy_policy_choice = {
    no_forward_proxy_policy = true
  }

  tls_interception_choice = {
    no_interception = true
  }
}
