# S3 Artifact Example
# Ideal for enterprise pipelines where CI builds publish generated API specs as build artifacts into central object storage.

data "aws_s3_object" "openapi_artifact" {
  bucket = "company-api-spec-artifacts"
  key    = "services/user-service/v1.0.0/openapi.json"
}

resource "f5xc_api_definition_uploader" "from_s3" {
  metadata = {
    name        = "s3-artifact-spec"
    namespace   = "shared"
    description = "OpenAPI specification retrieved from an S3 artifact"
  }

  string_value = data.aws_s3_object.openapi_artifact.body
}
