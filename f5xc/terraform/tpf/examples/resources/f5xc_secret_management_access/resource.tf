resource "f5xc_secret_management_access" "example" {
  name          = "example_secret_management_access"
  namespace     = "example_namespace"
  provider_name = "example_secret_management_access"
  access_info = {
    server_endpoint = "example_secret_management_access"
  }
}
