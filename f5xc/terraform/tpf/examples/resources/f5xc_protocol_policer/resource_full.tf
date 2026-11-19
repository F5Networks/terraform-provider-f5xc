# Example: Protocol Policer - Full

resource "f5xc_policer" "example" {
  name                       = "example-protopolicer-pol"
  namespace                  = "system"
  description                = "Policer for protocol policer example"
  committed_information_rate = 10000
  burst_size                 = 10000
  policer_mode               = ["POLICER_MODE_NOT_SHARED"]
}

resource "f5xc_protocol_policer" "example" {
  name        = "example-protopolicer"
  namespace   = "system"
  description = "Full protocol policer example"

  protocol_policer = [{
    protocol = {
      type_choice = {
        tcp = {}
      }
    }

    policer = [{
      name      = f5xc_policer.example.name
      namespace = "system"
    }]
  }]

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform-team"
  }
}
