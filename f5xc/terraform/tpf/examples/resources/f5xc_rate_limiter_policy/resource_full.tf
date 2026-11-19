# Example: Rate Limiter Policy - Full

resource "f5xc_rate_limiter_policy" "example" {
  name        = "example-rate-limiter-policy"
  namespace   = "shared"
  description = "Full rate limiter policy example"

  labels = {
    env  = "example"
    team = "platform"
  }

  annotations = {
    owner = "example-owner"
  }

  rules = [
    {
      metadata = {
        name        = "example-rule-full"
        description = "Full rule example"
      }
      spec = {
        action_choice = {
          apply_rate_limiter = true
        }
        country_choice = {
          country_list = {
            country_codes = ["COUNTRY_US"]
          }
        }
        http_method = {
          methods = ["GET"]
        }
        path = {
          exact_values = ["/api/login"]
          transformers = ["LOWER_CASE"]
        }
        domain_matcher = {
          exact_values = ["example-domain.com"]
        }
      }
    }
  ]
}
