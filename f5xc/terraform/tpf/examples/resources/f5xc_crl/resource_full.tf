# Example: Crl - Full

resource "f5xc_crl" "example" {
  name             = "example-crl-f2iat7n5"
  namespace        = "default"
  description      = "CRL example"
  server_address   = "mycrl.example.com"
  server_port      = 80
  refresh_interval = 24
  timeout          = 10

  labels = {
    env  = "example"
    tier = "shared"
  }

  annotations = {
    owner = "platform"
    team  = "security"
  }

  access_info_choice = {
    http_access = {
      path = "/repo/latest.crl"
    }
  }
}
