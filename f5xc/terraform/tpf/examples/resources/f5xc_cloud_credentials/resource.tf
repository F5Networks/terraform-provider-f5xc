resource "f5xc_cloud_credentials" "example" {
  name      = "example_cloud_credentials"
  namespace = "example_namespace"
  cloud = {
    gcp_cred_file = {
      gcp_cred_file = "example_gcp_cred_file"
    }
  }
}
