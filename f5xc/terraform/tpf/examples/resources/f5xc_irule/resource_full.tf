# Example: Irule - Full

resource "f5xc_irule" "example" {
  name              = "example-irule"
  namespace         = "default"
  description       = "Full irule example"
  irule_description = "Full irule example"
  irule             = base64encode("when CLIENT_ACCEPTED { log local0. \"client accepted\" }")

  labels = {
    env  = "example"
    tier = "shared"
  }

  annotations = {
    owner = "platform-team"
    team  = "edge"
  }
}
