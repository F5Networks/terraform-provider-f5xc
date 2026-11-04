# YAML-to-JSON Example
# Use this method when developers write the OpenAPI specification in YAML, but you prefer to pass a normalized, validated, and compact JSON string to F5 XC.

resource "f5xc_api_definition_uploader" "yaml_to_json" {
  metadata = {
    name        = "yaml-converted-to-json"
    namespace   = "shared"
    description = "OpenAPI YAML specification normalized to JSON"
  }

  # Parses the local YAML file into a Terraform map and serializes it to JSON
  string_value = jsonencode(yamldecode(file("${path.module}/files/openapi.yaml")))
}
