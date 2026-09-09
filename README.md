<a href="https://terraform.io">
    <img src="images/tf.jpg" alt="Terraform logo" title="Terraform" align="left" height="50" />
</a>

# Terraform Provider for F5XC

The F5XC Terraform Provider allows managing resources within the F5 XC Distributed Cloud Platform.

We recommend using the latest version of Terraform Core ([the latest version can be found here](https://developer.hashicorp.com/terraform/install)).

* [Terraform Website](https://www.terraform.io)
* [F5XC Provider Documentation](https://registry.terraform.io/providers/f5xc/f5xc/latest/docs)
* [F5XC Provider Usage Examples](https://github.com/f5xc/terraform-provider-f5xc/tree/main/examples)
* [Pre-release builds of the F5 Distributed Cloud (`f5xc`) Terraform provider](https://github.com/F5Networks/terraform-provider-f5xc/blob/main/local.md)

## Usage Example

```hcl
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
  # Optional: Backend-driven Throttling (disabled by default unless block is declared)
  throttling = {}
}

# 2. (Alternative) Configure the F5XC Provider using an API Token
provider "f5xc" {
  api_token = "<your_api_token>"
  url       = "https://<tenant_name>.console.ves.volterra.io/api"
  # Optional: Backend-driven Throttling (disabled by default unless block is declared)
  throttling = {
    max_retries         = 5
    initial_backoff     = "250ms"
    max_backoff         = "8s"
    multiplier          = 2.0
    jitter_ratio        = 0.2
    respect_retry_after = true
  }
}
```
