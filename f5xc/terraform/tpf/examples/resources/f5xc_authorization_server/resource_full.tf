# Example: Authorization Server - Full

resource "f5xc_authorization_server" "example" {
  name        = "example-authorization-server"
  namespace   = "default"
  description = "Full authorization server example"
  jwks_uri    = "https://www.googleapis.com/oauth2/v3/certs"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform-team"
  }
}
