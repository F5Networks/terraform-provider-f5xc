# Example: Service Credential - Full

data "f5xc_service_credential" "example" {
  name      = f5xc_service_credential.example.credential_name
  namespace = f5xc_service_credential.example.namespace
}
