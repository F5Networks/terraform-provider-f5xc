# Example: User Identification - Full

resource "f5xc_user_identification" "example" {
  name        = "example-user-identification"
  namespace   = "default"
  description = "User identification full example"

  rules = [{
    identifier_choice = {
      client_ip = true
    }
    },
    {
      identifier_choice = {
        http_header_name = "X-User-ID"
      }
  }]

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform-team"
  }
}
