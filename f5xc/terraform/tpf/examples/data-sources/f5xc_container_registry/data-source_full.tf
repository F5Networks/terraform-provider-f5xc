# Example: Container Registry - Full

data "f5xc_container_registry" "example" {
  name      = f5xc_container_registry.example.name
  namespace = f5xc_container_registry.example.namespace
}
