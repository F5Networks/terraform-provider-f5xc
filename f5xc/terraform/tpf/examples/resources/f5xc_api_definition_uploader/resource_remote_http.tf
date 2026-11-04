# Remote HTTP Endpoint Example
# Useful when backend services (such as FastAPI, Spring Boot, or Fastify) 
# publish their OpenAPI document at runtime (e.g., /v3/api-docs or openapi.json), or 
# when pulling directly from a raw GitHub/GitLab release artifact.

variable "github_token" {
  type        = string
  description = "GitHub personal access token for private repositories"
  sensitive   = true
  default     = ""
}

data "http" "remote_openapi" {
  url = "https://raw.githubusercontent.com/my-org/…/openapi.json"

  request_headers = {
    Accept        = "application/json"
    Authorization = var.github_token != "" ? "Bearer ${var.github_token}" : ""
  }
}

resource "f5xc_api_definition_uploader" "from_remote_url" {
  metadata = {
    name        = "remote-api-spec"
    namespace   = "shared"
    description = "OpenAPI specification retrieved from a remote endpoint"
  }

  string_value = data.http.remote_openapi.response_body
}
