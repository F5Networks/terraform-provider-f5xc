# Example: Api Definition - Full

resource "f5xc_api_definition_uploader" "spec" {
  metadata = {
    name      = "example-api-definition-spec"
    namespace = "shared"
  }
  string_value = "{\"openapi\":\"3.0.0\",\"info\":{\"title\":\"Full Test API\",\"version\":\"1.0.0\"},\"paths\":{\"/2.0/users/{username}\":{\"get\":{\"operationId\":\"getUserByName\",\"parameters\":[{\"name\":\"username\",\"in\":\"path\",\"required\":true,\"schema\":{\"type\":\"string\"}}],\"responses\":{\"200\":{\"description\":\"The User\"}}}},\"/2.0/repositories/{username}\":{\"get\":{\"operationId\":\"getRepositoriesByOwner\",\"parameters\":[{\"name\":\"username\",\"in\":\"path\",\"required\":true,\"schema\":{\"type\":\"string\"}}],\"responses\":{\"200\":{\"description\":\"repos\"}}}}}}"
}

resource "f5xc_api_definition" "example" {
  name      = "example-api-definition"
  namespace = "shared"

  swagger_specs = [f5xc_api_definition_uploader.spec.metadata.url]

  # Path is intentionally NOT present in the inline swagger above.
  # Under mixed_schema_origin the API drops inclusion-list entries whose
  # path is already declared in the swagger (auto-discovered), which
  # causes a "provider produced inconsistent result after apply" error.
  api_inventory_inclusion_list = [
    {
      method = ["GET"]
      path   = "/2.0/starred/{username}"
    }
  ]

  non_api_endpoints = [
    {
      method = ["HEAD"]
      path   = "/readyz"
    }
  ]

  description = "API definition example"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform"
  }

  schema_updates_strategy_choice = {
    mixed_schema_origin = true
  }
}
