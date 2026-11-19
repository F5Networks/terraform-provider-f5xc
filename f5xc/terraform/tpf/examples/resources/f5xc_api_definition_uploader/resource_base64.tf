# Base64 Write-Only Input Example (Terraform >= 1.11)
# Common in automated CI/CD pipelines (such as GitHub Actions, GitLab CI, or
# external secret managers) where file contents are passed into Terraform via environment variables or
# variables as Base64-encoded strings, or read directly with filebase64(...).
# Because bytes_value is write-only, the payload is never stored in Terraform state.

variable "openapi_spec_base64" {
  type        = string
  description = "Base64-encoded OpenAPI specification"
  sensitive   = true
}

resource "f5xc_api_definition_uploader" "from_base64" {
  metadata = {
    name        = "base64-spec"
    namespace   = "shared"
    description = "OpenAPI specification uploaded from Base64 input without storing payload in state"
  }

  bytes_value  = var.openapi_spec_base64
  content_hash = sha256(base64decode(var.openapi_spec_base64))
}
