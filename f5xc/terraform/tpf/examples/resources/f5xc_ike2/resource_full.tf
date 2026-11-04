# Example: Ike2 - Full

resource "f5xc_ike2" "example" {
  name        = "example-ike2"
  namespace   = "default"
  description = "Full ike2 example"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform-team"
  }
}
