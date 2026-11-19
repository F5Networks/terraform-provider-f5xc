# Example: Proxy - Proxy Policy Transition

resource "f5xc_forward_proxy_policy" "example" {
  name      = "example-fpp"
  namespace = "default"

  proxy_choice = {
    drp_http_connect = true
  }

  rule_choice = {
    allow_all = true
  }
}

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
    active_forward_proxy_policies = {
      forward_proxy_policies = [{
        name      = "example-fpp"
        namespace = "default"
      }]
    }
  }

  tls_interception_choice = {
    no_interception = true
  }

  depends_on = [f5xc_forward_proxy_policy.example]
}
