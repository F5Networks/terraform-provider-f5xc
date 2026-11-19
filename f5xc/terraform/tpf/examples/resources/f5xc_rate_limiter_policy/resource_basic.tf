# Example: Rate Limiter Policy - Basic

resource "f5xc_rate_limiter_policy" "example" {
  name      = "example-rate-limiter-policy"
  namespace = "shared"

  rules = [
    {
      metadata = {
        name = "example-rule"
      }
      spec = {
        action_choice = {
          apply_rate_limiter = true
        }
        country_choice = {
          any_country = true
        }
      }
    }
  ]
}
