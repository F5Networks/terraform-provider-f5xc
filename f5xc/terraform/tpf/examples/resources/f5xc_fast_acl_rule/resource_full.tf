# Example: Fast Acl Rule - Full

resource "f5xc_fast_acl_rule" "example" {
  name        = "example-fast-acl-rule"
  namespace   = "system"
  description = "Full fast ACL rule example"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "network"
  }

  action = {
    action_choice = {
      simple_action = ["DENY"]
    }
  }

  port = [
    {
      port_value_type_choice = {
        user_defined = 8080
      }
    },
    {
      port_value_type_choice = {
        dns = true
      }
    }
  ]

  source_choice = {
    prefix = {
      prefix      = ["10.40.0.0/16"]
      ipv6_prefix = ["2001:0db8:0010:0000:0000:0000:0000:0000/64"]
    }
  }
}
