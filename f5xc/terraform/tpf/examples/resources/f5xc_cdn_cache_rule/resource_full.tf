# Example: Cdn Cache Rule - Full

resource "f5xc_cdn_cache_rule" "example" {
  name        = "example-cdn-cache-rule"
  namespace   = "default"
  description = "CDN cache rule full"
  labels = {
    env = "example"
  }
  annotations = {
    owner = "edge"
  }

  cache_rules = {
    rule_name = "rule-main"
    rule_expression_list = [
      {
        expression_name = "expr-main"
        cache_rule_expression = [
          {
            cookie_matcher = [
              {
                name = "session"
                operator = {
                  cache_operator_choice = {
                    contains = "abc"
                  }
                }
              }
            ]
            path_match = {
              operator = {
                cache_operator_choice = {
                  match_regex = "^/api/v1/.*"
                }
              }
            }
            query_parameters = [
              {
                key = "region"
                operator = {
                  cache_operator_choice = {
                    equals = "us"
                  }
                }
              }
            ]
            cache_headers = [
              {
                name = ["SCHEME"]
                operator = {
                  cache_operator_choice = {
                    equals = "https"
                  }
                }
              }
            ]
          }
        ]
      }
    ]

    cache_actions_choice = {
      eligible_for_cache = {
        eligible_for_cache_choice = {
          scheme_proxy_host_uri = {
            cache_ttl              = "15m"
            cache_override         = true
            ignore_response_cookie = true
          }
        }
      }
    }
  }
}
