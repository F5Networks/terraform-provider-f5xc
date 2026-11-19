# Example: User Group - Full

data "f5xc_user_group" "example" {
  name      = f5xc_user_group.example.name
  namespace = "system"
}
