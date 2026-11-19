# Example: Ike1 - Full

resource "f5xc_ike1" "example" {
  name        = "example-ike1"
  namespace   = "default"
  description = "Full ike1 example"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform-team"
  }
}
