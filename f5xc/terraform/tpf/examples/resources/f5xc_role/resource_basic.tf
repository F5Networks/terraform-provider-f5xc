# Example: Role - Basic

resource "f5xc_role" "example" {
  name       = "example-role"
  namespace  = "system"
  api_groups = ["ves-io-web-access-read"]
}
