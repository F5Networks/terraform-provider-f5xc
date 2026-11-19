# Example: User Group - Full

resource "f5xc_user" "member" {
	email      = "user1@example.com"
	first_name = "User"
	last_name  = "One"
	namespace  = "system"
	type       = ["USER"]
}

resource "f5xc_user_group" "example" {
  name         = "example-user-group"
  description  = "User group full example"
  display_name = "User Group Example"

  namespace_roles = [
    {
      namespace = "system"
      role      = "ves-io-monitor-role"
    }
  ]

	usernames = [f5xc_user.member.email]
}
