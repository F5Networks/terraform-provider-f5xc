# 1. Specify the version of the F5XC Provider to use
terraform {
  required_providers {
    f5xc = {
      source  = "F5Networks/f5xc"
      version = ">=1.0.0"
    }
  }
}

# 2. Configure the F5XC Provider.
# Set F5XCProvider_API_TOKEN in the environment or use one of the other
# authentication methods documented in the provider authentication guide.
provider "f5xc" {
  url = "https://<tenant_name>.console.ves.volterra.io/api"

  # Declaring the block enables retries for HTTP 429 responses.
  throttling = {
    max_retries         = 5
    initial_backoff     = "250ms"
    max_backoff         = "8s"
    multiplier          = 2.0
    jitter_ratio        = 0.2
    respect_retry_after = true
  }
}
