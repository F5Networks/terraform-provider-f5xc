<a href="https://terraform.io">
    <img src="images/tf.jpg" alt="Terraform logo" title="Terraform" align="left" height="50" />
</a>

# Terraform Provider for F5XC

The F5XC Terraform Provider allows managing resources within the F5 XC Distributed Cloud Platform.

We recommend using the latest version of Terraform Core ([the latest version can be found here](https://developer.hashicorp.com/terraform/install)).

* [Terraform Website](https://www.terraform.io)
* [F5XC Provider Documentation](https://registry.terraform.io/providers/f5xc/f5xc/latest/docs)
* [F5XC Provider Usage Examples](https://github.com/f5xc/terraform-provider-f5xc/tree/main/examples)

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

# Support & Community Resources

This guide helps you quickly find support, community resources, and share new ideas related to the F5 XC platform.

**Please do not open GitHub issues for support requests or feature ideas**. Please use the channels below instead.

---

## Report an Issue

Create a support ticket through the **F5 XC Console** by following these steps:

1. Log in to the [F5 XC Console](https://console.ves.volterra.io/) (F5 customer access required)
2. Navigate to **Support** in the top right and select **Contact Support**
3. Select **Automation** as the workspace.
4. Set **Type** to **Technical Support**.
5. Choose **Topic** as **Terraform Provider**.

If you do not have F5 customer access, contact your account manager or visit [F5 Support](https://my.f5.com/manage/s/createcase) to get started.

---

## Feature Ideas & Feedback

If you have a suggestion for the Terraform provider or the XC platform please share it on the **F5 Cloud Ideas Portal**:

- **Link:** https://www.f5cloudideas.com/ideas/new
- **Workspace:** XC Console
- **Category:** Tools

---

## Community Support

For examples, best practices, troubleshooting guidance, or peer discussions please visit **F5 DevCentral** for community-driven content:
https://community.f5.com/

---

Your feedback and participation help improve the platform for everyone.
