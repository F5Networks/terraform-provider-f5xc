# Example: Service Policy - Basic

data "f5xc_service_policy" "example" {
  name      = f5xc_service_policy.example.name
  namespace = f5xc_service_policy.example.namespace
}
