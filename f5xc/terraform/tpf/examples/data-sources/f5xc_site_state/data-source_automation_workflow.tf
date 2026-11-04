# Automation Workflow Example: Wait for CE to become ONLINE before deploying workloads

# 1. Provision the CE (e.g. AWS VPC Site)
resource "f5xc_aws_vpc_site" "example" {
  name      = "my-aws-ce"
  namespace = "system"
  # ... site configuration ...
}

# 2. Query operational state of the CE site
data "f5xc_site_state" "example" {
  name      = f5xc_aws_vpc_site.example.name
  namespace = "system"
}

# 3. Assert CE is ONLINE before continuing dependent operations
resource "terraform_data" "wait_for_ce_online" {
  input = data.f5xc_site_state.example.is_online

  lifecycle {
    postcondition {
      condition     = self.output == true
      error_message = "CE site ${data.f5xc_site_state.example.name} is not online yet! Current state: ${data.f5xc_site_state.example.site_state}"
    }
  }
}
