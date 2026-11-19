# Example: Cloud Credentials - Basic

resource "f5xc_cloud_credentials" "example" {
  name      = "example-cloud-cred"
  namespace = "system"

  cloud_choice = {
    aws_secret_key = {
      access_key = "AKIAEXAMPLE123"
      secret_key = {
        secret_info_choice = {
          clear_secret_info = {
            url = "string:///U0VDUkVUMQ=="
          }
        }
      }
    }
  }
}
