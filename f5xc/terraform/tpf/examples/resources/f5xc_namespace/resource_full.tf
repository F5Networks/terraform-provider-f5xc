# Example: Namespace - Full

resource "f5xc_namespace" "example" {
  name        = "example-namespace"
  description = "Full namespace example"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform-team"
  }
}
