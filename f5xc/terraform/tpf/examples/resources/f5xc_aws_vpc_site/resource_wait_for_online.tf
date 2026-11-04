# Automation Workflow Example: Verify CE operational state after provisioning

resource "f5xc_aws_vpc_site" "example" {
  name      = "my-aws-ce"
  namespace = "system"
  # ... site configuration ...
}

# Fetch operational state of the provisioned CE
data "f5xc_site_state" "example" {
  name      = f5xc_aws_vpc_site.example.name
  namespace = "system"
}

# Wait/Assert CE is ONLINE
resource "terraform_data" "wait_for_ce_online" {
  input = data.f5xc_site_state.example.is_online

  lifecycle {
    postcondition {
      condition     = self.output == true
      error_message = "CE site ${data.f5xc_site_state.example.name} is not online yet! Current state: ${data.f5xc_site_state.example.site_state}"
    }
  }
}
