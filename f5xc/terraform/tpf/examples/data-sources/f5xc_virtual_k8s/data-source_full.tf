# Example: Virtual K8s - Full

data "f5xc_virtual_k8s" "example" {
  name      = f5xc_virtual_k8s.example.name
  namespace = f5xc_virtual_k8s.example.namespace
}
