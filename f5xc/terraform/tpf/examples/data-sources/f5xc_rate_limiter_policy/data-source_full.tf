# Example: Rate Limiter Policy - Full

data "f5xc_rate_limiter_policy" "example" {
  name      = f5xc_rate_limiter_policy.example.name
  namespace = f5xc_rate_limiter_policy.example.namespace
}
