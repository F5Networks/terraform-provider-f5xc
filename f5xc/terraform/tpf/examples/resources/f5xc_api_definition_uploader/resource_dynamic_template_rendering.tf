# Dynamic Template Rendering Example
# Use this approach when your OpenAPI specification needs runtime values injected, 
# such as environment names, target domain names, versions, or contact details.

# Template File (templates/openapi.yaml.tftpl)
# yaml
# openapi: 3.0.0
# info:
#   title: "Shop API (${environment})"
#   version: "${api_version}"
# servers:
#   - url: "https://${domain_name}/api/v1"
# paths:
#   /orders:
#     get:
#       summary: "List orders"
#       responses:
#         "200":
#           description: "Successful response"

variable "environment" {
  type        = string
  description = "Target deployment environment"
  default     = "staging"
}

variable "domain_name" {
  type        = string
  description = "Base domain name for the API"
  default     = "api.staging.example.com"
}

resource "f5xc_api_definition_uploader" "templated" {
  metadata = {
    name        = "templated-api-spec"
    namespace   = "shared"
    description = "OpenAPI specification rendered from a template"
  }

  string_value = templatefile("${path.module}/templates/openapi.yaml.tftpl", {
    environment = var.environment
    domain_name = var.domain_name
    api_version = "2.4.0"
  })
}
