# Example: Segment - Full

resource "f5xc_segment" "example" {
  name        = "example-segment"
  namespace   = "system"
  description = "segment full"

  labels = {
    env  = "example"
    team = "platform"
  }

  annotations = {
    owner = "example-owner"
  }

  internet_connection_choice = {
    enable = true
  }
}
