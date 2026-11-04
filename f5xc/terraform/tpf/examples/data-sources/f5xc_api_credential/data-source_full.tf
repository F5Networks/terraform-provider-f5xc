# Example: Api Credential - Full

data "f5xc_api_credential" "example" {
  name      = f5xc_api_credential.example.credential_name
  namespace = f5xc_api_credential.example.namespace
}
