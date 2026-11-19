# Example: Token - Full

resource "f5xc_token" "example" {
  name        = "example-token"
  namespace   = "system"
  description = "Token full example"

  labels = {
    team = "edge"
    env  = "example"
  }

  annotations = {
    owner = "platform-team"
  }
}
