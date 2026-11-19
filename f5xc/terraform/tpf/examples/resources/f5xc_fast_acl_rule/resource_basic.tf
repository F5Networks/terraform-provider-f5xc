# Example: Fast Acl Rule - Basic

resource "f5xc_fast_acl_rule" "example" {
  name      = "example-fast-acl-rule"
  namespace = "system"

  action = {
    action_choice = {
      simple_action = ["DENY"]
    }
  }

  port = [
    {
      port_value_type_choice = {
        all = true
      }
    }
  ]

  source_choice = {
    prefix = {
      prefix = ["10.0.0.0/8"]
    }
  }
}
