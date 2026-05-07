resource "f5xc_cloud_credentials" "example" {
  name      = "example_cloud_credentials"
  namespace = "example_namespace"
  cloud = {
    aws_assume_role = {
      aws_assume_role = "example_aws_assume_role"
    }
  }
}
