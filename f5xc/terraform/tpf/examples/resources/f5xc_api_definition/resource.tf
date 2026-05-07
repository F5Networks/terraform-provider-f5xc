resource "f5xc_api_definition" "example" {
  name      = "example_api_definition"
  namespace = "example_namespace"
  schema_updates_strategy = {
    strict_schema_origin = true
  }
}
