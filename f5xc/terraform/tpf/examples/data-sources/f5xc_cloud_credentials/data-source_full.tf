# Example: Cloud Credentials - Full

data "f5xc_cloud_credentials" "example" {
  name      = f5xc_cloud_credentials.example.name
  namespace = f5xc_cloud_credentials.example.namespace
}
