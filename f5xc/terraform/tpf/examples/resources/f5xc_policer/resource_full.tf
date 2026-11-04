# Example: Policer - Full

resource "f5xc_policer" "example" {
  name                       = "example-policer"
  namespace                  = "default"
  description                = "Full policer example"
  committed_information_rate = 10000
  burst_size                 = 10000
  policer_mode               = ["POLICER_MODE_NOT_SHARED"]

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform-team"
  }
}
