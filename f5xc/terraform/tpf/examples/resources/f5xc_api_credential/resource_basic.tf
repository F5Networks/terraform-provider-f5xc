# Example: Api Credential - Basic

resource "f5xc_api_credential" "example" {
  namespace       = "system"
  name            = "example-api-credential"
  expiration_days = 30

  spec = {
    type     = ["API_CERTIFICATE"]
    password = "TestPassword123!"
  }
}
