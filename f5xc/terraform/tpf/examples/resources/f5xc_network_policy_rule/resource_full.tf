# Example: Network Policy Rule - Full

resource "f5xc_network_policy_rule" "example" {
  name      = "example-network-policy-rule"
  namespace = "terraform-example"

  labels = {
    env  = "example"
    team = "platform"
  }

  annotations = {
    owner = "example-owner"
  }

  action   = ["ALLOW"]
  protocol = "TCP"
  ports    = ["53", "80", "443", "10000-10010"]

  remote_endpoint_choice = {
    prefix = {
      prefix = ["10.1.0.0/16", "10.2.0.0/16"]
    }
  }

  advanced_action = {
    action = ["LOG"]
  }
}
