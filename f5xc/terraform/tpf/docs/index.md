---
page_title: "Provider: F5 XC"
description: |-
  The F5XC provider is used to configure and manage F5 Distributed Cloud (F5 XC) resources.
---

# F5 XC Provider

The F5 XC provider is used to interact with F5 Distributed Cloud Services (formerly Volterra).

It allows you to manage resources such as load balancers, origin pools, sites, and more.
This provider supports the new Terraform Plugin Framework.

## Example Usage

```terraform
provider "f5xc" {
  url     = "https://<tenant>.console.ves.volterra.io/api"

  # Authentication using API Token
  api_token = "your-api-token"

  # Authentication using PKCS#12 file
  # api_p12_file = "/path/to/your.p12"
  # Note: The password for the p12 file is read from the VES_P12_PASSWORD environment variable.

  # Optional: Rate Limiting
  limiter {
    rate  = 30.0
    burst = 50
  }
}
```

## Schema

### Optional

- `api_ca_cert` (String) The F5XC API CA Cert for api operations. Can also be set via `F5XCProvider_API_CA_CERT` environment variable.
- `api_cert` (String) The F5XC API cert for api operations. Can also be set via `F5XCProvider_API_CERT` environment variable.
- `api_key` (String) The F5XC API key for api operations. Can also be set via `F5XCProvider_API_KEY` environment variable.
- `api_p12_file` (String) Absolute path to p12 file. Can also be set via `F5XCProvider_API_P12_FILE` environment variable.
- `api_token` (String) The F5XC API token for api operations. Can also be set via `F5XCProvider_API_TOKEN` environment variable.
- `limiter` (Attributes) The F5XC API configuration block for API rate limiting. (see [below for nested schema](#nestedatt--limiter))
- `tenant` (String) The Tenant values, used only when Vesenv is set. Can also be set via `F5XCProvider_TENANT` environment variable.
- `test` (Boolean) The F5XC API Test flag. Defaults to false. Can also be set via `F5XCProvider_TEST` environment variable.
- `timeout` (String) The F5XC API call Timeout value, by default its 20s. Can also be set via `F5XCProvider_TIMEOUT` environment variable.
- `url` (String) The F5XC API server url. Can also be set via `F5XCProvider_URL` environment variable.
- `vesenv` (Boolean) The Vesenv flag for internal use. Default is false. Can also be set via `F5XCProvider_VESENV` environment variable.

<a id="nestedatt--limiter"></a>
### Nested Schema for `limiter`

Required:

- `burst` (Number) The maximum burst size for requests. Allows short bursts up to this many requests before rate limiting applies. Minimum value: 1.
- `rate` (Number) The maximum number of requests per minute. This value will be converted to requests per second internally (rate/60). Minimum value: 1.0. Example: rate=60 allows 1 request per second.
