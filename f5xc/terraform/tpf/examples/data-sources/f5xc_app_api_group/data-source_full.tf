# Example: App API Group - Full

data "f5xc_app_api_group" "example" {
  name      = f5xc_app_api_group.example.name
  namespace = f5xc_app_api_group.example.namespace
}
