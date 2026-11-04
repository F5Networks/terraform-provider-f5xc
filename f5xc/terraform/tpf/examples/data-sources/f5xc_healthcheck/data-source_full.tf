# Example: Healthcheck - Full

data "f5xc_healthcheck" "example" {
  name      = f5xc_healthcheck.example.name
  namespace = f5xc_healthcheck.example.namespace
}
