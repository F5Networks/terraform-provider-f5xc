# 1. Specify the version of the F5XC Provider to use
terraform {
  required_providers {
    f5xc = {
      source  = "F5Networks/f5xc"
      version = ">=1.0.0"
    }
  }
}

# 2. Configure the F5XC Provider using an API P12 file
provider "f5xc" {
  api_p12_file = "/path/to/api_credential.p12"
  url          = "https://<tenant_name>.console.ves.volterra.io/api"
  throttling = {
    enabled             = true
    max_retries         = 5
    initial_backoff     = "250ms"
    max_backoff         = "8s"
    multiplier          = 2.0
    jitter_ratio        = 0.2
    respect_retry_after = true
  }
}

# 2. (Alternative) Configure the F5XC Provider using an API Token
provider "f5xc" {
  api_token = "<your_api_token>"
  url       = "https://<tenant_name>.console.ves.volterra.io/api"
  throttling = {
    enabled             = true
    max_retries         = 5
    initial_backoff     = "250ms"
    max_backoff         = "8s"
    multiplier          = 2.0
    jitter_ratio        = 0.2
    respect_retry_after = true
  }
}