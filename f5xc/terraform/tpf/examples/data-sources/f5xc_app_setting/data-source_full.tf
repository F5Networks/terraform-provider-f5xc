# Example: App Setting - Full

data "f5xc_app_setting" "example" {
  name      = f5xc_app_setting.example.name
  namespace = f5xc_app_setting.example.namespace
}
