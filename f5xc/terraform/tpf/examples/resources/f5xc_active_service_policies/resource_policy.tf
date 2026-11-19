# Example: Active Service Policies - Policy

resource "f5xc_service_policy" "policy_one" {
  name      = "example-active-service-policy-one-84vt1kz4"
  namespace = "default"

  server_choice = {
    any_server = true
  }

  rule_choice = {
    allow_all_requests = true
  }
}


resource "f5xc_active_service_policies" "example" {
  namespace = "default"

  service_policies = [
    {
      name      = f5xc_service_policy.policy_one.name
      namespace = f5xc_service_policy.policy_one.namespace
    }
  ]
}
