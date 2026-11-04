# Static Local File Example
# Use this approach when the OpenAPI specification is committed directly to
# the repository as a static JSON or YAML file.
#
# Best practice: use ${path.module}/... so Terraform resolves the file path
# relative to the module directory, regardless of the working directory.

resource "f5xc_api_definition_uploader" "from_file" {
  metadata = {
    name        = "api-spec-from-file"
    namespace   = "shared"
    description = "OpenAPI specification loaded directly from local file"
  }

  # Read the file contents as a raw string. This works with JSON and YAML.
  string_value = file("${path.module}/files/openapi.json")
}

