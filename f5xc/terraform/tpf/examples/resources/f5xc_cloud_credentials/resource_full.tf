# Example: Cloud Credentials - Full

resource "f5xc_cloud_credentials" "example" {
  name        = "example-cloud-cred"
  namespace   = "system"
  description = "cloud credentials full"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform"
  }

  cloud_choice = {
    aws_assume_role = {
      role_arn         = "arn:aws:iam::123456789012:role/f5xc-role"
      session_name     = "cloud-f5xc-deployment"
      duration_seconds = 3600
      session_tags = {
        env  = "example"
        team = "infra"
      }
    }
  }
}
