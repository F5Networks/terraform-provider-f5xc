# Example: Authorization Server - Full

data "f5xc_authorization_server" "example" {
  name      = f5xc_authorization_server.example.name
  namespace = f5xc_authorization_server.example.namespace
}
