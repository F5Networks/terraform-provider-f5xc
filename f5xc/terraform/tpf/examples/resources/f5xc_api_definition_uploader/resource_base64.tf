# Base64 Input Example
# Common in automated CI/CD pipelines (such as GitHub Actions, GitLab CI, or 
# external secret managers) where file contents are passed into Terraform via environment variables or 
# variables as base64-encoded strings.

variable "openapi_spec_base64" {
  type        = string
  description = "Base64-encoded OpenAPI specification"
  sensitive   = true
}

resource "f5xc_api_definition_uploader" "from_base64" {
  metadata = {
    name        = "base64-decoded-spec"
    namespace   = "shared"
    description = "OpenAPI specification decoded from Base64 input"
  }

  string_value = base64decode(var.openapi_spec_base64)
}
