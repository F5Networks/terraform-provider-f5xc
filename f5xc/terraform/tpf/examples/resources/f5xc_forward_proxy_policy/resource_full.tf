# Example: Forward Proxy Policy - Full

resource "f5xc_forward_proxy_policy" "example" {
  name        = "example-forward-proxy-policy"
  namespace   = "system"
  description = "Full forward proxy policy example"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "netops"
  }

  proxy_choice = {
    proxy_label_selector = {
      expressions = ["app == proxy"]
    }
  }

  rule_choice = {
    rule_list = {
      rules = [
        {
          metadata = {
            name        = "example-rule-full"
            description = "allow all sources to http destinations"
          }

          action = ["ALLOW"]

          source_choice = {
            prefix_list = {
              prefixes      = ["10.0.0.0/8"]
              ipv6_prefixes = ["2001:0db8:0000:0000:0000:0000:0000:0000/64"]
            }
          }

          destination_choice = {
            http_list = {
              http_list = [
                {
                  domain_choice = {
                    exact_value = "api.example.com"
                  }
                  path_choice = {
                    any_path = true
                  }
                }
              ]
            }
          }

          http_connect_choice = {
            port_matcher = {
              ports = ["443"]
            }
          }
        }
      ]
    }
  }

  segment_policy = {
    src_segment_choice = {
      src_any = true
    }
    dst_segment_choice = {
      dst_any = true
    }
  }
}
