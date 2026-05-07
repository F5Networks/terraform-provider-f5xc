resource "f5xc_app_setting" "example" {
  name      = "example_app_setting"
  namespace = "example_namespace"
  app_type_settings = [{
    app_type_ref = [{
      name      = "example_app_setting"
      namespace = "example_namespace"
    }]
  }]
}
