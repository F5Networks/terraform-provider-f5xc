# Static Local File Example (Write-Only, Terraform >= 1.11)
# Use this approach when the OpenAPI specification is committed directly to
# the repository as a static JSON or YAML file.
#
# string_value is write-only and never persisted in Terraform state; content_hash
# (computed automatically when omitted, or explicitly set via filesha256) tracks changes.
#
# Best practice: use ${path.module}/... so Terraform resolves the file path
# relative to the module directory, regardless of the working directory.

resource "f5xc_api_definition_uploader" "from_file" {
  metadata = {
    name        = "api-spec-from-file"
    namespace   = "shared"
    description = "OpenAPI specification loaded directly from local file"
  }

  # Read the file contents as a raw write-only string (JSON or YAML).
  string_value = file("${path.module}/files/openapi.json")
  content_hash = filesha256("${path.module}/files/openapi.json")
}

