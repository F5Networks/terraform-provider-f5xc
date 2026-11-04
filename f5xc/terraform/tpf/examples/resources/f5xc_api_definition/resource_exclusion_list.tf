# Example: Api Definition - Exclusion List

resource "f5xc_api_definition_uploader" "spec" {
  metadata = {
    name      = "example-api-definition-spec"
    namespace = "shared"
  }
  string_value = "{\"openapi\":\"3.0.0\",\"info\":{\"title\":\"Test API\",\"version\":\"1.0.0\"},\"paths\":{\"/2.0/repositories/{username}\":{\"get\":{\"operationId\":\"getRepositoriesByOwner\",\"parameters\":[{\"name\":\"username\",\"in\":\"path\",\"required\":true,\"schema\":{\"type\":\"string\"}}],\"responses\":{\"200\":{\"description\":\"repos\"}}}},\"/2.0/repositories/{username}/{slug}/pullrequests/{pid}/merge\":{\"post\":{\"operationId\":\"mergePullRequest\",\"parameters\":[{\"name\":\"username\",\"in\":\"path\",\"required\":true,\"schema\":{\"type\":\"string\"}},{\"name\":\"slug\",\"in\":\"path\",\"required\":true,\"schema\":{\"type\":\"string\"}},{\"name\":\"pid\",\"in\":\"path\",\"required\":true,\"schema\":{\"type\":\"string\"}}],\"responses\":{\"204\":{\"description\":\"merged\"}}}}}}"
}

resource "f5xc_api_definition" "example" {
  name      = "example-api-definition"
  namespace = "shared"

  swagger_specs = [f5xc_api_definition_uploader.spec.metadata.url]

  api_inventory_exclusion_list = [
    {
      method = ["GET"]
      path   = "/2.0/repositories/{username}"
    }
  ]

  schema_updates_strategy_choice = {
    strict_schema_origin = true
  }
}
