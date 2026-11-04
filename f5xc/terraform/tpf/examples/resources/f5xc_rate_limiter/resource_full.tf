# Example: Rate Limiter - Full

resource "f5xc_rate_limiter" "example" {
  name        = "example-ratelimiter"
  namespace   = "default"
  description = "Full rate limiter example"

  limits = [{
    total_number = 100
    unit         = ["MINUTE"]

    burst_multiplier = 2

    action_choice = {
      disabled = true
    }
  }]

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform-team"
  }
}
