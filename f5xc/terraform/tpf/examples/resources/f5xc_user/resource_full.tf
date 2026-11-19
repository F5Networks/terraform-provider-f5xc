# Example: User - Full

resource "f5xc_user" "example" {
  email      = "example-user@example.com"
  first_name = "Example"
  last_name  = "User"
  namespace  = "system"
  type       = ["USER"]
}
